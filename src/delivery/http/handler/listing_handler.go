package handler

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gothi/vouchrs/src/delivery/http/request"
	"github.com/gothi/vouchrs/src/delivery/http/response"
	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/internal/domain/port"
	"github.com/gothi/vouchrs/src/pkg/contextkey"
	"github.com/gothi/vouchrs/src/pkg/pagination"
)

type ListingHandler struct {
	listing port.ListingService
}

func NewListingHandler(listing port.ListingService) *ListingHandler {
	return &ListingHandler{listing: listing}
}

type createListingBody struct {
	BrandID        string   `json:"brand_id"              validate:"required,uuid"  example:"550e8400-e29b-41d4-a716-446655440000"`
	FaceValue      float64  `json:"face_value"            validate:"required,gt=0"  example:"1000"`
	CardCode       string   `json:"card_code"             validate:"required,min=4" example:"AMZN-XXXX-XXXX-XXXX"`
	AcceptPool     bool     `json:"accept_pool"                                     example:"true"`
	CustomDiscount *float64 `json:"custom_discount,omitempty"                       example:"5"`
}

// CreateListing godoc
//
//	@Summary      Create a listing (seller)
//	@Description  Post a gift card for sale. Runs Gate 1 Qwikcilver verification. The card code is AES-256 encrypted at rest and never returned by any API endpoint — it is delivered to the buyer's email only.
//	@Tags         listings
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        body body createListingBody true "Listing details"
//	@Success      201  {object} response.Response{data=entity.Listing}
//	@Failure      400  {object} response.Response "Validation error"
//	@Failure      401  {object} response.Response
//	@Failure      409  {object} response.Response "Duplicate card code (CONFLICT)"
//	@Failure      422  {object} response.Response "Gate 1 failed (VERIFICATION_FAILED) or daily limit (LISTING_LIMIT_REACHED)"
//	@Router       /api/v1/listings [post]
func (h *ListingHandler) Create(w http.ResponseWriter, r *http.Request) {
	sellerID, ok := r.Context().Value(contextkey.UserID).(uuid.UUID)
	if !ok {
		response.Error(w, apperror.ErrUnauthorized)
		return
	}

	var body createListingBody
	if err := request.Decode(r, &body); err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, err.Error()))
		return
	}

	brandID, err := uuid.Parse(body.BrandID)
	if err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, "invalid brand_id"))
		return
	}

	listing, err := h.listing.CreateListing(r.Context(), sellerID, port.CreateListingInput{
		BrandID:        brandID,
		FaceValue:      body.FaceValue,
		CardCode:       body.CardCode,
		AcceptPool:     body.AcceptPool,
		CustomDiscount: body.CustomDiscount,
	})
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusCreated, listing)
}

// GetListing godoc
//
//	@Summary      Get a listing by ID
//	@Description  Returns listing details. The code_encrypted and code_hash fields are always stripped.
//	@Tags         listings
//	@Produce      json
//	@Param        id  path string true "Listing UUID" example("550e8400-e29b-41d4-a716-446655440000")
//	@Success      200 {object} response.Response{data=entity.Listing}
//	@Failure      400 {object} response.Response "Invalid UUID"
//	@Failure      404 {object} response.Response
//	@Router       /api/v1/listings/{id} [get]
func (h *ListingHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, "invalid listing id"))
		return
	}
	var listing *entity.Listing
	listing, err = h.listing.GetListing(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	// Never expose the encrypted code in the API
	listing.CodeEncrypted = ""
	listing.CodeHash = ""
	response.JSON(w, http.StatusOK, listing)
}

