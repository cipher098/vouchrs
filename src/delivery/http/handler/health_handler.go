package handler

import (
	"net/http"

	"github.com/gothi/vouchrs/src/delivery/http/response"
)

// GET /health
func Health(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
