// Package phonepe implements the PaymentGateway port for PhonePe.
// To switch payment gateways, create a new package in src/external/payment/
// and implement port.PaymentGateway. Change the injection in cmd/api/main.go.
package phonepe

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gothi/vouchrs/src/internal/domain/port"
)

const (
	uatBaseURL  = "https://api-preprod.phonepe.com/apis/pg-sandbox"
	prodBaseURL = "https://api.phonepe.com/apis/hermes"
)

type phonePeClient struct {
	merchantID string
	saltKey    string
	saltIndex  string
	baseURL    string
	httpClient *http.Client
}

// NewPhonePeGateway creates a PaymentGateway backed by PhonePe.
func NewPhonePeGateway(merchantID, saltKey, saltIndex, env string) port.PaymentGateway {
	base := prodBaseURL
	if env == "UAT" {
		base = uatBaseURL
	}
	return &phonePeClient{
		merchantID: merchantID,
		saltKey:    saltKey,
		saltIndex:  saltIndex,
		baseURL:    base,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

type initiatePayPayload struct {
	MerchantID            string `json:"merchantId"`
	MerchantTransactionID string `json:"merchantTransactionId"`
	MerchantUserID        string `json:"merchantUserId"`
	Amount                int64  `json:"amount"` // paise
	RedirectURL           string `json:"redirectUrl"`
	RedirectMode          string `json:"redirectMode"`
	CallbackURL           string `json:"callbackUrl"`
	PaymentInstrument     struct {
		Type string `json:"type"`
	} `json:"paymentInstrument"`
}

type phonePeResponse struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    struct {
		MerchantID            string `json:"merchantId"`
		MerchantTransactionID string `json:"merchantTransactionId"`
		InstrumentResponse    struct {
			Type         string `json:"type"`
			RedirectInfo struct {
				URL    string `json:"url"`
				Method string `json:"method"`
			} `json:"redirectInfo"`
		} `json:"instrumentResponse"`
	} `json:"data"`
}

func (c *phonePeClient) CreateOrder(ctx context.Context, input port.PaymentOrderInput) (*port.PaymentOrderResult, error) {
	p := initiatePayPayload{
		MerchantID:            c.merchantID,
		MerchantTransactionID: input.MerchantTransactionID,
		MerchantUserID:        input.UserID.String(),
		Amount:                int64(input.Amount * 100), // convert INR to paise
		RedirectURL:           input.RedirectURL,
		RedirectMode:          "REDIRECT",
		CallbackURL:           input.CallbackURL,
	}
	p.PaymentInstrument.Type = "PAY_PAGE"

	payloadBytes, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString(payloadBytes)

	// PhonePe checksum: SHA256(base64Payload + "/pg/v1/pay" + saltKey) + "###" + saltIndex
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(encoded+"/pg/v1/pay"+c.saltKey)))
	checksum := hash + "###" + c.saltIndex

	reqBody, _ := json.Marshal(map[string]string{"request": encoded})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/pg/v1/pay", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-VERIFY", checksum)
	req.Header.Set("X-MERCHANT-ID", c.merchantID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("phonepe create order: %w", err)
	}
	defer resp.Body.Close()

	var result phonePeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode phonepe response: %w", err)
	}
	if !result.Success {
		return nil, fmt.Errorf("phonepe error %s: %s", result.Code, result.Message)
	}

	return &port.PaymentOrderResult{
		MerchantTransactionID: result.Data.MerchantTransactionID,
		PaymentURL:            result.Data.InstrumentResponse.RedirectInfo.URL,
	}, nil
}

func (c *phonePeClient) VerifyWebhook(_ context.Context, body []byte, headers map[string]string) (*port.PaymentWebhookEvent, error) {
	xVerify := headers["X-VERIFY"]
	if xVerify == "" {
		return nil, fmt.Errorf("missing X-VERIFY header")
	}

	// Decode the base64 response payload
	var wrapper struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, fmt.Errorf("parse webhook body: %w", err)
	}

	// Verify checksum
	expected := fmt.Sprintf("%x###%s", sha256.Sum256([]byte(wrapper.Response+c.saltKey)), c.saltIndex)
	if xVerify != expected {
		return nil, fmt.Errorf("webhook signature mismatch")
	}

	decoded, err := base64.StdEncoding.DecodeString(wrapper.Response)
	if err != nil {
		return nil, fmt.Errorf("decode webhook payload: %w", err)
	}

	var event struct {
		Success bool   `json:"success"`
		Code    string `json:"code"`
		Data    struct {
			MerchantTransactionID string  `json:"merchantTransactionId"`
			Amount                float64 `json:"amount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(decoded, &event); err != nil {
		return nil, fmt.Errorf("parse webhook event: %w", err)
	}

	status := "FAILURE"
	if event.Success && event.Code == "PAYMENT_SUCCESS" {
		status = "SUCCESS"
	}

	return &port.PaymentWebhookEvent{
		MerchantTransactionID: event.Data.MerchantTransactionID,
		Status:                status,
		Amount:                event.Data.Amount / 100, // paise to INR
	}, nil
}

func (c *phonePeClient) GetPaymentStatus(ctx context.Context, merchantTransactionID string) (*port.PaymentWebhookEvent, error) {
	path := fmt.Sprintf("/pg/v1/status/%s/%s", c.merchantID, merchantTransactionID)
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(path+c.saltKey)))
	checksum := hash + "###" + c.saltIndex

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-VERIFY", checksum)
	req.Header.Set("X-MERCHANT-ID", c.merchantID)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("phonepe status check: %w", err)
	}
	defer resp.Body.Close()

	b, _ := io.ReadAll(resp.Body)
	var result struct {
		Success bool   `json:"success"`
		Code    string `json:"code"`
		Data    struct {
			MerchantTransactionID string  `json:"merchantTransactionId"`
			Amount                float64 `json:"amount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(b, &result); err != nil {
		return nil, err
	}

	status := "FAILURE"
	if result.Success && result.Code == "PAYMENT_SUCCESS" {
		status = "SUCCESS"
	} else if result.Code == "PAYMENT_PENDING" {
		status = "PENDING"
	}

	return &port.PaymentWebhookEvent{
		MerchantTransactionID: result.Data.MerchantTransactionID,
		Status:                status,
		Amount:                result.Data.Amount / 100,
	}, nil
}

// merchantTransactionID generates a unique ID for a PhonePe order.
func MerchantTransactionID() string {
	return "CS" + uuid.New().String()[:18]
}
