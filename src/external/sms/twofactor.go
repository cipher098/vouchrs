package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gothi/vouchrs/src/internal/domain/port"
)

const twoFactorBaseURL = "https://2factor.in/API/V1"

type twoFactorClient struct {
	apiKey     string
	httpClient *http.Client
}

type twoFactorResponse struct {
	Status  string `json:"Status"`
	Details string `json:"Details"`
}

// NewTwoFactor creates an SMSService backed by 2factor.in.
func NewTwoFactor(apiKey string) port.SMSService {
	return &twoFactorClient{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// SendOTP calls the AUTOGEN endpoint and returns the session ID.
func (c *twoFactorClient) SendOTP(ctx context.Context, phone string) (string, error) {
	url := fmt.Sprintf("%s/%s/SMS/%s/AUTOGEN", twoFactorBaseURL, c.apiKey, phone)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build 2factor request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("2factor request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result twoFactorResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("parse 2factor response: %w", err)
	}
	if result.Status != "Success" {
		return "", fmt.Errorf("2factor error: %s", result.Details)
	}
	return result.Details, nil // Details is the session ID on success
}

// VerifyOTP calls the VERIFY endpoint and returns nil on match.
func (c *twoFactorClient) VerifyOTP(ctx context.Context, sessionID, otp string) error {
	url := fmt.Sprintf("%s/%s/SMS/VERIFY/%s/%s", twoFactorBaseURL, c.apiKey, sessionID, otp)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build 2factor verify request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("2factor verify request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result twoFactorResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parse 2factor verify response: %w", err)
	}
	if result.Status != "Success" {
		return fmt.Errorf("2factor verify error: %s", result.Details)
	}
	return nil
}
