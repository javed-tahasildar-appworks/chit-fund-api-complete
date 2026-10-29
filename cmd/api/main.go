package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"chit-fund-api/internal/config"
	"chit-fund-api/internal/handler"
	"chit-fund-api/internal/middleware"
	"chit-fund-api/internal/repository"
	"chit-fund-api/internal/service"
	"chit-fund-api/pkg/cache"
	"chit-fund-api/pkg/database"
	"chit-fund-api/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "chit-fund-api/docs"
)

// @title           Chit Fund Management API
// @version         2.0
// @description     Production-ready multi-tenant chit fund management system.
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.email  support@chitfund.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

func main() {
	cfg := config.Load()
	logger.Init(cfg.Environment)
	log.Info().Msg("Starting Chit Fund API v2...")

	db, err := database.NewPostgresPool(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	redisClient := cache.NewRedisClient(cfg.Redis)
	defer redisClient.Close()

	repos := initRepositories(db)
	services := initServices(repos, redisClient, cfg)
	handlers := initHandlers(services, cfg)

	router := setupRouter(cfg, handlers)

	srv := &http.Server{
		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:        router,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	log.Info().Msgf("Server running on port %d — Swagger: http://localhost:%d/swagger/index.html", cfg.Server.Port, cfg.Server.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Forced shutdown")
	}
	log.Info().Msg("Server stopped")
}

// ─── Dependency wiring ────────────────────────────────────────────────────────

func initRepositories(db *database.PostgresPool) *repository.Repositories {
	return &repository.Repositories{
		Company:      repository.NewCompanyRepository(db),
		Branch:       repository.NewBranchRepository(db),
		State:        repository.NewStateRepository(db),
		User:         repository.NewUserRepository(db),
		Role:         repository.NewRoleRepository(db),
		Subscriber:   repository.NewSubscriberRepository(db),
		KYC:          repository.NewKYCRepository(db),
		ChitGroup:    repository.NewChitGroupRepository(db),
		Auction:      repository.NewAuctionRepository(db),
		Installment:  repository.NewInstallmentRepository(db),
		Payout:       repository.NewPayoutRepository(db),
		Default:      repository.NewDefaultRepository(db),
		LegalNotice:  repository.NewLegalNoticeRepository(db),
		Dashboard:    repository.NewDashboardRepository(db),
		AuditLog:     repository.NewAuditLogRepository(db),
		Mobile:       repository.NewMobileRepository(db),
		Payment:      repository.NewPaymentRepository(db),
		Notification: repository.NewNotificationRepository(db),
		Financial:    repository.NewFinancialRepository(db),
	}
}

func initServices(repos *repository.Repositories, redisCache *cache.RedisClient, cfg *config.Config) *service.Services {
	return &service.Services{
		Auth:         service.NewAuthService(repos.User, cfg.JWT, redisCache),
		Company:      service.NewCompanyService(repos.Company, redisCache),
		Branch:       service.NewBranchService(repos.Branch, repos.State, repos.Role, redisCache),
		User:         service.NewUserService(repos.User, redisCache),
		Subscriber:   service.NewSubscriberService(repos.Subscriber, repos.KYC, redisCache),
		ChitGroup:    service.NewChitGroupService(repos.ChitGroup, redisCache),
		Auction:      service.NewAuctionService(repos.Auction, repos.ChitGroup, redisCache),
		Installment:  service.NewInstallmentService(repos.Installment, redisCache),
		Payout:       service.NewPayoutService(repos.Payout, repos.ChitGroup, redisCache),
		Default:      service.NewDefaultService(repos.Default, redisCache),
		LegalNotice:  service.NewLegalNoticeService(repos.LegalNotice, redisCache),
		Dashboard:    service.NewDashboardService(repos.Dashboard),
		Report:       service.NewReportService(repos.Dashboard, repos.AuditLog),
		Mobile:       service.NewMobileService(repos.Mobile, cfg.JWT, redisCache),
		Payment:      service.NewPaymentService(repos.Payment, cfg.Razorpay.KeyID, cfg.Razorpay.KeySecret, cfg.Razorpay.AccountNumber),
		Notification: service.NewNotificationService(repos.Notification),
		Financial:    service.NewFinancialService(repos.Financial),
	}
}

func initHandlers(svcs *service.Services, cfg *config.Config) *handler.Handlers {
	return &handler.Handlers{
		Auth:         handler.NewAuthHandler(svcs.Auth),
		Company:      handler.NewCompanyHandler(svcs.Company, svcs.Branch),
		User:         handler.NewUserHandler(svcs.User),
		Subscriber:   handler.NewSubscriberHandler(svcs.Subscriber),
		ChitGroup:    handler.NewChitGroupHandler(svcs.ChitGroup),
		Auction:      handler.NewAuctionHandler(svcs.Auction),
		Installment:  handler.NewInstallmentHandler(svcs.Installment),
		Payout:       handler.NewPayoutHandler(svcs.Payout),
		Default:      handler.NewDefaultHandler(svcs.Default, svcs.LegalNotice),
		LegalNotice:  handler.NewLegalNoticeHandler(svcs.LegalNotice),
		Dashboard:    handler.NewDashboardHandler(svcs.Dashboard),
		Report:       handler.NewReportHandler(svcs.Report),
		Mobile:       handler.NewMobileHandler(svcs.Mobile, svcs.Auth),
		Payment:      handler.NewPaymentHandler(svcs.Payment, cfg.Razorpay.WebhookSecret),
		Notification: handler.NewNotificationHandler(svcs.Notification),
		Financial:    handler.NewFinancialHandler(svcs.Financial),
	}
}

// ─── Router ───────────────────────────────────────────────────────────────────

func setupRouter(cfg *config.Config, h *handler.Handlers) *gin.Engine {
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.Logger())
	r.Use(middleware.CORS())

	// Swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "version": "2.0.0", "time": time.Now().Format(time.RFC3339)})
	})

	v1 := r.Group("/api/v1")

	// ── Public routes (no auth) ───────────────────────────────────────────────
	auth := v1.Group("/auth")
	{
		auth.POST("/login", h.Auth.Login)
		auth.POST("/refresh", h.Auth.RefreshToken) // no auth needed
	}

	// ── Protected routes ──────────────────────────────────────────────────────
	p := v1.Group("")
	p.Use(middleware.AuthRequired(cfg.JWT.Secret))
	p.Use(middleware.TenantContext())
	p.Use(middleware.RateLimit(cfg.RateLimit))
	{
		// ── Auth (protected) ─────────────────────────────────────────────────
		p.POST("/auth/logout", h.Auth.Logout)
		p.GET("/auth/me", h.Auth.GetMe)
		p.POST("/auth/change-password", h.Auth.ChangePassword)

		// ── Dashboard ─────────────────────────────────────────────────────────
		p.GET("/dashboard", h.Dashboard.GetBundle)
		p.GET("/dashboard/kpis", h.Dashboard.GetKPIs)
		p.GET("/dashboard/collections", h.Dashboard.GetCollections)

		// ── Company & Configuration ───────────────────────────────────────────
		p.GET("/company", h.Company.GetProfile)
		p.PUT("/company", middleware.RequirePermission("config", "company", "update"), h.Company.UpdateProfile)

		branches := p.Group("/branches")
		{
			branches.GET("", h.Company.ListBranches)
			branches.POST("", middleware.RequirePermission("config", "branches", "create"), h.Company.CreateBranch)
			branches.GET("/:id", h.Company.GetBranch)
			branches.PUT("/:id", middleware.RequirePermission("config", "branches", "update"), h.Company.UpdateBranch)
		}

		p.GET("/states", h.Company.ListStates)
		p.GET("/states/:id", h.Company.GetState)

		// ── Users & Access ────────────────────────────────────────────────────
		users := p.Group("/users")
		users.Use(middleware.RequirePermission("admin", "users", "view"))
		{
			users.GET("", h.User.List)
			users.POST("", middleware.RequirePermission("admin", "users", "create"), h.User.Create)
			users.GET("/:id", h.User.GetByID)
			users.PUT("/:id", middleware.RequirePermission("admin", "users", "update"), h.User.Update)
			users.PATCH("/:id/deactivate", middleware.RequirePermission("admin", "users", "update"), h.User.Deactivate)
			users.PATCH("/:id/activate", middleware.RequirePermission("admin", "users", "update"), h.User.Activate)
		}

		roles := p.Group("/roles")
		{
			roles.GET("", h.Company.ListRoles)
			roles.GET("/:id", h.Company.GetRole)
			roles.PUT("/:id", middleware.RequireRole("SUPER_ADMIN", "ADMIN"), h.Company.UpdateRolePermissions)
		}

		// ── Subscribers & KYC ────────────────────────────────────────────────
		subscribers := p.Group("/subscribers")
		subscribers.Use(middleware.RequirePermission("subscribers", "subscribers", "view"))
		{
			subscribers.GET("", h.Subscriber.List)
			subscribers.POST("", middleware.RequirePermission("subscribers", "subscribers", "create"), h.Subscriber.Create)
			subscribers.GET("/:id", h.Subscriber.GetByID)
			subscribers.PUT("/:id", middleware.RequirePermission("subscribers", "subscribers", "update"), h.Subscriber.Update)
			subscribers.PATCH("/:id/blacklist", middleware.RequirePermission("subscribers", "subscribers", "update"), h.Subscriber.Blacklist)
			subscribers.PATCH("/:id/unblacklist", middleware.RequirePermission("subscribers", "subscribers", "update"), h.Subscriber.Unblacklist)
			subscribers.GET("/:id/documents", h.Subscriber.ListDocuments)
			subscribers.POST("/:id/documents", h.Subscriber.UploadDocument)
			subscribers.PATCH("/:id/documents/:docId/verify", middleware.RequirePermission("subscribers", "kyc", "verify"), h.Subscriber.VerifyDocument)
			subscribers.GET("/:id/banks", h.Subscriber.ListBanks)
			subscribers.POST("/:id/banks", h.Subscriber.AddBank)
			subscribers.DELETE("/:id/banks/:bankId", h.Subscriber.DeleteBank)
		}

		// ── Chit Groups ───────────────────────────────────────────────────────
		chits := p.Group("/chit-groups")
		chits.Use(middleware.RequirePermission("chits", "groups", "view"))
		{
			chits.GET("", h.ChitGroup.List)
			chits.POST("", middleware.RequirePermission("chits", "groups", "create"), h.ChitGroup.Create)
			chits.GET("/:id", h.ChitGroup.GetByID)
			chits.PUT("/:id", middleware.RequirePermission("chits", "groups", "update"), h.ChitGroup.Update)
			chits.PATCH("/:id/close", middleware.RequirePermission("chits", "groups", "close"), h.ChitGroup.Close)
			chits.GET("/:id/subscribers", h.ChitGroup.ListSubscribers)
			chits.POST("/:id/subscribers", middleware.RequirePermission("chits", "subscribers", "create"), h.ChitGroup.AddSubscriber)
			chits.DELETE("/:id/subscribers/:csId", middleware.RequirePermission("chits", "subscribers", "delete"), h.ChitGroup.RemoveEnrollment)
		}

		// ── Auctions ──────────────────────────────────────────────────────────
		auctions := p.Group("/auctions")
		auctions.Use(middleware.RequirePermission("chits", "auctions", "view"))
		{
			auctions.GET("", h.Auction.List)
			auctions.POST("", middleware.RequirePermission("chits", "auctions", "create"), h.Auction.Create)
			auctions.GET("/:id", h.Auction.GetByID)
			auctions.PUT("/:id", middleware.RequirePermission("chits", "auctions", "update"), h.Auction.Update)
			auctions.PATCH("/:id/conduct", middleware.RequirePermission("chits", "auctions", "conduct"), h.Auction.Conduct)
			auctions.PATCH("/:id/cancel", middleware.RequirePermission("chits", "auctions", "conduct"), h.Auction.Cancel)
			auctions.PATCH("/:id/post-result", middleware.RequirePermission("chits", "auctions", "conduct"), h.Auction.PostResult)
			auctions.GET("/:id/bids", h.Auction.ListBids)
			auctions.POST("/:id/bids", h.Auction.SubmitBid)
			auctions.PATCH("/:id/bids/:bidId/win", middleware.RequirePermission("chits", "auctions", "conduct"), h.Auction.MarkBidWinner)
		}

		// ── Installments & Payments ───────────────────────────────────────────
		installments := p.Group("/installments")
		installments.Use(middleware.RequirePermission("payments", "installments", "view"))
		{
			installments.GET("", h.Installment.List)
			installments.POST("/generate", middleware.RequirePermission("payments", "installments", "create"), h.Installment.GenerateBatch)
			installments.GET("/:id", h.Installment.GetByID)
		}

		subInst := p.Group("/subscriber-installments")
		subInst.Use(middleware.RequirePermission("payments", "installments", "view"))
		{
			subInst.GET("", h.Installment.ListSubscriberInstallments)
			subInst.GET("/:id", h.Installment.GetSubscriberInstallmentByID)
		}

		transactions := p.Group("/payment-transactions")
		transactions.Use(middleware.RequirePermission("payments", "transactions", "view"))
		{
			transactions.GET("", h.Installment.ListPaymentTransactions)
			transactions.POST("", middleware.RequirePermission("payments", "transactions", "create"), h.Installment.RecordPayment)
			transactions.GET("/:id", h.Installment.GetPaymentTransactionByID)
		}

		// ── Payouts & Dividends ───────────────────────────────────────────────
		payouts := p.Group("/payouts")
		payouts.Use(middleware.RequirePermission("payments", "payouts", "view"))
		{
			payouts.GET("", h.Payout.List)
			payouts.POST("", middleware.RequirePermission("payments", "payouts", "create"), h.Payout.Initiate)
			payouts.GET("/:id", h.Payout.GetByID)
			payouts.PATCH("/:id/approve", middleware.RequireRole("MANAGER", "ADMIN", "SUPER_ADMIN"), h.Payout.Approve)
			payouts.PATCH("/:id/disburse", middleware.RequirePermission("payments", "payouts", "disburse"), h.Payout.Disburse)
		}

		dividends := p.Group("/dividends")
		{
			dividends.GET("", h.Payout.ListDividends)
			dividends.POST("/calculate", middleware.RequirePermission("payments", "dividends", "create"), h.Payout.CalculateDividend)
			dividends.POST("/post", middleware.RequirePermission("payments", "dividends", "create"), h.Payout.PostDividend)
		}

		// ── Defaults & Legal ──────────────────────────────────────────────────
		defaults := p.Group("/defaults")
		defaults.Use(middleware.RequirePermission("legal", "defaults", "view"))
		{
			defaults.GET("", h.Default.List)
			defaults.POST("/detect", middleware.RequirePermission("legal", "defaults", "detect"), h.Default.Detect)
			defaults.PATCH("/:id/flag", middleware.RequirePermission("legal", "defaults", "update"), h.Default.Flag)
			defaults.PATCH("/:id/resolve", middleware.RequirePermission("legal", "defaults", "update"), h.Default.Resolve)
			defaults.POST("/:id/reminder", h.Default.SendReminder)
		}

		legalNotices := p.Group("/legal-notices")
		legalNotices.Use(middleware.RequirePermission("legal", "notices", "view"))
		{
			legalNotices.GET("", h.LegalNotice.List)
			legalNotices.POST("", middleware.RequirePermission("legal", "notices", "create"), h.LegalNotice.Issue)
			legalNotices.GET("/:id", h.LegalNotice.GetByID)
			legalNotices.PATCH("/:id/escalate", middleware.RequirePermission("legal", "notices", "escalate"), h.LegalNotice.Escalate)
		}

		// ── Reports & Audit ───────────────────────────────────────────────────
		reports := p.Group("/reports")
		reports.Use(middleware.RequirePermission("reports", "reports", "view"))
		{
			reports.GET("/monthly-collections", h.Report.MonthlyCollections)
			reports.GET("/defaulters", h.Report.DefaultersReport)
			reports.GET("/chit-performance", h.Report.ChitPerformance)
			reports.GET("/subscriber-ledger/:id", h.Report.SubscriberLedger)
		}

		p.GET("/audit-logs", middleware.RequirePermission("admin", "audit", "view"), h.Report.AuditLogs)
		p.GET("/system-logs", middleware.RequireRole("SUPER_ADMIN", "ADMIN"), h.Report.SystemLogs)

		// ── Financial Suite ──────────────────────────────────────────────────
		finance := p.Group("/finance")
		finance.Use(middleware.RequirePermission("reports", "financial", "view"))
		{
			// Financial Statements
			finance.GET("/pl", h.Financial.ProfitLoss)                     // P&L Statement
			finance.GET("/balance-sheet", h.Financial.BalanceSheet)        // Balance Sheet
			finance.GET("/cash-flow", h.Financial.CashFlow)                // Cash Flow Statement
			finance.GET("/creditors-debtors", h.Financial.CreditorDebtors) // Creditor/Debtor Schedule
			finance.GET("/gst", h.Financial.GSTSummary)                    // GST Summary

			// Expense Management
			finance.GET("/expenses", h.Financial.ListExpenses)
			finance.POST("/expenses", middleware.RequirePermission("reports", "financial", "create"),
				h.Financial.CreateExpense)

			// Asset Management
			finance.POST("/assets", middleware.RequirePermission("reports", "financial", "create"),
				h.Financial.CreateAsset)

			// Capital Movements
			finance.POST("/capital", middleware.RequireRole("SUPER_ADMIN", "ADMIN"),
				h.Financial.CreateCapital)
		}
	}

	// ── Mobile App Routes ────────────────────────────────────────────────────
	// These are completely separate from the staff API routes above.
	appGroup := r.Group("/api/mobile")
	{
		// ── Subscriber App — Public (no auth) ────────────────────────────────
		subPublic := appGroup.Group("/subscriber")
		{
			subPublic.POST("/register", h.Mobile.AppRegister)
			subPublic.POST("/login", h.Mobile.AppLogin)
			subPublic.POST("/send-otp", h.Mobile.AppSendOTP)
			subPublic.POST("/verify-otp", h.Mobile.AppVerifyOTP)
		}

		// ── Subscriber App — Protected (requires subscriber JWT) ─────────────
		subAuth := appGroup.Group("/subscriber")
		subAuth.Use(middleware.AppSubscriberAuth(cfg.JWT.Secret))
		{
			subAuth.PUT("/device", h.Mobile.AppUpdateDevice)
			subAuth.GET("/dashboard", h.Mobile.AppGetDashboard)
			subAuth.GET("/chits", h.Mobile.AppGetChits)
			subAuth.GET("/dues", h.Mobile.AppGetDues)
			subAuth.GET("/payments", h.Mobile.AppGetPaymentHistory)
			subAuth.GET("/notifications", h.Mobile.AppGetNotifications)
			subAuth.PATCH("/notifications/:id/read", h.Mobile.AppMarkNotificationRead)
			subAuth.PATCH("/notifications/read-all", h.Mobile.AppMarkAllNotificationsRead)
		}

		// ── Agent App — Public ────────────────────────────────────────────────
		agentPublic := appGroup.Group("/agent")
		{
			agentPublic.POST("/login", h.Mobile.AgentLogin)
		}

		// ── Agent App — Protected (requires staff JWT + agent role) ──────────
		agentAuth := appGroup.Group("/agent")
		agentAuth.Use(middleware.AuthRequired(cfg.JWT.Secret))
		agentAuth.Use(middleware.TenantContext())
		agentAuth.Use(middleware.RequireAgentRole())
		{
			agentAuth.GET("/dashboard", h.Mobile.AgentDashboard)
			agentAuth.GET("/subscribers", h.Mobile.AgentGetSubscribers)
			agentAuth.POST("/collect", h.Mobile.AgentCollect)
		}
	}

	// ── Digital Payment Routes ────────────────────────────────────────────────
	// Staff API: full CRUD on payments
	// Subscriber App: create order + verify only
	// Webhook: outside all auth groups

	// Webhook (no auth — Razorpay calls this directly from their servers)
	r.POST("/webhooks/razorpay", h.Payment.RazorpayWebhook)

	// Subscriber app online payment (Feature 1 — subscriber initiates)
	subPayments := r.Group("/api/mobile/subscriber/payments")
	subPayments.Use(middleware.AppSubscriberAuth(cfg.JWT.Secret))
	{
		subPayments.POST("/orders", h.Payment.CreateOrder)
		subPayments.POST("/verify", h.Payment.VerifyPayment)
		subPayments.GET("/:id", h.Payment.GetPayment)
	}

	return r
}

