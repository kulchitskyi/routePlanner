package main

import (
	"fmt"
	"os"
	"log/slog"
	"strings"
	"time"

	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberRedis "github.com/gofiber/storage/redis/v3"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/redis/go-redis/v9"

	"routePlanner/internal/auth"
	"routePlanner/internal/cache"
	"routePlanner/internal/configs"
	"routePlanner/internal/llm"
	"routePlanner/internal/places"
	"routePlanner/internal/repository"
	"routePlanner/internal/router"
	"routePlanner/internal/travelroutes"
	"routePlanner/internal/users"
	"routePlanner/pkg/db"
)

var (
	reqCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	reqDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

func init() {
	prometheus.MustRegister(
		reqCounter,
		reqDuration,
	)
}

func getLogLevel(logLevel string) slog.Level {
	switch strings.ToLower(logLevel) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func main() {
	cfg := configs.LoadConfig()
	err := configs.ValidateConfig(cfg)
	if err != nil {
		fmt.Errorf("Config validation failed: %v", err)
		os.Exit(1)
	}

	logLevel := getLogLevel(cfg.Server.LogLevel)
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)
	logger.Info("Starting application", "log_level", logLevel.String())

	gormDB := db.InitDB(cfg.DB.DatabaseURL, logger)

	redisAddr := fmt.Sprintf("%s:%s", cfg.Cache.RedisHost, cfg.Cache.RedisPort)
	logger.Info("Attempting to connect to Redis", "address", redisAddr)

	rds := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: cfg.Cache.RedisPassword,
		DB:       0,
		Protocol: 2,
	})

	fiberStore := fiberRedis.New(fiberRedis.Config{
		URL: fmt.Sprintf("redis://%s:%s@%s/1", "", cfg.Cache.RedisPassword, redisAddr),
	})

	userRepo := repository.NewPostgresUserRepository(gormDB)
	placeRepo := repository.NewPostgresPlaceRepository(gormDB)

	LLMService := llm.NewLLMService(cfg.LLM.OllamaURL, logger.With("layer", "llm_service"))

	// Trigger LLM warmup
	go func() {
		LLMService.Warmup()
	}()

	jwtService := auth.NewJWTService(cfg.Auth.JWTSecret, cfg.Auth.JWTExpirationHours)
	userService := users.NewUserService(userRepo)
	placeService := places.NewPlaceService(placeRepo)

	geoapifyClient := travelroutes.NewGeoapifyClient(
		cfg.API.GeoapifyAPIKey,
		logger.With("layer", "geoapify_client"),
		cfg.API.ReqTimeout)

	routeService := travelroutes.NewRouteService(
		placeRepo,
		logger.With("layer", "route_service"),
		cache.NewRedisCache[[]string](rds),
		LLMService,
		geoapifyClient,
		cfg.Server.CacheTagsRefreshTimeMin)

	userHandler := users.NewHandler(userService, logger.With("layer", "user_handler"))
	authHandler := users.NewAuthHandler(userService, jwtService, logger.With("layer", "auth_handler"))
	placeHandler := places.NewHandler(placeService, logger.With("layer", "place_handler"))
	routeHandler := travelroutes.NewHandler(routeService, logger.With("layer", "route_handler"))

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		Concurrency:           cfg.Server.MaxConnections,
		ReadTimeout:           time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout:          time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:           time.Duration(cfg.Server.IdleTimeout) * time.Second,
	})

	app.Use(cors.New(cors.Config{
		AllowOrigins: cfg.Server.AllowOrigins,
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Content-Type, Authorization",
	}))

	app.Use(func(c *fiber.Ctx) error {
		start  := time.Now()
		method := c.Method()
		path   := c.Route().Path

		err    := c.Next()
		status := fmt.Sprintf("%d", c.Response().StatusCode())

		reqDuration.WithLabelValues(method, path).Observe(time.Since(start).Seconds())
		reqCounter.WithLabelValues(method, path, status).Inc()
		logger.Info("Request processed", "method", method, "path", path, "status", status, "duration", time.Since(start))

		return err
	})

	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
	app.Static("/", "./public")

	router.SetUpRoutes(app, router.Config{
		UserHandler:  userHandler,
		AuthHandler:  authHandler,
		PlaceHandler: placeHandler,
		RouteHandler: routeHandler,
		JWTService:   jwtService,
	}, logger, fiberStore, cfg)

	idleConnsClosed := make(chan struct{})

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		logger.Info("Gracefully shutting down...")

		if err := app.Shutdown(); err != nil {
			logger.Error("Fiber shutdown error", "error", err)
		}

		sqlDB, err := gormDB.DB()
		if err == nil && sqlDB != nil {
			logger.Info("Closing database connection...")
			sqlDB.Close()
		}

		logger.Info("Closing redis connection...")
		rds.Close()

		logger.Info("Closing fiber store connection...")
		fiberStore.Close()

		close(idleConnsClosed)
	}()

	logger.Info("Server is starting", "port", cfg.Server.Port)
	if err := app.Listen(":" + cfg.Server.Port); err != nil {
		logger.Error("Failed to start server", "error", err)
	}

	<-idleConnsClosed
	logger.Info("Server shutdown complete")
}
