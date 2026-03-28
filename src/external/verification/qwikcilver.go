// Package verification provides Qwikcilver-based gift card balance checking
// using headless Chrome (chromedp). Only Amazon India is confirmed working at launch.
package verification

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/gothi/vouchrs/src/internal/domain/port"
)

const (
	qwikcilverURL = "https://amazonbalance.qwikcilver.com"
	resultPass    = "pass"
	resultFail    = "fail"
)

type qwikcilverScraper struct {
	timeoutSec int
	headless   bool
	logger     *slog.Logger
}

// NewQwikcilverVerifier creates a VerificationService backed by Qwikcilver headless scraping.
func NewQwikcilverVerifier(timeoutSec int, headless bool, logger *slog.Logger) port.VerificationService {
	return &qwikcilverScraper{
		timeoutSec: timeoutSec,
		headless:   headless,
		logger:     logger,
	}
}

// Verify checks a gift card code on Qwikcilver.
// brandSlug is used to select the correct verification URL (only "amazon" supported at launch).
func (s *qwikcilverScraper) Verify(ctx context.Context, brandSlug, cardCode string) (*port.VerificationResult, error) {
	verifyURL, err := s.urlForBrand(brandSlug)
	if err != nil {
		return &port.VerificationResult{
			IsValid:    false,
			FailReason: err.Error(),
		}, nil
	}

	opts := chromedp.DefaultExecAllocatorOptions[:]
	if s.headless {
		opts = append(opts, chromedp.Flag("headless", true))
	}
	opts = append(opts,
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	defer allocCancel()

	chromeCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	timeout := time.Duration(s.timeoutSec) * time.Second
	chromeCtx, cancelTimeout := context.WithTimeout(chromeCtx, timeout)
	defer cancelTimeout()

	var balanceText, statusText, rawPage string

	err = chromedp.Run(chromeCtx,
		chromedp.Navigate(verifyURL),
		chromedp.WaitVisible(`input[type="text"]`, chromedp.ByQuery),
		chromedp.SetValue(`input[type="text"]`, strings.TrimSpace(cardCode), chromedp.ByQuery),
		chromedp.Click(`button[type="submit"]`, chromedp.ByQuery),
		chromedp.Sleep(2*time.Second),
		chromedp.InnerHTML(`body`, &rawPage, chromedp.ByQuery),
		chromedp.Text(`.balance, .card-balance, [class*="balance"]`, &balanceText, chromedp.ByQuery, chromedp.AtLeast(0)),
		chromedp.Text(`.status, .card-status, [class*="status"]`, &statusText, chromedp.ByQuery, chromedp.AtLeast(0)),
	)
	if err != nil {
		s.logger.Warn("qwikcilver scrape error", "brand", brandSlug, "error", err)
		return &port.VerificationResult{
			IsValid:    false,
			FailReason: fmt.Sprintf("scraper error: %s", err.Error()),
		}, nil
	}

	responseHash := fmt.Sprintf("%x", sha256.Sum256([]byte(rawPage)))
	balance, failReason := parseBalance(balanceText)

	isValid := balance > 0 && !strings.Contains(strings.ToLower(rawPage), "invalid") &&
		!strings.Contains(strings.ToLower(rawPage), "expired") &&
		!strings.Contains(strings.ToLower(statusText), "claimed")

	if !isValid && failReason == "" {
		failReason = detectFailReason(rawPage, statusText)
	}

	return &port.VerificationResult{
		IsValid:      isValid,
		Balance:      balance,
		Status:       parseStatus(rawPage, statusText),
		FailReason:   failReason,
		ResponseHash: responseHash,
	}, nil
}

func (s *qwikcilverScraper) urlForBrand(brandSlug string) (string, error) {
	switch brandSlug {
	case "amazon", "amazon-india":
		return qwikcilverURL, nil
	default:
		return "", fmt.Errorf("brand %q is not yet supported for verification", brandSlug)
	}
}

func parseBalance(text string) (float64, string) {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "₹", "")
	text = strings.ReplaceAll(text, ",", "")
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, "balance not found on page"
	}
	var balance float64
	if _, err := fmt.Sscanf(text, "%f", &balance); err != nil {
		return 0, fmt.Sprintf("could not parse balance: %q", text)
	}
	return balance, ""
}

func parseStatus(rawPage, statusText string) string {
	lower := strings.ToLower(rawPage + statusText)
	switch {
	case strings.Contains(lower, "expired"):
		return "expired"
	case strings.Contains(lower, "claimed") || strings.Contains(lower, "redeemed"):
		return "claimed"
	case strings.Contains(lower, "invalid"):
		return "invalid"
	default:
		return "active"
	}
}

func detectFailReason(rawPage, statusText string) string {
	lower := strings.ToLower(rawPage + statusText)
	switch {
	case strings.Contains(lower, "expired"):
		return "card has expired"
	case strings.Contains(lower, "claimed") || strings.Contains(lower, "redeemed"):
		return "card has already been redeemed"
	case strings.Contains(lower, "invalid"):
		return "invalid card code"
	case strings.Contains(lower, "zero") || strings.Contains(lower, "0.00"):
		return "card has zero balance"
	default:
		return "verification failed"
	}
}