// package main

// import (
// 	"context"
// 	"fmt"
// 	"net/http"
// 	"os"
// 	"os/signal"
// 	"syscall"
// 	"time"

// 	"chit-fund-api/internal/config"
// 	"chit-fund-api/internal/handler"
// 	"chit-fund-api/internal/middleware"
// 	"chit-fund-api/internal/repository"
// 	"chit-fund-api/internal/service"
// 	"chit-fund-api/pkg/cache"
// 	"chit-fund-api/pkg/database"
// 	"chit-fund-api/pkg/logger"

// 	"github.com/gin-gonic/gin"
// 	"github.com/rs/zerolog/log"
// 	swaggerFiles "github.com/swaggo/files"
// 	ginSwagger "github.com/swaggo/gin-swagger"

// 	_ "chit-fund-api/docs"
// )

// // @title           Chit Fund Management API
// // @version         2.0
// // @description     Production-ready multi-tenant chit fund management system.
// // @termsOfService  http://swagger.io/terms/

// // @contact.name   API Support
// // @contact.email  support@chitfund.com

// // @license.name  MIT
// // @license.url   https://opensource.org/licenses/MIT

// // @host      localhost:8080
// // @BasePath  /api/v1

// // @securityDefinitions.apikey BearerAuth
// // @in header
// // @name Authorization
// // @description Type "Bearer" followed by a space and JWT token.

