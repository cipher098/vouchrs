package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gothi/vouchrs/src/internal/domain/port"
)

const twoFactorBaseURL = "https://2factor.in/API/V1"

type twoFactorClient struct {
	apiKey       string
	templateName string
	httpClient   *http.Client
}

type twoFactorResponse struct {
	Status  string `json:"Status"`
	Details string `json:"Details"`
}

// NewTwoFactor creates an SMSService backed by 2factor.in.
// templateName must match the template created on your 2factor.in dashboard,
// e.g. "VOUCHRS_OTP". The template body should contain {otp} as the placeholder.
func NewTwoFactor(apiKey, templateName string) port.SMSService {
	return &twoFactorClient{
		apiKey:       apiKey,
		templateName: templateName,
		httpClient:   &http.Client{Timeout: 10 * time.Second},
	}
}

// SendOTP delivers otp to phone using the configured SMS template.
// URL: GET /API/V1/{api_key}/SMS/{phone}/{otp}/{template_name}
func (c *twoFactorClient) SendOTP(ctx context.Context, phone, otp string) error {
	url := fmt.Sprintf("%s/%s/SMS/%s/%s/%s",
		twoFactorBaseURL, c.apiKey, normalizePhone(phone), otp, c.templateName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build 2factor request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("2factor request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result twoFactorResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parse 2factor response: %w", err)
	}
	if result.Status != "Success" {
		return fmt.Errorf("2factor error: %s", result.Details)
	}
	return nil
}

// normalizePhone strips the leading '+' so 2factor.in receives e.g. "919876543210".
func normalizePhone(phone string) string {
	return strings.TrimPrefix(phone, "+")
}
