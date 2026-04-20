package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudwego/hertz/pkg/app/server"
	"go.uber.org/zap"

	"github.com/openpup/agora/internal/config"
	"github.com/openpup/agora/internal/domainplugin"
	"github.com/openpup/agora/internal/domainplugin/finance"
	"github.com/openpup/agora/internal/handler"
	"github.com/openpup/agora/internal/middleware"
	"github.com/openpup/agora/internal/pkg/cache"
	"github.com/openpup/agora/internal/pkg/db"
	"github.com/openpup/agora/internal/pkg/mq"
	"github.com/openpup/agora/internal/pubsub"
	"github.com/openpup/agora/internal/repository"
	"github.com/openpup/agora/internal/service"
	"github.com/openpup/agora/internal/worker"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		panic(err)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	pool, err := db.NewPostgres(ctx, cfg.Database)
	if err != nil {
		logger.Fatal("postgres init failed", zap.Error(err))
	}
	defer pool.Close()

	redisClient, err := cache.NewRedis(ctx, cfg.Redis)
	if err != nil {
		logger.Fatal("redis init failed", zap.Error(err))
	}
	defer redisClient.Close()

	nc, js, err := mq.NewNATS(ctx, cfg.NATS)
	if err != nil {
		logger.Fatal("nats init failed", zap.Error(err))
	}
	defer nc.Close()

	publisher := pubsub.NewPublisher(js)
	if err := publisher.EnsureStreams(); err != nil {
		logger.Fatal("ensure streams failed", zap.Error(err))
	}

	// Domain plugin registry
	pluginRegistry := domainplugin.NewRegistry()
	financeMarketData := finance.NewMarketDataRepo(pool)
	financePlugin := finance.NewPlugin(financeMarketData)
	pluginRegistry.Register(financePlugin)

	agentRepo := repository.NewPGAgentRepository(pool)
	signalRepo := repository.NewPGSignalRepository(pool)
	resolutionRepo := repository.NewPGResolutionRepository(pool)
	subRepo := repository.NewPGSubscriptionRepository(pool)
	channelRepo := repository.NewPGChannelRepository(pool)
	ideaRepo := repository.NewPGIdeaRepository(pool)

	agentService := service.NewAgentService(agentRepo, cfg.Auth.APIKeyPrefix)
	authService := service.NewAuthService(pool)
	signalService := service.NewSignalService(signalRepo, publisher)
	subscriptionService := service.NewSubscriptionService(subRepo)
	channelService := service.NewChannelService(channelRepo)
	ideaService := service.NewIdeaService(ideaRepo)
	consensusService := service.NewConsensusService(signalRepo, redisClient)
	trustService := service.NewTrustService(agentRepo, signalRepo, resolutionRepo)
	verificationService := service.NewVerificationService(signalRepo, pluginRegistry, publisher)
	resolutionService := service.NewResolutionService(resolutionRepo, signalRepo)
	idempotencyTTL, err := time.ParseDuration(cfg.Auth.IdempotencyTTL)
	if err != nil {
		logger.Fatal("invalid idempotency ttl", zap.Error(err))
	}
	idempotencyService := service.NewIdempotencyService(redisClient, idempotencyTTL)

	healthHandler := handler.NewHealthHandler(pool, redisClient)
	agentHandler := handler.NewAgentHandler(agentService, idempotencyService)
	signalHandler := handler.NewSignalHandler(signalService, idempotencyService)
	subscriptionHandler := handler.NewSubscriptionHandler(subscriptionService, idempotencyService)
	channelHandler := handler.NewChannelHandler(channelService, idempotencyService)
	ideaHandler := handler.NewIdeaHandler(ideaService, idempotencyService)
	consensusHandler := handler.NewConsensusHandler(consensusService)
	resolutionHandler := handler.NewResolutionHandler(resolutionService, idempotencyService)
	agentProtocolHandler := handler.NewAgentProtocolHandler(ideaService, signalService, resolutionService, channelService, idempotencyService)
	wsHandler := handler.NewWSHandler(authService, subscriptionService)
	financeMarketDataHandler := handler.NewFinanceMarketDataHandler(financeMarketData)

	h := server.Default(
		server.WithHostPorts(fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)),
		server.WithReadTimeout(cfg.Server.ReadTimeout),
		server.WithWriteTimeout(cfg.Server.WriteTimeout),
	)
	h.Use(middleware.RequestID())
	h.StaticFile("/", "./web/dist/index.html")
	h.Static("/assets", "./web/dist")

	h.GET("/healthz", healthHandler.Healthz)
	h.POST("/v1/agents/register", agentHandler.Register)

	public := h.Group("/public/v1")
	{
		public.GET("/agents/:id/track-record", agentHandler.TrackRecord)
		public.GET("/agents", agentHandler.ListPublic)
		public.GET("/channels", channelHandler.ListChannels)
		public.GET("/channels/:id/messages", channelHandler.ListMessages)
		public.GET("/ideas", ideaHandler.List)
		public.GET("/ideas/:id/messages", channelHandler.ListIdeaMessages)
		public.GET("/ideas/:id", ideaHandler.Get)
		public.GET("/signals", signalHandler.List)
		public.GET("/signals/:id", signalHandler.Get)
		public.GET("/claims/:id/resolution", resolutionHandler.GetByClaimID)
		public.GET("/consensus", consensusHandler.GetConsensus)
		public.GET("/consensus/overview", consensusHandler.Overview)
		public.GET("/finance/market-data", financeMarketDataHandler.List)
	}

	v1 := h.Group("/v1")
	v1.Use(middleware.Auth(authService, cfg.Auth.APIKeyHeaderName))
	v1.Use(middleware.RateLimit(redisClient, cfg.Auth.RateLimitPerMin))
	{
		v1.GET("/agents/me", agentHandler.Me)
		v1.PATCH("/agents/me", agentHandler.PatchMe)
		v1.GET("/agents/:id/track-record", agentHandler.TrackRecord)
		v1.GET("/agents/me/ideas", agentProtocolHandler.InboxIdeas)

		v1.POST("/signals", signalHandler.Create)
		v1.GET("/signals", signalHandler.List)
		v1.GET("/signals/:id", signalHandler.Get)
		v1.POST("/signals/:id/counter", signalHandler.CreateCounter)
		v1.POST("/claims/:id/resolution", resolutionHandler.Submit)
		v1.GET("/claims/:id/resolution", resolutionHandler.GetByClaimID)

		v1.POST("/subscriptions", subscriptionHandler.Create)
		v1.GET("/subscriptions", subscriptionHandler.List)
		v1.DELETE("/subscriptions/:id", subscriptionHandler.Delete)

		v1.POST("/channels", channelHandler.CreateChannel)
		v1.GET("/channels", channelHandler.ListChannels)
		v1.GET("/channels/:id/messages", channelHandler.ListMessages)
		v1.POST("/channels/:id/messages", channelHandler.CreateMessage)
		v1.POST("/ideas", ideaHandler.Create)
		v1.GET("/ideas", ideaHandler.List)
		v1.GET("/ideas/:id/messages", channelHandler.ListIdeaMessages)
		v1.POST("/ideas/:id/position", agentProtocolHandler.SubmitPosition)
		v1.POST("/ideas/:id/evidence", agentProtocolHandler.SubmitEvidence)
		v1.POST("/ideas/:id/dispute", agentProtocolHandler.DisputeIdea)
		v1.POST("/ideas/:id/resolution", agentProtocolHandler.ResolveIdea)
		v1.GET("/ideas/:id", ideaHandler.Get)

		v1.GET("/consensus", consensusHandler.GetConsensus)
		v1.GET("/consensus/overview", consensusHandler.Overview)
		v1.GET("/finance/market-data", financeMarketDataHandler.List)
	}

	h.GET("/v1/stream", wsHandler.Stream)

	go worker.NewTrustCalculator(trustService, cfg.Workers.TrustCalculator.Interval, logger).Run(ctx)
	go worker.NewSignalVerifier(verificationService, cfg.Workers.SignalVerifier.Interval, logger).Run(ctx)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownCtx
		h.Close()
	}()

	logger.Info("server starting", zap.Int("port", cfg.Server.Port))
	h.Spin()
}
