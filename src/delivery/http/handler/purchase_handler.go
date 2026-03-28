package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gothi/vouchrs/src/delivery/http/response"
	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/internal/domain/port"
	"github.com/gothi/vouchrs/src/pkg/contextkey"
)

type PurchaseHandler struct {
	purchase port.PurchaseService
	payment  port.PaymentGateway
}

func NewPurchaseHandler(purchase port.PurchaseService, payment port.PaymentGateway) *PurchaseHandler {
	return &PurchaseHandler{purchase: purchase, payment: payment}
}

// --- doc types ---

type initiateBuyBody struct {
	ReturnURL string `json:"return_url" example:"https://vouchrs.in/purchase/confirm?txn_id=123"`
}

type initiateBuyResponse struct {
	TransactionID string  `json:"transaction_id"  example:"123e4567-e89b-12d3-a456-426614174000"`
	PaymentURL    string  `json:"payment_url"     example:"https://api-preprod.phonepe.com/apis/pg-sandbox/..."`
	Amount        float64 `json:"amount"          example:"910"`
	LockExpiresAt string  `json:"lock_expires_at" example:"2024-01-01T10:10:00Z"`
	ReturnURL     string  `json:"return_url"      example:"https://vouchrs.in/purchase/confirm?txn_id=123"`
}

// InitiateBuy godoc
//
//	@Summary      Initiate purchase (buyer)
//	@Description  Runs Gate 2 re-verification, atomically locks the listing for 10 minutes, creates a pending transaction, and returns a PhonePe payment URL. The buyer must complete payment before lock_expires_at or the listing is released. **The card code is never returned here** — it is sent to the buyer's email after payment succeeds.
//	@Tags         purchase
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id   path string          true "Listing UUID to purchase"
//	@Param        body body initiateBuyBody false "Optional return URL for PhonePe redirect"
//	@Success      200 {object} response.Response{data=initiateBuyResponse}
//	@Failure      400 {object} response.Response "Invalid UUID"
//	@Failure      401 {object} response.Response
//	@Failure      403 {object} response.Response "Buyer is banned or trying to buy own listing"
//	@Failure      409 {object} response.Response "Listing locked or not available (LISTING_LOCKED / LISTING_NOT_AVAILABLE)"
//	@Failure      422 {object} response.Response "Gate 2 failed — card was tampered (CARD_TAMPERED)"
//	@Router       /api/v1/listings/{id}/buy [post]
func (h *PurchaseHandler) InitiateBuy(w http.ResponseWriter, r *http.Request) {
	buyerID, ok := r.Context().Value(contextkey.UserID).(uuid.UUID)
	if !ok {
		response.Error(w, apperror.ErrUnauthorized)
		return
	}
	listingID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, "invalid listing id"))
		return
	}

	var body initiateBuyBody
	// body is optional — ignore decode errors
	_ = json.NewDecoder(r.Body).Decode(&body)

	result, err := h.purchase.InitiateBuy(r.Context(), buyerID, listingID)
	if err != nil {
		response.Error(w, err)
		return
	}

	returnURL := body.ReturnURL
	if returnURL == "" {
		returnURL = result.ReturnURL
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"transaction_id":  result.Transaction.ID,
		"payment_url":     result.PaymentURL,
		"amount":          result.Transaction.BuyerAmount,
		"lock_expires_at": result.LockExpiresAt,
		"return_url":      returnURL,
	})
}

// InitiateBuyFromPool godoc
//
//	@Summary      Initiate purchase from pool group (buyer)
//	@Description  Resolves the oldest LIVE listing in the pool (FIFO) and initiates purchase. Identical flow to InitiateBuy once a listing is selected.
//	@Tags         purchase
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id   path string          true "Pool group UUID"
//	@Param        body body initiateBuyBody false "Optional return URL for PhonePe redirect"
//	@Success      200 {object} response.Response{data=initiateBuyResponse}
//	@Failure      400 {object} response.Response "Invalid UUID"
//	@Failure      401 {object} response.Response
//	@Failure      403 {object} response.Response "Buyer is banned"
//	@Failure      404 {object} response.Response "Pool is empty"
//	@Failure      409 {object} response.Response "Listing locked (LISTING_LOCKED)"
//	@Failure      422 {object} response.Response "Gate 2 failed (CARD_TAMPERED)"
//	@Router       /api/v1/pool-groups/{id}/buy [post]
func (h *PurchaseHandler) InitiateBuyFromPool(w http.ResponseWriter, r *http.Request) {
	buyerID, ok := r.Context().Value(contextkey.UserID).(uuid.UUID)
	if !ok {
		response.Error(w, apperror.ErrUnauthorized)
		return
	}
	poolGroupID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, "invalid pool group id"))
		return
	}

	var body initiateBuyBody
	_ = json.NewDecoder(r.Body).Decode(&body)

	result, err := h.purchase.InitiateBuyFromPool(r.Context(), buyerID, poolGroupID)
	if err != nil {
		response.Error(w, err)
		return
	}

	returnURL := body.ReturnURL
	if returnURL == "" {
		returnURL = result.ReturnURL
	}

	response.JSON(w, http.StatusOK, map[string]interface{}{
		"transaction_id":  result.Transaction.ID,
		"payment_url":     result.PaymentURL,
		"amount":          result.Transaction.BuyerAmount,
		"lock_expires_at": result.LockExpiresAt,
		"return_url":      returnURL,
	})
}