// func main() {
// 	cfg := config.Load()
// 	logger.Init(cfg.Environment)
// 	log.Info().Msg("Starting Chit Fund API v2...")

// 	db, err := database.NewPostgresPool(cfg.Database)
// 	if err != nil {
// 		log.Fatal().Err(err).Msg("Failed to connect to database")
// 	}
// 	defer db.Close()

// 	redisClient := cache.NewRedisClient(cfg.Redis)
// 	defer redisClient.Close()

// 	repos := initRepositories(db)
// 	services := initServices(repos, redisClient, cfg)
// 	handlers := initHandlers(services)

// 	router := setupRouter(cfg, handlers)

// 	srv := &http.Server{
// 		Addr:           fmt.Sprintf(":%d", cfg.Server.Port),
// 		Handler:        router,
// 		ReadTimeout:    15 * time.Second,
// 		WriteTimeout:   30 * time.Second,
// 		IdleTimeout:    60 * time.Second,
// 		MaxHeaderBytes: 1 << 20,
// 	}

// 	go func() {
// 		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
// 			log.Fatal().Err(err).Msg("Server failed")
// 		}
// 	}()

// 	log.Info().Msgf("Server running on port %d — Swagger: http://localhost:%d/swagger/index.html", cfg.Server.Port, cfg.Server.Port)

