package payout

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gothi/vouchrs/src/internal/domain/port"
)

const razorpayPayoutsURL = "https://api.razorpay.com/v1/payouts"

type razorpayClient struct {
	keyID         string
	keySecret     string
	accountNumber string
	httpClient    *http.Client
}

// NewRazorpayPayout creates a PayoutService backed by Razorpay.
func NewRazorpayPayout(keyID, keySecret, accountNumber string) port.PayoutService {
	return &razorpayClient{
		keyID:         keyID,
		keySecret:     keySecret,
		accountNumber: accountNumber,
		httpClient:    &http.Client{Timeout: 15 * time.Second},
	}
}

type razorpayPayoutPayload struct {
	AccountNumber string `json:"account_number"`
	FundAccount   struct {
		AccountType string `json:"account_type"`
		VPA         struct {
			Address string `json:"address"`
		} `json:"vpa"`
		Contact struct {
			Name string `json:"name"`
		} `json:"contact"`
	} `json:"fund_account"`
	Amount      int64  `json:"amount"` // paise
	Currency    string `json:"currency"`
	Mode        string `json:"mode"`
	Purpose     string `json:"purpose"`
	Queue       bool   `json:"queue_if_low_balance"`
	ReferenceID string `json:"reference_id"`
	Narration   string `json:"narration"`
}

type razorpayPayoutResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Error  *struct {
		Description string `json:"description"`
	} `json:"error"`
}

func (c *razorpayClient) CreatePayout(ctx context.Context, input port.CreatePayoutInput) (*port.PayoutResult, error) {
	p := razorpayPayoutPayload{
		AccountNumber: c.accountNumber,
		Amount:        int64(input.Amount * 100),
		Currency:      "INR",
		Mode:          "UPI",
		Purpose:       input.Purpose,
		Queue:         true,
		ReferenceID:   input.ReferenceID,
		Narration:     input.Narration,
	}
	p.FundAccount.AccountType = "vpa"
	p.FundAccount.VPA.Address = input.UPIID
	p.FundAccount.Contact.Name = "CardSwap Seller"

	body, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, razorpayPayoutsURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.keyID, c.keySecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("razorpay payout: %w", err)
	}
	defer resp.Body.Close()

	var result razorpayPayoutResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, fmt.Errorf("razorpay error: %s", result.Error.Description)
	}

	return &port.PayoutResult{
		PayoutID: result.ID,
		Status:   result.Status,
	}, nil
}

func (c *razorpayClient) GetPayoutStatus(ctx context.Context, payoutID string) (*port.PayoutResult, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		razorpayPayoutsURL+"/"+payoutID, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.keyID, c.keySecret)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result razorpayPayoutResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &port.PayoutResult{
		PayoutID: result.ID,
		Status:   result.Status,
	}, nil
}
