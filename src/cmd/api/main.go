// CardSwap India API
//
//	@title          CardSwap India API
//	@version        1.0
//	@description    P2P gift card exchange marketplace. Card codes are AES-256 encrypted at rest and **never returned in API responses** — they are delivered to the buyer's registered email only.
//	@contact.name   CardSwap Support
//	@contact.email  support@cardswap.in
//
//	@host       localhost:8080
//	@BasePath   /
//	@schemes    https http
//
//	@securityDefinitions.apikey BearerAuth
//	@in         header
//	@name       Authorization
//	@description JWT access token. Format: `Bearer <token>`
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/gothi/vouchrs/config"
	deliveryhttp "github.com/gothi/vouchrs/src/delivery/http"
	"github.com/gothi/vouchrs/src/delivery/http/handler"
	"github.com/gothi/vouchrs/src/delivery/http/middleware"
	"github.com/gothi/vouchrs/src/delivery/worker"
	"github.com/gothi/vouchrs/src/external/cache"
	"github.com/gothi/vouchrs/src/external/cipher"
	emailpkg "github.com/gothi/vouchrs/src/external/email"
	"github.com/gothi/vouchrs/src/external/oauth"
	"github.com/gothi/vouchrs/src/external/payment/phonepe"
	"github.com/gothi/vouchrs/src/external/payout"
	"github.com/gothi/vouchrs/src/external/queue"
	smspkg "github.com/gothi/vouchrs/src/external/sms"
	"github.com/gothi/vouchrs/src/external/token"
	"github.com/gothi/vouchrs/src/external/verification"
	repoPkg "github.com/gothi/vouchrs/src/internal/repository/postgres"
	"github.com/gothi/vouchrs/src/internal/usecase/admin"
	"github.com/gothi/vouchrs/src/internal/usecase/auth"
	"github.com/gothi/vouchrs/src/internal/usecase/dashboard"
	"github.com/gothi/vouchrs/src/internal/usecase/listing"
	payoutUC "github.com/gothi/vouchrs/src/internal/usecase/payout"
	"github.com/gothi/vouchrs/src/internal/usecase/purchase"
	"github.com/gothi/vouchrs/src/internal/usecase/request"
	"github.com/gothi/vouchrs/src/pkg/logger"
)

func main() {
	// Load .env in non-production environments
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.App.Env)

	// --- Infrastructure ---

	ctx := context.Background()

	db, err := repoPkg.NewPool(ctx, cfg.DB.DSN)
	if err != nil {
		log.Error("connect to postgres", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	cacheService, redisClient := cache.NewRedisCache(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Error("connect to redis", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	// --- External adapters ---

	cipherSvc, err := cipher.NewAESCipher(cfg.Cipher.Key)
	if err != nil {
		log.Error("init cipher", "error", err)
		os.Exit(1)
	}

	tokenSvc := token.NewJWTService(
		cfg.JWT.AccessSecret, cfg.JWT.RefreshSecret,
		cfg.JWT.AccessTTLMin, cfg.JWT.RefreshTTLDay,
		cacheService,
	)

	smsSvc := smspkg.NewFast2SMS(cfg.Fast2SMS.APIKey)
	emailSvc := emailpkg.NewResendClient(cfg.Resend.APIKey, cfg.Resend.From)
	oauthSvc := oauth.NewGoogleOAuth(cfg.Google.ClientID, cfg.Google.ClientSecret, cfg.Google.RedirectURL)

	paymentGW := phonepe.NewPhonePeGateway(
		cfg.PhonePe.MerchantID, cfg.PhonePe.SaltKey,
		cfg.PhonePe.SaltIndex, cfg.PhonePe.Env,
	)

	payoutSvc := payout.NewRazorpayPayout(
		cfg.Razorpay.KeyID, cfg.Razorpay.KeySecret, cfg.Razorpay.AccountNumber,
	)

	verifySvc := verification.NewQwikcilverVerifier(
		cfg.Qwikcilver.TimeoutSeconds, cfg.Qwikcilver.Headless, log,
	)

	jobQueue, asynqClient := queue.NewAsynqJobQueue(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	defer asynqClient.Close()

	// --- Repositories ---

	userRepo := repoPkg.NewUserRepository(db)
	brandRepo := repoPkg.NewBrandRepository(db)
	listingRepo := repoPkg.NewListingRepository(db)
	poolGroupRepo := repoPkg.NewPoolGroupRepository(db)
	txnRepo := repoPkg.NewTransactionRepository(db)
	buyReqRepo := repoPkg.NewBuyRequestRepository(db)
	cardReqRepo := repoPkg.NewCardRequestRepository(db)
	verifyLogRepo := repoPkg.NewVerificationLogRepository(db)
	fraudFlagRepo := repoPkg.NewFraudFlagRepository(db)

	// --- Use cases ---

	authSvc := auth.NewService(
		userRepo, tokenSvc, cacheService,
		smsSvc, emailSvc, oauthSvc,
		cfg.OTP.Length, cfg.Admin.Emails, log,
	)

	listingSvc := listing.NewService(
		listingRepo, userRepo, brandRepo, poolGroupRepo,
		verifyLogRepo, fraudFlagRepo,
		cipherSvc, verifySvc, jobQueue, log,
	)

	purchaseSvc := purchase.NewService(
		listingRepo, txnRepo, userRepo, brandRepo,
		poolGroupRepo, verifyLogRepo, fraudFlagRepo,
		verifySvc, paymentGW, cipherSvc, emailSvc, jobQueue,
		cfg.App.BaseURL+"/api/v1/webhooks/phonepe",
		cfg.App.BaseURL+"/payment/return",
		log,
	)

	requestSvc := request.NewService(
		buyReqRepo, cardReqRepo, listingRepo, brandRepo,
		userRepo, emailSvc, cfg.Admin.Emails, log,
	)

	dashboardSvc := dashboard.NewService(listingRepo, txnRepo, buyReqRepo, cardReqRepo)

	payoutUsecaseSvc := payoutUC.NewService(txnRepo, userRepo, payoutSvc, log)

	adminSvc := admin.NewService(
		userRepo, listingRepo, txnRepo, cardReqRepo,
		fraudFlagRepo, emailSvc, log,
	)

	// --- Handlers ---

	authHandler := handler.NewAuthHandler(authSvc)
	listingHandler := handler.NewListingHandler(listingSvc)
	purchaseHandler := handler.NewPurchaseHandler(purchaseSvc, paymentGW)
	requestHandler := handler.NewRequestHandler(requestSvc)
	dashboardHandler := handler.NewDashboardHandler(dashboardSvc)
	adminHandler := handler.NewAdminHandler(adminSvc)

	// --- Router (with middleware) ---

	router := deliveryhttp.NewRouter(
		tokenSvc,
		authHandler,
		listingHandler,
		purchaseHandler,
		requestHandler,
		dashboardHandler,
		adminHandler,
	)

	// Wrap router with global middleware
	wrappedRouter := middleware.Recover(log)(
		middleware.Logger(log)(router),
	)

	// --- Background worker (same process) ---

	w := worker.New(
		cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB,
		cfg.Asynq.Concurrency,
		requestSvc, purchaseSvc, payoutUsecaseSvc,
		listingRepo, log,
	)
	go func() {
		if err := w.Start(); err != nil {
			log.Error("worker stopped", "error", err)
		}
	}()

	// --- HTTP server ---

	srv := deliveryhttp.NewServer(cfg.App.Port, wrappedRouter)

	// Graceful shutdown on SIGINT / SIGTERM
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Start(); err != nil {
			log.Info("server stopped", "error", err)
		}
	}()

	<-stop
	log.Info("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