// CancelListing godoc
//
//	@Summary      Cancel a listing (seller)
//	@Description  Cancel a LIVE listing. Only the original seller can cancel. Returns 403 if called by any other user.
//	@Tags         listings
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id  path string true "Listing UUID"
//	@Success      200 {object} response.Response{data=map[string]string} "listing cancelled"
//	@Failure      400 {object} response.Response "Invalid UUID"
//	@Failure      401 {object} response.Response
//	@Failure      403 {object} response.Response "Not the listing owner"
//	@Failure      404 {object} response.Response
//	@Router       /api/v1/listings/{id} [delete]
func (h *ListingHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	sellerID, ok := r.Context().Value(contextkey.UserID).(uuid.UUID)
	if !ok {
		response.Error(w, apperror.ErrUnauthorized)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, "invalid listing id"))
		return
	}
	if err := h.listing.CancelListing(r.Context(), sellerID, id); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "listing cancelled"})
}

// GetPoolGroup godoc
//
//	@Summary      Get a pool group by ID
//	@Description  Returns pool group details including buyer price, discount, and active listing count. Use this to show the detail page for a Vouchrs pool listing.
//	@Tags         marketplace
//	@Produce      json
//	@Param        id  path string true "Pool group UUID"
//	@Success      200 {object} response.Response{data=entity.PoolGroup}
//	@Failure      400 {object} response.Response "Invalid UUID"
//	@Failure      404 {object} response.Response
//	@Router       /api/v1/pool-groups/{id} [get]
func (h *ListingHandler) GetPoolGroup(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, "invalid pool group id"))
		return
	}
	pg, err := h.listing.GetPoolGroup(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, pg)
}

// RecommendedPrice godoc
//
//	@Summary      Get recommended listing price
//	@Description  Returns the platform-recommended pricing breakdown for a given brand and face value. Use this before submitting POST /api/v1/listings to show the seller what they will receive.
//	@Tags         listings
//	@Produce      json
//	@Security     BearerAuth
//	@Param        brand_id    query string  true "Brand UUID"
//	@Param        face_value  query number  true "Card face value in INR"
//	@Success      200 {object} response.Response{data=port.RecommendedPriceResult}
//	@Failure      400 {object} response.Response "Missing or invalid params"
//	@Failure      401 {object} response.Response
//	@Failure      404 {object} response.Response "Brand not found"
//	@Router       /api/v1/listings/recommended-price [get]
func (h *ListingHandler) RecommendedPrice(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	brandID, err := uuid.Parse(q.Get("brand_id"))
	if err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, "invalid brand_id"))
		return
	}
	var faceValue float64
	if _, err := fmt.Sscanf(q.Get("face_value"), "%f", &faceValue); err != nil || faceValue <= 0 {
		response.Error(w, apperror.New(apperror.ErrBadRequest, "face_value must be a positive number"))
		return
	}
	result, err := h.listing.GetRecommendedPrice(r.Context(), brandID, faceValue)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

// Marketplace godoc
//
//	@Summary      Browse marketplace
//	@Description  Returns Vouchrs pool groups (aggregated inventory) at the top and individual seller listings below. Pool groups show the best available price for each brand+denomination combination.
//	@Tags         marketplace
//	@Produce      json
//	@Param        brand_id query string false "Filter by brand UUID"
//	@Param        sort_by  query string false "Sort field" Enums(price_asc,price_desc,discount_desc)
//	@Param        page     query int    false "Page number (default 1)"   minimum(1)
//	@Param        limit    query int    false "Items per page (default 20, max 100)" minimum(1) maximum(100)
//	@Success      200 {object} response.Response{data=port.MarketplaceResult}
//	@Failure      500 {object} response.Response
//	@Router       /api/v1/marketplace [get]
func (h *ListingHandler) Marketplace(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := port.MarketplaceFilter{
		SortBy:     q.Get("sort_by"),
		Pagination: pagination.FromRequest(r),
	}

	if brandStr := q.Get("brand_id"); brandStr != "" {
		bid, err := uuid.Parse(brandStr)
		if err == nil {
			f.BrandID = &bid
		}
	}

	result, err := h.listing.GetMarketplace(r.Context(), f)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