// GetTransaction godoc
//
//	@Summary      Get transaction details
//	@Description  Returns a transaction. Only the buyer or seller of that transaction can access it.
//	@Tags         purchase
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id  path string true "Transaction UUID"
//	@Success      200 {object} response.Response{data=entity.Transaction}
//	@Failure      400 {object} response.Response "Invalid UUID"
//	@Failure      401 {object} response.Response
//	@Failure      403 {object} response.Response "Not a party to this transaction"
//	@Failure      404 {object} response.Response
//	@Router       /api/v1/transactions/{id} [get]
func (h *PurchaseHandler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(contextkey.UserID).(uuid.UUID)
	if !ok {
		response.Error(w, apperror.ErrUnauthorized)
		return
	}
	txnID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, "invalid transaction id"))
		return
	}
	var txn *entity.Transaction
	txn, err = h.purchase.GetTransaction(r.Context(), userID, txnID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, txn)
}

// ConfirmRedemption godoc
//
//	@Summary      Confirm card redemption (buyer)
//	@Description  Called by the buyer after they have successfully used the gift card. Marks the transaction as completed and triggers the seller payout.
//	@Tags         purchase
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id  path string true "Transaction UUID"
//	@Success      200 {object} response.Response{data=map[string]string} "redemption confirmed"
//	@Failure      400 {object} response.Response "Invalid UUID"
//	@Failure      401 {object} response.Response
//	@Failure      403 {object} response.Response "Not the buyer of this transaction"
//	@Failure      404 {object} response.Response
//	@Router       /api/v1/transactions/{id}/confirm [post]
func (h *PurchaseHandler) ConfirmRedemption(w http.ResponseWriter, r *http.Request) {
	buyerID, ok := r.Context().Value(contextkey.UserID).(uuid.UUID)
	if !ok {
		response.Error(w, apperror.ErrUnauthorized)
		return
	}
	txnID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, "invalid transaction id"))
		return
	}
	if err := h.purchase.ConfirmRedemption(r.Context(), buyerID, txnID); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "redemption confirmed"})
}

// PhonePeWebhook godoc
//
//	@Summary      PhonePe payment webhook (internal)
//	@Description  Receives payment SUCCESS/FAILURE events from PhonePe. Verified via X-VERIFY checksum header. On success: marks listing sold, emails card code to buyer (never in response body), queues payout. On failure: unlocks listing.
//	@Tags         webhooks
//	@Accept       json
//	@Produce      json
//	@Param        X-VERIFY header string true "PhonePe SHA256 checksum"
//	@Success      200 {object} response.Response{data=map[string]string} "acknowledged"
//	@Failure      400 {object} response.Response "Invalid signature or body"
//	@Router       /api/v1/webhooks/phonepe [post]
func (h *PurchaseHandler) PhonePeWebhook(w http.ResponseWriter, r *http.Request) {
	body := make([]byte, r.ContentLength)
	if _, err := r.Body.Read(body); err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, "read body"))
		return
	}

	headers := map[string]string{
		"X-VERIFY": r.Header.Get("X-VERIFY"),
	}

	event, err := h.payment.VerifyWebhook(r.Context(), body, headers)
	if err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, "invalid webhook signature"))
		return
	}

	switch event.Status {
	case "SUCCESS":
		if err := h.purchase.HandlePaymentSuccess(r.Context(), event.MerchantTransactionID); err != nil {
			response.Error(w, err)
			return
		}
	case "FAILURE":
		if err := h.purchase.HandlePaymentFailure(r.Context(), event.MerchantTransactionID); err != nil {
			response.Error(w, err)
			return
		}
	}

	// PhonePe expects 200 OK to acknowledge receipt
	response.JSON(w, http.StatusOK, map[string]string{"status": "acknowledged"})
}
