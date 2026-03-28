package handler

import (
	"net/http"

	"github.com/gothi/vouchrs/src/delivery/http/request"
	"github.com/gothi/vouchrs/src/delivery/http/response"
	"github.com/gothi/vouchrs/src/internal/domain/apperror"
	"github.com/gothi/vouchrs/src/internal/domain/port"
	"github.com/gothi/vouchrs/src/pkg/contextkey"
)

type AuthHandler struct {
	auth port.AuthService
}

func NewAuthHandler(auth port.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

// --- request / response doc types ---

type requestOTPBody struct {
	Contact string `json:"contact" validate:"required" example:"+919999999999"`
}

type verifyOTPBody struct {
	Contact string `json:"contact" validate:"required" example:"+919999999999"`
	OTP     string `json:"otp" validate:"required,len=6" example:"123456"`
}

type refreshTokenBody struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type tokenPairDoc struct {
	AccessToken  string `json:"access_token"  example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

type authUserDoc struct {
	ID         string `json:"id"          example:"123e4567-e89b-12d3-a456-426614174000"`
	Phone      string `json:"phone"       example:"+919999999999"`
	Email      string `json:"email"       example:"user@example.com"`
	FullName   string `json:"full_name"   example:"Rahul Sharma"`
	Role       string `json:"role"        example:"buyer"`
	IsVerified bool   `json:"is_verified" example:"true"`
}

type verifyOTPResponse struct {
	Tokens tokenPairDoc `json:"tokens"`
	User   authUserDoc  `json:"user"`
}

type meResponse struct {
	UserID string `json:"user_id" example:"123e4567-e89b-12d3-a456-426614174000"`
	Role   string `json:"role"    example:"buyer"`
	Email  string `json:"email"   example:"user@example.com"`
}

// RequestOTP godoc
//
//	@Summary      Request OTP
//	@Description  Send a 6-digit OTP to a phone number (SMS) or email address. Rate-limited to 5 attempts per 15 minutes.
//	@Tags         auth
//	@Accept       json
//	@Produce      json
//	@Param        body body requestOTPBody true "Phone number or email address"
//	@Success      200  {object} response.Response{data=map[string]string} "OTP sent"
//	@Failure      400  {object} response.Response
//	@Failure      429  {object} response.Response "Too many attempts"
//	@Router       /api/v1/auth/request-otp [post]
func (h *AuthHandler) RequestOTP(w http.ResponseWriter, r *http.Request) {
	var body requestOTPBody
	if err := request.Decode(r, &body); err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, err.Error()))
		return
	}
	if err := h.auth.RequestOTP(r.Context(), body.Contact, r.RemoteAddr); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "OTP sent"})
}

// VerifyOTP godoc
//
//	@Summary      Verify OTP and receive JWT tokens
//	@Description  Validate the 6-digit OTP. Creates a new user on first login. Returns access + refresh JWT pair.
//	@Tags         auth
//	@Accept       json
//	@Produce      json
//	@Param        body body verifyOTPBody true "Contact and OTP"
//	@Success      200  {object} response.Response{data=verifyOTPResponse}
//	@Failure      400  {object} response.Response
//	@Failure      401  {object} response.Response "OTP invalid or expired"
//	@Failure      403  {object} response.Response "User is banned"
//	@Router       /api/v1/auth/verify-otp [post]
func (h *AuthHandler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var body verifyOTPBody
	if err := request.Decode(r, &body); err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, err.Error()))
		return
	}
	tokens, user, err := h.auth.VerifyOTP(r.Context(), body.Contact, body.OTP)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"tokens": tokens,
		"user": map[string]interface{}{
			"id":          user.ID,
			"phone":       user.Phone,
			"email":       user.Email,
			"full_name":   user.FullName,
			"role":        user.Role,
			"is_verified": user.IsVerified,
		},
	})
}

// RefreshToken godoc
//
//	@Summary      Refresh access token
//	@Description  Exchange a valid refresh token for a new access + refresh token pair.
//	@Tags         auth
//	@Accept       json
//	@Produce      json
//	@Param        body body refreshTokenBody true "Refresh token"
//	@Success      200  {object} response.Response{data=tokenPairDoc}
//	@Failure      400  {object} response.Response
//	@Failure      401  {object} response.Response "Refresh token invalid or expired"
//	@Router       /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	var body refreshTokenBody
	if err := request.Decode(r, &body); err != nil {
		response.Error(w, apperror.New(apperror.ErrBadRequest, err.Error()))
		return
	}
	tokens, err := h.auth.RefreshToken(r.Context(), body.RefreshToken)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, tokens)
}

// Logout godoc
//
//	@Summary      Logout
//	@Description  Revoke the current access token. The token is invalidated immediately server-side.
//	@Tags         auth
//	@Produce      json
//	@Security     BearerAuth
//	@Success      200  {object} response.Response{data=map[string]string} "logged out"
//	@Failure      401  {object} response.Response
//	@Router       /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := extractBearerToken(r)
	if token == "" {
		response.Error(w, apperror.ErrUnauthorized)
		return
	}
	if err := h.auth.Logout(r.Context(), token); err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

// AdminOAuthLogin godoc
//
//	@Summary      Admin — initiate Google OAuth login
//	@Description  Redirects the admin browser to Google's OAuth consent page. Open this URL directly in a browser.
//	@Tags         admin-auth
//	@Param        state query string false "CSRF state token" example("cardswap_admin")
//	@Success      307  "Redirect to Google OAuth consent page"
//	@Router       /api/v1/admin/auth/login [get]
func (h *AuthHandler) AdminOAuthLogin(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	if state == "" {
		state = "cardswap_admin"
	}
	url := h.auth.GetAdminOAuthURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

// AdminOAuthCallback godoc
//
//	@Summary      Admin — Google OAuth callback
//	@Description  Google redirects here after consent. Returns JWT tokens if the email is in the admin allowlist.
//	@Tags         admin-auth
//	@Produce      json
//	@Param        code  query string true  "OAuth authorization code from Google"
//	@Param        state query string false "OAuth state"
//	@Success      200  {object} response.Response{data=verifyOTPResponse} "Tokens and admin user"
//	@Failure      400  {object} response.Response "Missing code"
//	@Failure      403  {object} response.Response "Email not in admin allowlist"
//	@Router       /api/v1/admin/auth/callback [get]
func (h *AuthHandler) AdminOAuthCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		response.Error(w, apperror.New(apperror.ErrBadRequest, "missing oauth code"))
		return
	}
	tokens, user, err := h.auth.HandleAdminOAuthCallback(r.Context(), code)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"tokens": tokens,
		"user":   map[string]interface{}{"id": user.ID, "email": user.Email, "role": user.Role},
	})
}

// Me godoc
//
//	@Summary      Get current user info
//	@Description  Returns the authenticated user's ID, role, and email decoded from the JWT claims.
//	@Tags         auth
//	@Produce      json
//	@Security     BearerAuth
//	@Success      200  {object} response.Response{data=meResponse}
//	@Failure      401  {object} response.Response
//	@Router       /api/v1/users/me [get]
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value(contextkey.UserID).(interface{ String() string })
	response.JSON(w, http.StatusOK, map[string]interface{}{
		"user_id": userID,
		"role":    r.Context().Value(contextkey.UserRole),
		"email":   r.Context().Value(contextkey.UserEmail),
	})
}

func extractBearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if len(h) > 7 && h[:7] == "Bearer " {
		return h[7:]
	}
	return ""
}
