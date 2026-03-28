package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gothi/vouchrs/src/delivery/http/request"
	"github.com/gothi/vouchrs/src/delivery/http/response"
	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/internal/domain/port"
	"github.com/gothi/vouchrs/src/pkg/contextkey"
)

type RequestHandler struct {
	requests port.RequestService
}

func NewRequestHandler(requests port.RequestService) *RequestHandler {
	return &RequestHandler{requests: requests}
}

type createBuyRequestBody struct {
	BrandID  string  `json:"brand_id"   validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	MinValue float64 `json:"min_value"  validate:"required,gt=0" example:"500"`
	MaxValue float64 `json:"max_value"  validate:"required,gt=0" example:"2000"`
	MaxPrice float64 `json:"max_price"  validate:"required,gt=0" example:"900"`
}

type createCardRequestBody struct {
	Brand        string  `json:"brand"          validate:"required"                                               example:"Amazon India"`
	DesiredValue float64 `json:"desired_value"  validate:"required,gt=0"                                          example:"500"`
	Urgency      string  `json:"urgency"        validate:"required,oneof=flexible within_24h within_1_week"       example:"within_24h"`
}

// CreateBuyRequest godoc
//
//	@Summary      Create a buy request
//	@Description  Register interest in buying a specific brand + value range. CardSwap will notify you by email when a matching listing goes live.
//	@Tags         requests
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        body body createBuyRequestBody true "Buy request criteria"
//	@Success      201  {object} response.Response{data=entity.BuyRequest}
//	@Failure      400  {object} response.Response
//	@Failure      401  {object} response.Response
//	@Router       /api/v1/buy-requests [post]
func (h *RequestHandler) CreateBuyRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(contextkey.UserID).(uuid.UUID)
	if !ok {
		response.Error(w, apperror.ErrUnauthorized)
		return
	}
	var body createBuyRequestBody
	if err := request.Decode(r, &body); err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, err.Error()))
		return
	}
	brandID, err := uuid.Parse(body.BrandID)
	if err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, "invalid brand_id"))
		return
	}
	req, err := h.requests.CreateBuyRequest(r.Context(), userID, port.CreateBuyRequestInput{
		BrandID:  brandID,
		MinValue: body.MinValue,
		MaxValue: body.MaxValue,
		MaxPrice: body.MaxPrice,
	})
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, req)
}

// ListMyBuyRequests godoc
//
//	@Summary      List my buy requests
//	@Description  Returns all active buy requests created by the authenticated user.
//	@Tags         requests
//	@Produce      json
//	@Security     BearerAuth
//	@Success      200  {object} response.Response{data=[]entity.BuyRequest}
//	@Failure      401  {object} response.Response
//	@Router       /api/v1/buy-requests [get]
func (h *RequestHandler) ListMyBuyRequests(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(contextkey.UserID).(uuid.UUID)
	if !ok {
		response.Error(w, apperror.ErrUnauthorized)
		return
	}
	reqs, err := h.requests.ListMyBuyRequests(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, reqs)
}

// DeleteBuyRequest godoc
//
//	@Summary      Delete a buy request
//	@Description  Remove a buy request. Only the owner can delete it.
//	@Tags         requests
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id  path string true "Buy request UUID"
//	@Success      200  {object} response.Response{data=map[string]string} "deleted"
//	@Failure      400  {object} response.Response "Invalid UUID"
//	@Failure      401  {object} response.Response
//	@Failure      403  {object} response.Response "Not the owner"
//	@Failure      404  {object} response.Response
//	@Router       /api/v1/buy-requests/{id} [delete]
func (h *RequestHandler) DeleteBuyRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(contextkey.UserID).(uuid.UUID)
	if !ok {
		response.Error(w, apperror.ErrUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, "invalid id"))
		return
	}
	if err := h.requests.DeleteBuyRequest(r.Context(), userID, id); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "deleted"})
}

// CreateCardRequest godoc
//
//	@Summary      Create a card request (admin queue)
//	@Description  Ask CardSwap to source a specific gift card on your behalf. Admin reviews and fulfils these requests manually. Urgency affects prioritisation.
//	@Tags         requests
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        body body createCardRequestBody true "Card request details"
//	@Success      201  {object} response.Response{data=entity.CardRequest}
//	@Failure      400  {object} response.Response
//	@Failure      401  {object} response.Response
//	@Router       /api/v1/card-requests [post]
func (h *RequestHandler) CreateCardRequest(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(contextkey.UserID).(uuid.UUID)
	if !ok {
		response.Error(w, apperror.ErrUnauthorized)
		return
	}
	var body createCardRequestBody
	if err := request.Decode(r, &body); err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, err.Error()))
		return
	}
	req, err := h.requests.CreateCardRequest(r.Context(), userID, port.CreateCardRequestInput{
		Brand:        body.Brand,
		DesiredValue: body.DesiredValue,
		Urgency:      entity.CardRequestUrgency(body.Urgency),
	})
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, req)
}