// 	quit := make(chan os.Signal, 1)
// 	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
// 	<-quit

// 	log.Info().Msg("Shutting down gracefully...")
// 	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
// 	defer cancel()
// 	if err := srv.Shutdown(ctx); err != nil {
// 		log.Fatal().Err(err).Msg("Forced shutdown")
// 	}
// 	log.Info().Msg("Server stopped")
// }

// // ─── Dependency wiring ────────────────────────────────────────────────────────

// func initRepositories(db *database.PostgresPool) *repository.Repositories {
// 	return &repository.Repositories{
// 		Company:     repository.NewCompanyRepository(db),
// 		Branch:      repository.NewBranchRepository(db),
// 		State:       repository.NewStateRepository(db),
// 		User:        repository.NewUserRepository(db),
// 		Role:        repository.NewRoleRepository(db),
// 		Subscriber:  repository.NewSubscriberRepository(db),
// 		KYC:         repository.NewKYCRepository(db),
// 		ChitGroup:   repository.NewChitGroupRepository(db),
// 		Auction:     repository.NewAuctionRepository(db),
// 		Installment: repository.NewInstallmentRepository(db),
// 		Payout:      repository.NewPayoutRepository(db),
// 		Default:     repository.NewDefaultRepository(db),
// 		LegalNotice: repository.NewLegalNoticeRepository(db),
// 		Dashboard:   repository.NewDashboardRepository(db),
// 		AuditLog:    repository.NewAuditLogRepository(db),
// 	}
// }

