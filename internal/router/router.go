package router

import (
	"github.com/gofiber/fiber/v2"
	"log/slog"

	"routePlanner/internal/auth"
	"routePlanner/internal/configs"
	"routePlanner/internal/places"
	"routePlanner/internal/travelroutes"
	"routePlanner/internal/users"
)

type Config struct {
	UserHandler  *users.Handler
	AuthHandler  *users.AuthHandler
	PlaceHandler *places.Handler
	RouteHandler *travelroutes.Handler
	JWTService   *auth.JWTService
}

func SetUpRoutes(
	app *fiber.App,
	cfg Config,
	logger *slog.Logger,
	storage fiber.Storage,
	appConfig *configs.Config,
) {
	if appConfig == nil {
		panic("appConfig cannot be nil")
	}

	if cfg.PlaceHandler == nil || cfg.UserHandler == nil || cfg.RouteHandler == nil || cfg.AuthHandler == nil || cfg.JWTService == nil {
		panic("all handlers in Config must be non-nil")
	}

	api := app.Group("/api")
	v1 := api.Group("/v1")

	places.RegisterRoutes(v1, cfg.PlaceHandler, logger, storage, appConfig.Server.PlaceCreateLimitPerHour, cfg.JWTService)
	users.RegisterRoutes(v1, cfg.UserHandler, cfg.AuthHandler, logger, cfg.JWTService)
	travelroutes.RegisterRoutes(v1,
		cfg.RouteHandler,
		logger,
		storage,
		appConfig.Server.RouteBuildTotalPerDay,
		appConfig.Server.RouteBuildUserPer30S)
}
