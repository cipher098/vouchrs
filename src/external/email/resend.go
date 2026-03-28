package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/internal/domain/port"
)

const resendSendURL = "https://api.resend.com/emails"

type resendClient struct {
	apiKey     string
	from       string
	httpClient *http.Client
}

type resendPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

// NewResendClient creates an EmailService backed by resend.com.
func NewResendClient(apiKey, from string) port.EmailService {
	return &resendClient{
		apiKey: apiKey,
		from:   from,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *resendClient) send(ctx context.Context, to, subject, html string) error {
	p := resendPayload{
		From:    c.from,
		To:      []string{to},
		Subject: subject,
		HTML:    html,
	}
	body, err := json.Marshal(p)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resendSendURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("resend request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend error %d: %s", resp.StatusCode, b)
	}
	return nil
}

func (c *resendClient) SendOTP(ctx context.Context, email, otp string) error {
	html := fmt.Sprintf(`
		<p>Your CardSwap verification code is:</p>
		<h1 style="letter-spacing:8px;font-size:36px;">%s</h1>
		<p>This code expires in 10 minutes. Do not share it with anyone.</p>
	`, otp)
	return c.send(ctx, email, "Your CardSwap OTP", html)
}

func (c *resendClient) SendCardCode(ctx context.Context, email, brandName string, faceValue float64, code string) error {
	html := fmt.Sprintf(`
		<h2>Your %s Gift Card — ₹%.0f</h2>
		<p>Your gift card code is:</p>
		<div style="background:#f5f5f5;padding:20px;font-family:monospace;font-size:24px;letter-spacing:4px;text-align:center;">
			<strong>%s</strong>
		</div>
		<p><strong>Important:</strong> Redeem this code within 15 minutes on the brand's website or app.</p>
		<p>If you face any issues, contact support@cardswap.in</p>
	`, brandName, faceValue, code)
	return c.send(ctx, email, fmt.Sprintf("Your %s Gift Card Code — CardSwap", brandName), html)
}

func (c *resendClient) SendPurchaseReceipt(ctx context.Context, email string, txn *entity.Transaction) error {
	html := fmt.Sprintf(`
		<h2>Purchase Confirmed</h2>
		<p>Transaction ID: <code>%s</code></p>
		<p>Amount Paid: ₹%.2f</p>
		<p>The gift card code has been sent to your email.</p>
		<p>Thank you for using CardSwap!</p>
	`, txn.ID, txn.BuyerAmount)
	return c.send(ctx, email, "CardSwap Purchase Receipt", html)
}

func (c *resendClient) SendCardRequestUpdate(ctx context.Context, email string, req *entity.CardRequest) error {
	statusMsg := map[entity.CardRequestStatus]string{
		entity.CardRequestStatusUnderReview: "We're working on sourcing your card.",
		entity.CardRequestStatusFulfilled:   "Great news! Your requested card is now available on CardSwap.",
		entity.CardRequestStatusRejected:    fmt.Sprintf("We couldn't source this card: %s", req.AdminNotes),
		entity.CardRequestStatusDeferred:    "We've added this to our roadmap and will notify you when available.",
	}
	msg, ok := statusMsg[req.Status]
	if !ok {
		return nil
	}
	html := fmt.Sprintf(`
		<h2>Update on Your Card Request</h2>
		<p>Brand: <strong>%s</strong></p>
		<p>%s</p>
	`, req.Brand, msg)
	return c.send(ctx, email, "CardSwap Card Request Update", html)
}

func (c *resendClient) SendBuyRequestAlert(ctx context.Context, email string, listing *entity.Listing, brandName string) error {
	html := fmt.Sprintf(`
		<h2>A Card You Wanted Is Now Available!</h2>
		<p>A %s gift card worth ₹%.0f is now listed at ₹%.2f (%.1f%% off).</p>
		<p><a href="https://cardswap.in/listings/%s">Buy it now before it's gone →</a></p>
	`, brandName, listing.FaceValue, listing.BuyerPrice, listing.DiscountPct, listing.ID)
	return c.send(ctx, email, fmt.Sprintf("%s Gift Card Available — CardSwap", brandName), html)
}

func (c *resendClient) SendAdminCardRequestNotification(ctx context.Context, adminEmails []string, req *entity.CardRequest) error {
	html := fmt.Sprintf(`
		<h2>New Card Request</h2>
		<p>Brand: <strong>%s</strong></p>
		<p>Value: ₹%.0f | Urgency: %s</p>
		<p>Request ID: <code>%s</code></p>
		<p>Review in admin panel.</p>
	`, req.Brand, req.DesiredValue, req.Urgency, req.ID)
	for _, email := range adminEmails {
		if err := c.send(ctx, email, "New Card Request — CardSwap Admin", html); err != nil {
			return err
		}
	}
	return nil
}