// func initServices(repos *repository.Repositories, redisCache *cache.RedisClient, cfg *config.Config) *service.Services {
// 	return &service.Services{
// 		Auth:        service.NewAuthService(repos.User, cfg.JWT, redisCache),
// 		Company:     service.NewCompanyService(repos.Company, redisCache),
// 		Branch:      service.NewBranchService(repos.Branch, repos.State, repos.Role, redisCache),
// 		User:        service.NewUserService(repos.User, redisCache),
// 		Subscriber:  service.NewSubscriberService(repos.Subscriber, repos.KYC, redisCache),
// 		ChitGroup:   service.NewChitGroupService(repos.ChitGroup, redisCache),
// 		Auction:     service.NewAuctionService(repos.Auction, repos.ChitGroup, redisCache),
// 		Installment: service.NewInstallmentService(repos.Installment, redisCache),
// 		Payout:      service.NewPayoutService(repos.Payout, repos.ChitGroup, redisCache),
// 		Default:     service.NewDefaultService(repos.Default, redisCache),
// 		LegalNotice: service.NewLegalNoticeService(repos.LegalNotice, redisCache),
// 		Dashboard:   service.NewDashboardService(repos.Dashboard),
// 		Report:      service.NewReportService(repos.Dashboard, repos.AuditLog),
// 	}
// }

