package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"
	"google.golang.org/protobuf/proto"

	"routePlanner/internal/auth"
	pb "routePlanner/internal/proto/weather"
)

type Handler struct {
	oidcService *auth.OIDCService
}

func NewHandler(oidcService *auth.OIDCService) *Handler {
	return &Handler{
		oidcService: oidcService,
	}
}

func (h *Handler) Upgrade(c *fiber.Ctx) error {
	token := c.Cookies("auth_token")
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).SendString("Missing auth cookie")
	}

	_, _, err := h.oidcService.ValidateToken(token)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("Invalid token")
	}

	if websocket.IsWebSocketUpgrade(c) {
		c.Locals("allowed", true)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

type locationMsg struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

func (h *Handler) Stream(c *websocket.Conn) {
	defer c.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	locCh := make(chan locationMsg, 1)

	go func() {
		defer cancel()
		for {
			_, msg, err := c.ReadMessage()
			if err != nil {
				return
			}

			var loc locationMsg
			if err := json.Unmarshal(msg, &loc); err != nil {
				continue
			}
			select {
			case locCh <- loc:
			default:
				select {
				case <-locCh:
				default:
				}
				locCh <- loc
			}
		}
	}()

	var ticker *time.Ticker
	var currentLoc locationMsg
	hasLocation := false

	for {
		select {
		case <-ctx.Done():
			return

		case loc := <-locCh:
			currentLoc = loc
			hasLocation = true

			if ticker != nil {
				ticker.Stop()
			}
			ticker = time.NewTicker(10 * time.Second)
			
			if data, err := fetchWeather(currentLoc.Lat, currentLoc.Lng); err == nil {
				if out, err := proto.Marshal(data); err == nil {
					if writeErr := c.WriteMessage(websocket.BinaryMessage, out); writeErr != nil {
						return
					}
				}
			}

		case <-func() <-chan time.Time {
			if ticker != nil {
				return ticker.C
			}
			return nil
		}():
			if !hasLocation {
				continue
			}
			if data, err := fetchWeather(currentLoc.Lat, currentLoc.Lng); err == nil {
				if out, err := proto.Marshal(data); err == nil {
					if writeErr := c.WriteMessage(websocket.BinaryMessage, out); writeErr != nil {
						return
					}
				}
			}
		}
	}
}

type OpenMeteoResponse struct {
	Current struct {
		Temperature float32 `json:"temperature_2m"`
		WindSpeed   float32 `json:"wind_speed_10m"`
		WeatherCode int     `json:"weather_code"`
	} `json:"current"`
}

func fetchWeather(lat, lng float64) (*pb.WeatherUpdate, error) {
	url := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current=temperature_2m,wind_speed_10m,weather_code",
		lat, lng,
	)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("open-meteo returned status %d", resp.StatusCode)
	}

	var data OpenMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &pb.WeatherUpdate{
		Latitude:    lat,
		Longitude:   lng,
		Temperature: data.Current.Temperature,
		WindSpeed:   data.Current.WindSpeed,
		Conditions:  mapWeatherCode(data.Current.WeatherCode),
		Timestamp:   time.Now().Unix(),
	}, nil
}

func mapWeatherCode(code int) string {
	if code <= 3 {
		return "Clear/Cloudy"
	} else if code <= 49 {
		return "Fog/Drizzle"
	} else if code <= 69 {
		return "Rain"
	} else if code <= 79 {
		return "Snow"
	} else {
		return "Storm"
	}
}
