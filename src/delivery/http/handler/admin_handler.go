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
	"github.com/gothi/vouchrs/src/pkg/pagination"
)

type AdminHandler struct {
	admin port.AdminService
}

func NewAdminHandler(admin port.AdminService) *AdminHandler {
	return &AdminHandler{admin: admin}
}

type reviewCardRequestBody struct {
	Action string `json:"action" validate:"required,oneof=approve reject defer" example:"approve"`
	Notes  string `json:"notes"  example:"Sourced from partner inventory"`
}

// ListCardRequests godoc
//
//	@Summary      List pending card requests (admin)
//	@Description  Returns all card requests submitted by buyers that need admin review.
//	@Tags         admin
//	@Produce      json
//	@Security     BearerAuth
//	@Success      200 {object} response.Response{data=[]entity.CardRequest}
//	@Failure      401 {object} response.Response
//	@Failure      403 {object} response.Response "Admin role required"
//	@Router       /api/v1/admin/card-requests [get]
func (h *AdminHandler) ListCardRequests(w http.ResponseWriter, r *http.Request) {
	var reqs []*entity.CardRequest
	var err error
	reqs, err = h.admin.ListCardRequests(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, reqs)
}

// ReviewCardRequest godoc
//
//	@Summary      Review a card request (admin)
//	@Description  Approve, reject, or defer a buyer's card request. Sends an email notification to the buyer.
//	@Tags         admin
//	@Accept       json
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id   path string             true "Card request UUID"
//	@Param        body body reviewCardRequestBody true "Review decision"
//	@Success      200  {object} response.Response{data=map[string]string} "updated"
//	@Failure      400  {object} response.Response
//	@Failure      401  {object} response.Response
//	@Failure      403  {object} response.Response "Admin role required"
//	@Failure      404  {object} response.Response
//	@Router       /api/v1/admin/card-requests/{id} [patch]
func (h *AdminHandler) ReviewCardRequest(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, "invalid id"))
		return
	}
	var body reviewCardRequestBody
	if err := request.Decode(r, &body); err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, err.Error()))
		return
	}
	if err := h.admin.ReviewCardRequest(r.Context(), id, port.AdminCardRequestAction(body.Action), body.Notes); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "updated"})
}

// ListFraudFlags godoc
//
//	@Summary      List unresolved fraud flags (admin)
//	@Description  Returns all open fraud flags raised during Gate 2 verification failures or manual reviews.
//	@Tags         admin
//	@Produce      json
//	@Security     BearerAuth
//	@Success      200 {object} response.Response{data=[]entity.FraudFlag}
//	@Failure      401 {object} response.Response
//	@Failure      403 {object} response.Response "Admin role required"
//	@Router       /api/v1/admin/fraud-flags [get]
func (h *AdminHandler) ListFraudFlags(w http.ResponseWriter, r *http.Request) {
	var flags []*entity.FraudFlag
	var err error
	flags, err = h.admin.ListFraudFlags(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, flags)
}

// ResolveFraudFlag godoc
//
//	@Summary      Resolve a fraud flag (admin)
//	@Description  Mark a fraud flag as reviewed and resolved.
//	@Tags         admin
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id  path string true "Fraud flag UUID"
//	@Success      200 {object} response.Response{data=map[string]string} "resolved"
//	@Failure      400 {object} response.Response "Invalid UUID"
//	@Failure      401 {object} response.Response
//	@Failure      403 {object} response.Response "Admin role required"
//	@Failure      404 {object} response.Response
//	@Router       /api/v1/admin/fraud-flags/{id}/resolve [patch]
func (h *AdminHandler) ResolveFraudFlag(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, "invalid id"))
		return
	}
	if err := h.admin.ResolveFraudFlag(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "resolved"})
}

// BanUser godoc
//
//	@Summary      Ban a user (admin)
//	@Description  Permanently ban a user. Banned users cannot create listings or initiate purchases.
//	@Tags         admin
//	@Produce      json
//	@Security     BearerAuth
//	@Param        id  path string true "User UUID"
//	@Success      200 {object} response.Response{data=map[string]string} "user banned"
//	@Failure      400 {object} response.Response "Invalid UUID"
//	@Failure      401 {object} response.Response
//	@Failure      403 {object} response.Response "Admin role required"
//	@Failure      404 {object} response.Response
//	@Router       /api/v1/admin/users/{id}/ban [patch]
func (h *AdminHandler) BanUser(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, "invalid id"))
		return
	}
	if err := h.admin.BanUser(r.Context(), id); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "user banned"})
}

// Stats godoc
//
//	@Summary      Platform statistics (admin)
//	@Description  Returns aggregated platform metrics: total users, live listings, GMV, pending card requests, and open fraud flags.
//	@Tags         admin
//	@Produce      json
//	@Security     BearerAuth
//	@Success      200 {object} response.Response{data=port.AdminStats}
//	@Failure      401 {object} response.Response
//	@Failure      403 {object} response.Response "Admin role required"
//	@Router       /api/v1/admin/stats [get]
func (h *AdminHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.admin.GetStats(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, stats)
}

// ListAllListings godoc
//
//	@Summary      List all listings (admin)
//	@Description  Paginated list of every listing on the platform, regardless of status.
//	@Tags         admin
//	@Produce      json
//	@Security     BearerAuth
//	@Param        page  query int false "Page number"
//	@Param        limit query int false "Items per page"
//	@Success      200 {object} response.Response{data=[]entity.Listing}
//	@Failure      401 {object} response.Response
//	@Failure      403 {object} response.Response "Admin role required"
//	@Router       /api/v1/admin/listings [get]
func (h *AdminHandler) ListListings(w http.ResponseWriter, r *http.Request) {
	p := pagination.FromRequest(r)
	var listings []*entity.Listing
	var meta *pagination.Meta
	var err error
	listings, meta, err = h.admin.ListAllListings(r.Context(), p)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Paginated(w, http.StatusOK, listings, *meta)
}

// ListAllTransactions godoc
//
//	@Summary      List all transactions (admin)
//	@Description  Paginated list of every transaction on the platform.
//	@Tags         admin
//	@Produce      json
//	@Security     BearerAuth
//	@Param        page  query int false "Page number"
//	@Param        limit query int false "Items per page"
//	@Success      200 {object} response.Response{data=[]entity.Transaction}
//	@Failure      401 {object} response.Response
//	@Failure      403 {object} response.Response "Admin role required"
//	@Router       /api/v1/admin/transactions [get]
func (h *AdminHandler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	p := pagination.FromRequest(r)
	var txns []*entity.Transaction
	var meta *pagination.Meta
	var err error
	txns, meta, err = h.admin.ListAllTransactions(r.Context(), p)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Paginated(w, http.StatusOK, txns, *meta)
}
