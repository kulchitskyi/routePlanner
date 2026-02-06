package configs

import (
	"fmt"
	"os"
	"reflect"
	"strconv"

	"github.com/go-playground/validator/v10"
)

type Config struct {
	Server struct {
		Port           string `env:"SERVER_PORT" default:"8080"`
		LogLevel       string `env:"LOG_LEVEL" default:"info"`
		AllowOrigins   string `env:"ALLOW_ORIGINS" validate:"required"`
		MaxConnections int    `env:"MAX_CONNECTIONS" default:"10000"`
		ReadTimeout    int    `env:"READ_TIMEOUT" default:"10"`
		WriteTimeout   int    `env:"WRITE_TIMEOUT" default:"20"`
		IdleTimeout    int    `env:"IDLE_TIMEOUT" default:"60"`

		PlaceCreateLimitPerHour int `env:"PLACE_CREATE_LIMIT_PER_HOUR" default:"100"`
		RouteBuildTotalPerDay   int `env:"ROUTE_BUILD_TOTAL_PER_DAY" default:"1000"`
		RouteBuildUserPer30S    int `env:"ROUTE_BUILD_USER_PER_30S" default:"5"`
		CacheTagsRefreshTimeMin int `env:"CACHE_TAGS_REFRESH_TIME_MIN" default:"60"`
	}
	DB struct {
		DatabaseURL string `env:"DATABASE_URL" validate:"required"`
	}
	Cache struct {
		RedisHost     string `env:"REDIS_HOST" validate:"required"`
		RedisPort     string `env:"REDIS_PORT" validate:"required"`
		RedisPassword string `env:"REDIS_PASSWORD" validate:"required"`
	}
	LLM struct {
		OllamaURL string `env:"OLLAMA_URL" validate:"required"`
	}
	API struct {
		GeoapifyAPIKey string `env:"GEOAPIFY_API_KEY" validate:"required"`
		ReqTimeout     int    `env:"GEOAPIFY_REQ_TIMEOUT" validate:"required"`
	}
	Auth struct {
		JWTSecret          string `env:"JWT_SECRET" validate:"required"`
		JWTExpirationHours int    `env:"JWT_EXPIRATION_HOURS" default:"24"`
	}
}

func LoadConfig() *Config {
	cfg := &Config{}

	if err := mapEnvToStruct(cfg); err != nil {
		panic(err)
	}
	return cfg
}

func mapEnvToStruct(ptr interface{}) error {
	v := reflect.ValueOf(ptr).Elem()
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		fieldV := v.Field(i)
		structField := t.Field(i)

		if fieldV.Kind() == reflect.Struct {
			if err := mapEnvToStruct(fieldV.Addr().Interface()); err != nil {
				return err
			}
			continue
		}

		envKey := structField.Tag.Get("env")
		if envKey == "" {
			continue
		}

		envVal := os.Getenv(envKey)
		if envVal == "" {
			envVal = structField.Tag.Get("default")
		}

		if envVal == "" {
			continue
		}
		
		switch fieldV.Kind() {
		case reflect.String:
			fieldV.SetString(envVal)
		case reflect.Int, reflect.Int64:
			val, err := strconv.Atoi(envVal)
			if err != nil {
				return fmt.Errorf("field %s: invalid int value %q", structField.Name, envVal)
			}
			fieldV.SetInt(int64(val))
		case reflect.Bool:
			boolVal, err := strconv.ParseBool(envVal)
			if err != nil {
				return fmt.Errorf("field %s: invalid bool value %q", structField.Name, envVal)
			}
			fieldV.SetBool(boolVal)
		default:
			return fmt.Errorf("field %s: unsupported type %s", structField.Name, fieldV.Kind())
		}
	}
	return nil
}

func ValidateConfig(cfg *Config) error {
	validate := validator.New()
	if err := validate.Struct(cfg); err != nil {
		return fmt.Errorf("config validation failed: %v", err)
	}
	return nil
}
