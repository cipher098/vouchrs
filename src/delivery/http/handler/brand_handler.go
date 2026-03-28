package handler

import (
	"net/http"

	"github.com/gothi/vouchrs/src/delivery/http/response"
	"github.com/gothi/vouchrs/src/internal/domain/entity"
	"github.com/gothi/vouchrs/src/internal/domain/port"
)

type BrandHandler struct {
	brands port.BrandRepository
}

func NewBrandHandler(brands port.BrandRepository) *BrandHandler {
	return &BrandHandler{brands: brands}
}

// ListBrands godoc
//
//	@Summary      List active brands
//	@Description  Returns all active brands with their current live listing count. Used by the frontend for the brand grid and filter chips.
//	@Tags         brands
//	@Produce      json
//	@Success      200 {object} response.Response{data=[]entity.Brand}
//	@Failure      500 {object} response.Response
//	@Router       /api/v1/brands [get]
func (h *BrandHandler) ListBrands(w http.ResponseWriter, r *http.Request) {
	brands, err := h.brands.ListWithCount(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	if brands == nil {
		brands = []*entity.Brand{}
	}
	response.JSON(w, http.StatusOK, brands)
}
