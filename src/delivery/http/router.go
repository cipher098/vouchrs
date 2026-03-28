package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	httpSwagger "github.com/swaggo/http-swagger/v2"

	_ "github.com/gothi/vouchrs/src/docs" // generated swagger docs
	"github.com/gothi/vouchrs/src/delivery/http/handler"
	"github.com/gothi/vouchrs/src/delivery/http/middleware"
	"github.com/gothi/vouchrs/src/internal/domain/port"
)

// NewRouter wires all routes and returns the chi router.
func NewRouter(
	tokens port.TokenService,
	authH *handler.AuthHandler,
	brandH *handler.BrandHandler,
	listingH *handler.ListingHandler,
	purchaseH *handler.PurchaseHandler,
	requestH *handler.RequestHandler,
	dashboardH *handler.DashboardHandler,
	adminH *handler.AdminHandler,
) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-VERIFY"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)

	// Health check — no auth
	r.Get("/health", handler.Health)

	// Swagger UI — /docs/ (interactive API explorer)
	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs/", http.StatusMovedPermanently)
	})
	r.Get("/docs/*", httpSwagger.Handler(
		httpSwagger.URL("/docs/doc.json"),
	))

	r.Route("/api/v1", func(r chi.Router) {
		// --- Public routes ---
		r.Get("/brands", brandH.ListBrands)
		r.Get("/marketplace", listingH.Marketplace)
		r.Get("/listings/recommended-price", listingH.RecommendedPrice)
		r.Get("/listings/{id}", listingH.GetByID)

		// --- Auth ---
		r.Route("/auth", func(r chi.Router) {
			r.Post("/request-otp", authH.RequestOTP)
			r.Post("/verify-otp", authH.VerifyOTP)
			r.Post("/refresh", authH.RefreshToken)
			r.With(middleware.Authenticate(tokens)).Post("/logout", authH.Logout)
		})

		// --- Admin OAuth ---
		r.Route("/admin/auth", func(r chi.Router) {
			r.Get("/login", authH.AdminOAuthLogin)
			r.Get("/callback", authH.AdminOAuthCallback)
		})

		// --- Webhook (no auth — verified by signature) ---
		r.Post("/webhooks/phonepe", purchaseH.PhonePeWebhook)

		// --- Authenticated routes ---
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(tokens))

			r.Get("/users/me", authH.Me)

			// Listings
			r.Post("/listings", listingH.Create)
			r.Delete("/listings/{id}", listingH.Cancel)

			// Purchase flow
			r.Post("/listings/{id}/buy", purchaseH.InitiateBuy)
			r.Post("/pool-groups/{id}/buy", purchaseH.InitiateBuyFromPool)
			r.Get("/transactions/{id}", purchaseH.GetTransaction)
			r.Post("/transactions/{id}/confirm", purchaseH.ConfirmRedemption)

			// Buy requests
			r.Post("/buy-requests", requestH.CreateBuyRequest)
			r.Get("/buy-requests", requestH.ListMyBuyRequests)
			r.Delete("/buy-requests/{id}", requestH.DeleteBuyRequest)

			// Card requests
			r.Post("/card-requests", requestH.CreateCardRequest)

			// Dashboard
			r.Get("/dashboard/listings", dashboardH.MyListings)
			r.Get("/dashboard/purchases", dashboardH.MyPurchases)
			r.Get("/dashboard/requests", dashboardH.MyRequests)

			// --- Admin-only routes ---
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole("admin"))

				r.Get("/admin/card-requests", adminH.ListCardRequests)
				r.Patch("/admin/card-requests/{id}", adminH.ReviewCardRequest)
				r.Get("/admin/fraud-flags", adminH.ListFraudFlags)
				r.Patch("/admin/fraud-flags/{id}/resolve", adminH.ResolveFraudFlag)
				r.Patch("/admin/users/{id}/ban", adminH.BanUser)
				r.Get("/admin/stats", adminH.Stats)
				r.Get("/admin/listings", adminH.ListListings)
				r.Get("/admin/transactions", adminH.ListTransactions)
			})
		})
	})

	return r
}