// func initHandlers(svcs *service.Services) *handler.Handlers {
// 	return &handler.Handlers{
// 		Auth:        handler.NewAuthHandler(svcs.Auth),
// 		Company:     handler.NewCompanyHandler(svcs.Company, svcs.Branch),
// 		User:        handler.NewUserHandler(svcs.User),
// 		Subscriber:  handler.NewSubscriberHandler(svcs.Subscriber),
// 		ChitGroup:   handler.NewChitGroupHandler(svcs.ChitGroup),
// 		Auction:     handler.NewAuctionHandler(svcs.Auction),
// 		Installment: handler.NewInstallmentHandler(svcs.Installment),
// 		Payout:      handler.NewPayoutHandler(svcs.Payout),
// 		Default:     handler.NewDefaultHandler(svcs.Default, svcs.LegalNotice),
// 		LegalNotice: handler.NewLegalNoticeHandler(svcs.LegalNotice),
// 		Dashboard:   handler.NewDashboardHandler(svcs.Dashboard),
// 		Report:      handler.NewReportHandler(svcs.Report),
// 	}
// }

// // ─── Router ───────────────────────────────────────────────────────────────────

// func setupRouter(cfg *config.Config, h *handler.Handlers) *gin.Engine {
// 	if cfg.Environment == "production" {
// 		gin.SetMode(gin.ReleaseMode)
// 	}

// 	r := gin.New()
// 	r.Use(gin.Recovery())
// 	r.Use(middleware.Logger())
// 	r.Use(middleware.CORS())

// 	// Swagger
// 	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

// 	// Health check
// 	r.GET("/health", func(c *gin.Context) {
// 		c.JSON(http.StatusOK, gin.H{"status": "healthy", "version": "2.0.0", "time": time.Now().Format(time.RFC3339)})
// 	})

// 	v1 := r.Group("/api/v1")

// 	// ── Public routes (no auth) ───────────────────────────────────────────────
// 	auth := v1.Group("/auth")
// 	{
// 		auth.POST("/login", h.Auth.Login)
// 		auth.POST("/refresh", h.Auth.RefreshToken) // no auth needed
// 	}

// 	// ── Protected routes ──────────────────────────────────────────────────────
// 	p := v1.Group("")
// 	p.Use(middleware.AuthRequired(cfg.JWT.Secret))
// 	p.Use(middleware.TenantContext())
// 	p.Use(middleware.RateLimit(cfg.RateLimit))
// 	{
// 		// ── Auth (protected) ─────────────────────────────────────────────────
// 		p.POST("/auth/logout", h.Auth.Logout)
// 		p.GET("/auth/me", h.Auth.GetMe)
// 		p.POST("/auth/change-password", h.Auth.ChangePassword)

// 		// ── Dashboard ─────────────────────────────────────────────────────────
// 		p.GET("/dashboard", h.Dashboard.GetBundle)
// 		p.GET("/dashboard/kpis", h.Dashboard.GetKPIs)
// 		p.GET("/dashboard/collections", h.Dashboard.GetCollections)

// 		// ── Company & Configuration ───────────────────────────────────────────
// 		p.GET("/company", h.Company.GetProfile)
// 		p.PUT("/company", middleware.RequirePermission("config", "company", "update"), h.Company.UpdateProfile)

