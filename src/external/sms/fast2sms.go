package sms

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gothi/vouchrs/src/internal/domain/port"
)

const fast2smsURL = "https://www.fast2sms.com/dev/bulkV2"

type fast2smsClient struct {
	apiKey     string
	httpClient *http.Client
}

type fast2smsResponse struct {
	Return  bool     `json:"return"`
	Message []string `json:"message"`
}

// NewFast2SMS creates an SMS service backed by fast2sms.
func NewFast2SMS(apiKey string) port.SMSService {
	return &fast2smsClient{
		apiKey: apiKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *fast2smsClient) SendOTP(ctx context.Context, phone, otp string) error {
	form := url.Values{}
	form.Set("variables_values", otp)
	form.Set("route", "otp")
	form.Set("numbers", phone)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fast2smsURL, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("build fast2sms request: %w", err)
	}
	req.Header.Set("authorization", c.apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fast2sms request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result fast2smsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parse fast2sms response: %w", err)
	}
	if !result.Return {
		return fmt.Errorf("fast2sms error: %v", result.Message)
	}
	return nil
}
