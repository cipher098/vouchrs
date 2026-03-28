package handler

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gothi/vouchrs/src/delivery/http/response"
	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/internal/domain/port"
	"github.com/gothi/vouchrs/src/pkg/contextkey"
	"github.com/gothi/vouchrs/src/pkg/pagination"
)

type DashboardHandler struct {
	dashboard port.DashboardService
}

func NewDashboardHandler(dashboard port.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboard: dashboard}
}

// MyListings godoc
//
//	@Summary      My listings (seller dashboard)
//	@Description  Returns a paginated list of listings created by the authenticated seller. Sensitive code fields are stripped.
//	@Tags         dashboard
//	@Produce      json
//	@Security     BearerAuth
//	@Param        page  query int false "Page number"
//	@Param        limit query int false "Items per page"
//	@Success      200 {object} response.Response{data=[]entity.Listing}
//	@Failure      401 {object} response.Response
//	@Router       /api/v1/dashboard/listings [get]
func (h *DashboardHandler) MyListings(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(contextkey.UserID).(uuid.UUID)
	if !ok {
		response.Error(w, apperror.ErrUnauthorized)
		return
	}
	p := pagination.FromRequest(r)
	var listings []*entity.Listing
	var meta *pagination.Meta
	var err error
	listings, meta, err = h.dashboard.GetMyListings(r.Context(), userID, p)
	if err != nil {
		response.Error(w, err)
		return
	}
	// Strip sensitive fields
	for _, l := range listings {
		l.CodeEncrypted = ""
		l.CodeHash = ""
	}
	response.Paginated(w, http.StatusOK, listings, *meta)
}

// MyPurchases godoc
//
//	@Summary      My purchases (buyer dashboard)
//	@Description  Returns a paginated list of transactions where the authenticated user is the buyer.
//	@Tags         dashboard
//	@Produce      json
//	@Security     BearerAuth
//	@Param        page  query int false "Page number"
//	@Param        limit query int false "Items per page"
//	@Success      200 {object} response.Response{data=[]entity.Transaction}
//	@Failure      401 {object} response.Response
//	@Router       /api/v1/dashboard/purchases [get]
func (h *DashboardHandler) MyPurchases(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(contextkey.UserID).(uuid.UUID)
	if !ok {
		response.Error(w, apperror.ErrUnauthorized)
		return
	}
	p := pagination.FromRequest(r)
	var txns []*entity.Transaction
	var meta *pagination.Meta
	var err error
	txns, meta, err = h.dashboard.GetMyPurchases(r.Context(), userID, p)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Paginated(w, http.StatusOK, txns, *meta)
}

// MyRequests godoc
//
//	@Summary      My requests (dashboard)
//	@Description  Returns all buy requests and card requests created by the authenticated user.
//	@Tags         dashboard
//	@Produce      json
//	@Security     BearerAuth
//	@Success      200 {object} response.Response{data=port.MyRequestsResult}
//	@Failure      401 {object} response.Response
//	@Router       /api/v1/dashboard/requests [get]
func (h *DashboardHandler) MyRequests(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(contextkey.UserID).(uuid.UUID)
	if !ok {
		response.Error(w, apperror.ErrUnauthorized)
		return
	}
	result, err := h.dashboard.GetMyRequests(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}