// 		branches := p.Group("/branches")
// 		{
// 			branches.GET("", h.Company.ListBranches)
// 			branches.POST("", middleware.RequirePermission("config", "branches", "create"), h.Company.CreateBranch)
// 			branches.GET("/:id", h.Company.GetBranch)
// 			branches.PUT("/:id", middleware.RequirePermission("config", "branches", "update"), h.Company.UpdateBranch)
// 		}

// 		p.GET("/states", h.Company.ListStates)
// 		p.GET("/states/:id", h.Company.GetState)

// 		// ── Users & Access ────────────────────────────────────────────────────
// 		users := p.Group("/users")
// 		users.Use(middleware.RequirePermission("admin", "users", "view"))
// 		{
// 			users.GET("", h.User.List)
// 			users.POST("", middleware.RequirePermission("admin", "users", "create"), h.User.Create)
// 			users.GET("/:id", h.User.GetByID)
// 			users.PUT("/:id", middleware.RequirePermission("admin", "users", "update"), h.User.Update)
// 			users.PATCH("/:id/deactivate", middleware.RequirePermission("admin", "users", "update"), h.User.Deactivate)
// 			users.PATCH("/:id/activate", middleware.RequirePermission("admin", "users", "update"), h.User.Activate)
// 		}

// 		roles := p.Group("/roles")
// 		{
// 			roles.GET("", h.Company.ListRoles)
// 			roles.GET("/:id", h.Company.GetRole)
// 			roles.PUT("/:id", middleware.RequireRole("SUPER_ADMIN", "ADMIN"), h.Company.UpdateRolePermissions)
// 		}

// 		// ── Subscribers & KYC ────────────────────────────────────────────────
// 		subscribers := p.Group("/subscribers")
// 		subscribers.Use(middleware.RequirePermission("subscribers", "subscribers", "view"))
// 		{
// 			subscribers.GET("", h.Subscriber.List)
// 			subscribers.POST("", middleware.RequirePermission("subscribers", "subscribers", "create"), h.Subscriber.Create)
// 			subscribers.GET("/:id", h.Subscriber.GetByID)
// 			subscribers.PUT("/:id", middleware.RequirePermission("subscribers", "subscribers", "update"), h.Subscriber.Update)
// 			subscribers.PATCH("/:id/blacklist", middleware.RequirePermission("subscribers", "subscribers", "update"), h.Subscriber.Blacklist)
// 			subscribers.PATCH("/:id/unblacklist", middleware.RequirePermission("subscribers", "subscribers", "update"), h.Subscriber.Unblacklist)
// 			subscribers.GET("/:id/documents", h.Subscriber.ListDocuments)
// 			subscribers.POST("/:id/documents", h.Subscriber.UploadDocument)
// 			subscribers.PATCH("/:id/documents/:docId/verify", middleware.RequirePermission("subscribers", "kyc", "verify"), h.Subscriber.VerifyDocument)
// 			subscribers.GET("/:id/banks", h.Subscriber.ListBanks)
// 			subscribers.POST("/:id/banks", h.Subscriber.AddBank)
// 			subscribers.DELETE("/:id/banks/:bankId", h.Subscriber.DeleteBank)
// 		}

// 		// ── Chit Groups ───────────────────────────────────────────────────────
// 		chits := p.Group("/chit-groups")
// 		chits.Use(middleware.RequirePermission("chits", "groups", "view"))
// 		{
// 			chits.GET("", h.ChitGroup.List)
// 			chits.POST("", middleware.RequirePermission("chits", "groups", "create"), h.ChitGroup.Create)
// 			chits.GET("/:id", h.ChitGroup.GetByID)
// 			chits.PUT("/:id", middleware.RequirePermission("chits", "groups", "update"), h.ChitGroup.Update)
// 			chits.PATCH("/:id/close", middleware.RequirePermission("chits", "groups", "close"), h.ChitGroup.Close)
// 			chits.GET("/:id/subscribers", h.ChitGroup.ListSubscribers)
// 			chits.POST("/:id/subscribers", middleware.RequirePermission("chits", "subscribers", "create"), h.ChitGroup.AddSubscriber)
// 			chits.DELETE("/:id/subscribers/:csId", middleware.RequirePermission("chits", "subscribers", "delete"), h.ChitGroup.RemoveEnrollment)
// 		}

// 		// ── Auctions ──────────────────────────────────────────────────────────
// 		auctions := p.Group("/auctions")
// 		auctions.Use(middleware.RequirePermission("chits", "auctions", "view"))
// 		{
// 			auctions.GET("", h.Auction.List)
// 			auctions.POST("", middleware.RequirePermission("chits", "auctions", "create"), h.Auction.Create)
// 			auctions.GET("/:id", h.Auction.GetByID)
// 			auctions.PUT("/:id", middleware.RequirePermission("chits", "auctions", "update"), h.Auction.Update)
// 			auctions.PATCH("/:id/conduct", middleware.RequirePermission("chits", "auctions", "conduct"), h.Auction.Conduct)
// 			auctions.PATCH("/:id/post-result", middleware.RequirePermission("chits", "auctions", "conduct"), h.Auction.PostResult)
// 			auctions.GET("/:id/bids", h.Auction.ListBids)
// 			auctions.POST("/:id/bids", h.Auction.SubmitBid)
// 			auctions.PATCH("/:id/bids/:bidId/win", middleware.RequirePermission("chits", "auctions", "conduct"), h.Auction.MarkBidWinner)
// 		}

