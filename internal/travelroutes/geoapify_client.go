package travelroutes

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"routePlanner/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	GeoapiCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "travelroutes",
			Subsystem: "geoapify",
			Name:      "requests_total",
			Help:      "Total number of requests to Geoapify API",
		},
	)
)

func init() {
	prometheus.MustRegister(GeoapiCounter)
}

type GeoapifyClient struct {
	apikey  string
	logger  *slog.Logger
	timeout time.Duration
}

func NewGeoapifyClient(apikey string, logger *slog.Logger, timeout int) *GeoapifyClient {
	return &GeoapifyClient{
		apikey:  apikey,
		logger:  logger,
		timeout: time.Duration(timeout) * time.Second,
	}
}

func (c *GeoapifyClient) BuildRoute(ctx context.Context, origin, destination models.Location) (string, error) {
	url := fmt.Sprintf(
		"https://api.geoapify.com/v1/routing?waypoints=%f,%f|%f,%f&mode=walk&apiKey=%s",
		origin.Latitude,
		origin.Longitude,
		destination.Latitude,
		destination.Longitude,
		c.apikey,
	)

	c.logger.Debug("Requesting route from Geoapify",
		"origin", fmt.Sprintf("%f,%f", origin.Latitude, origin.Longitude),
		"destination", fmt.Sprintf("%f,%f", destination.Latitude, destination.Longitude),
	)
	timeout := c.timeout
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < 0 {
			return "", context.DeadlineExceeded
		}
		if remaining < timeout {
			timeout = remaining
		}
	}

	agent := fiber.Get(url).Timeout(timeout)

	GeoapiCounter.Inc()

	status, body, errs := agent.Bytes()

	if len(errs) > 0 {
		return "", errs[0]
	}

	if status != fiber.StatusOK {
		return "", fmt.Errorf("geoapify returned status: %d", status)
	}
	return string(body), nil
}
