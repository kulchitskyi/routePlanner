package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v2"

	"routePlanner/internal/configs"
	er "routePlanner/internal/errors"
	"routePlanner/internal/users"
)

type OIDCHandler struct {
	oidcService *OIDCService
	appConfig   *configs.Config
	userService *users.UserService
}

func NewOIDCHandler(oidcService *OIDCService, appConfig *configs.Config, userService *users.UserService) *OIDCHandler {
	return &OIDCHandler{
		oidcService: oidcService,
		appConfig:   appConfig,
		userService: userService,
	}
}

func (h *OIDCHandler) Login(c *fiber.Ctx) error {
	cfg := h.oidcService.GetConfig()
	if cfg == nil {
		return er.InternalError(c, fmt.Errorf("OIDC config not loaded"))
	}

	authEndpoint := strings.Replace(cfg.AuthorizationEndpoint, "casdoor", "localhost", 1)

	authURL := fmt.Sprintf("%s?client_id=%s&response_type=code&redirect_uri=%s&scope=openid profile email",
		authEndpoint,
		h.appConfig.OIDC.ClientID,
		h.appConfig.OIDC.RedirectURI,
	)

	return c.Redirect(authURL, fiber.StatusTemporaryRedirect)
}

func (h *OIDCHandler) Callback(c *fiber.Ctx) error {
	code := c.Query("code")
	if code == "" {
		return er.Unauthorized(c, "missing authorization code")
	}

	cfg := h.oidcService.GetConfig()
	if cfg == nil {
		return er.InternalError(c, fmt.Errorf("OIDC config not loaded"))
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("client_id", h.appConfig.OIDC.ClientID)
	data.Set("client_secret", h.appConfig.OIDC.ClientSecret)
	data.Set("redirect_uri", h.appConfig.OIDC.RedirectURI)

	tokenURL := strings.Replace(cfg.TokenEndpoint, "localhost", "casdoor", 1)
	req, err := http.NewRequest(http.MethodPost, tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return er.InternalError(c, err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	resp, err := h.oidcService.httpClient.Do(req)
	if err != nil {
		return er.InternalError(c, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return er.InternalError(c, fmt.Errorf("failed to get token: %s", string(bodyBytes)))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return er.InternalError(c, err)
	}

	userID, claims, err := h.oidcService.ValidateToken(tokenResp.IDToken)
	if err == nil {
		email, _ := claims["email"].(string)
		name, _ := claims["preferred_username"].(string)
		if name == "" {
			name, _ = claims["name"].(string)
		}
		if name == "" {
			name = "OIDC User"
		}
		_, _ = h.userService.GetOrCreateOIDCUser(c.Context(), userID, email, name)
	}

	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    tokenResp.IDToken,
		Path:     "/",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		MaxAge:   3600,
	})

	return c.Redirect("/")
}

func (h *OIDCHandler) Logout(c *fiber.Ctx) error {
	c.Cookie(&fiber.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		MaxAge:   -1,
	})
	return c.JSON(fiber.Map{"status": "logged out"})
}

func (h *OIDCHandler) UserInfo(c *fiber.Ctx) error {
	var tokenString string
	authHeader := c.Get("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			tokenString = parts[1]
		}
	}
	if tokenString == "" {
		tokenString = c.Cookies("auth_token")
	}
	if tokenString == "" {
		return er.Unauthorized(c, "Missing authorization")
	}

	sub, claims, err := h.oidcService.ValidateToken(tokenString)
	if err != nil {
		return er.Unauthorized(c, "Invalid or expired token: "+err.Error())
	}

	return c.JSON(fiber.Map{
		"user_id": sub,
		"claims":  claims,
	})
}