// 		// ── Installments & Payments ───────────────────────────────────────────
// 		installments := p.Group("/installments")
// 		installments.Use(middleware.RequirePermission("payments", "installments", "view"))
// 		{
// 			installments.GET("", h.Installment.List)
// 			installments.POST("/generate", middleware.RequirePermission("payments", "installments", "create"), h.Installment.GenerateBatch)
// 			installments.GET("/:id", h.Installment.GetByID)
// 		}

// 		subInst := p.Group("/subscriber-installments")
// 		subInst.Use(middleware.RequirePermission("payments", "installments", "view"))
// 		{
// 			subInst.GET("", h.Installment.ListSubscriberInstallments)
// 			subInst.GET("/:id", h.Installment.GetSubscriberInstallmentByID)
// 		}

// 		transactions := p.Group("/payment-transactions")
// 		transactions.Use(middleware.RequirePermission("payments", "transactions", "view"))
// 		{
// 			transactions.GET("", h.Installment.ListPaymentTransactions)
// 			transactions.POST("", middleware.RequirePermission("payments", "transactions", "create"), h.Installment.RecordPayment)
// 			transactions.GET("/:id", h.Installment.GetPaymentTransactionByID)
// 		}

// 		// ── Payouts & Dividends ───────────────────────────────────────────────
// 		payouts := p.Group("/payouts")
// 		payouts.Use(middleware.RequirePermission("payments", "payouts", "view"))
// 		{
// 			payouts.GET("", h.Payout.List)
// 			payouts.POST("", middleware.RequirePermission("payments", "payouts", "create"), h.Payout.Initiate)
// 			payouts.GET("/:id", h.Payout.GetByID)
// 			payouts.PATCH("/:id/approve", middleware.RequireRole("MANAGER", "ADMIN", "SUPER_ADMIN"), h.Payout.Approve)
// 			payouts.PATCH("/:id/disburse", middleware.RequirePermission("payments", "payouts", "disburse"), h.Payout.Disburse)
// 		}

// 		dividends := p.Group("/dividends")
// 		{
// 			dividends.GET("", h.Payout.ListDividends)
// 			dividends.POST("/calculate", middleware.RequirePermission("payments", "dividends", "create"), h.Payout.CalculateDividend)
// 			dividends.POST("/post", middleware.RequirePermission("payments", "dividends", "create"), h.Payout.PostDividend)
// 		}

// 		// ── Defaults & Legal ──────────────────────────────────────────────────
// 		defaults := p.Group("/defaults")
// 		defaults.Use(middleware.RequirePermission("legal", "defaults", "view"))
// 		{
// 			defaults.GET("", h.Default.List)
// 			defaults.POST("/detect", middleware.RequirePermission("legal", "defaults", "detect"), h.Default.Detect)
// 			defaults.PATCH("/:id/flag", middleware.RequirePermission("legal", "defaults", "update"), h.Default.Flag)
// 			defaults.PATCH("/:id/resolve", middleware.RequirePermission("legal", "defaults", "update"), h.Default.Resolve)
// 			defaults.POST("/:id/reminder", h.Default.SendReminder)
// 		}

// 		legalNotices := p.Group("/legal-notices")
// 		legalNotices.Use(middleware.RequirePermission("legal", "notices", "view"))
// 		{
// 			legalNotices.GET("", h.LegalNotice.List)
// 			legalNotices.POST("", middleware.RequirePermission("legal", "notices", "create"), h.LegalNotice.Issue)
// 			legalNotices.GET("/:id", h.LegalNotice.GetByID)
// 			legalNotices.PATCH("/:id/escalate", middleware.RequirePermission("legal", "notices", "escalate"), h.LegalNotice.Escalate)
// 		}

// 		// ── Reports & Audit ───────────────────────────────────────────────────
// 		reports := p.Group("/reports")
// 		reports.Use(middleware.RequirePermission("reports", "reports", "view"))
// 		{
// 			reports.GET("/monthly-collections", h.Report.MonthlyCollections)
// 			reports.GET("/defaulters", h.Report.DefaultersReport)
// 			reports.GET("/chit-performance", h.Report.ChitPerformance)
// 			reports.GET("/subscriber-ledger/:id", h.Report.SubscriberLedger)
// 		}

// 		p.GET("/audit-logs", middleware.RequirePermission("admin", "audit", "view"), h.Report.AuditLogs)
// 		p.GET("/system-logs", middleware.RequireRole("SUPER_ADMIN", "ADMIN"), h.Report.SystemLogs)
// 	}

// 	return r
// }
