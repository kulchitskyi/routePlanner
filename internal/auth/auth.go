package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid or expired token")
)

type OIDCConfig struct {
	Issuer                string
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	JwksURI               string `json:"jwks_uri"`
	UserinfoEndpoint      string `json:"userinfo_endpoint"`
}

type JWKS struct {
	Keys []JWK `json:"keys"`
}

type JWK struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type OIDCService struct {
	config     *OIDCConfig
	jwks       *JWKS
	publicKeys map[string]*rsa.PublicKey
	mu         sync.RWMutex
	httpClient *http.Client
}

func internalURL(urlStr string) string {
	return strings.Replace(urlStr, "localhost", "casdoor", 1)
}

func NewOIDCService(issuer string) (*OIDCService, error) {
	s := &OIDCService{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		publicKeys: make(map[string]*rsa.PublicKey),
	}

	discoveryURL := internalURL(fmt.Sprintf("%s/.well-known/openid-configuration", issuer))
	resp, err := s.httpClient.Get(discoveryURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch openid configuration from %s: %v", discoveryURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch openid configuration, status: %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var config OIDCConfig
	if err := json.Unmarshal(body, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal openid configuration: %v", err)
	}
	config.Issuer = issuer
	s.config = &config

	if err := s.refreshJWKS(); err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %v", err)
	}

	return s, nil
}

func (s *OIDCService) GetConfig() *OIDCConfig {
	return s.config
}

func (s *OIDCService) refreshJWKS() error {
	resp, err := s.httpClient.Get(internalURL(s.config.JwksURI))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to fetch JWKS, status: %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var jwks JWKS
	if err := json.Unmarshal(body, &jwks); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.jwks = &jwks

	for _, key := range jwks.Keys {
		if key.Kty == "RSA" && key.Use == "sig" {
			nBytes, err := base64.RawURLEncoding.DecodeString(key.N)
			if err != nil {
				continue
			}
			eBytes, err := base64.RawURLEncoding.DecodeString(key.E)
			if err != nil {
				continue
			}

			n := new(big.Int).SetBytes(nBytes)
			e := 0
			for _, b := range eBytes {
				e = e<<8 + int(b)
			}

			pubKey := &rsa.PublicKey{N: n, E: e}
			s.publicKeys[key.Kid] = pubKey
		}
	}

	return nil
}

func (s *OIDCService) getPublicKey(kid string) (*rsa.PublicKey, error) {
	s.mu.RLock()
	key, ok := s.publicKeys[kid]
	s.mu.RUnlock()

	if ok {
		return key, nil
	}

	if err := s.refreshJWKS(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	key, ok = s.publicKeys[kid]
	s.mu.RUnlock()

	if ok {
		return key, nil
	}
	return nil, fmt.Errorf("public key for kid %s not found", kid)
}

func (s *OIDCService) ValidateToken(tokenString string) (string, jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, fmt.Errorf("kid header not found")
		}

		return s.getPublicKey(kid)
	})

	if err != nil || !token.Valid {
		return "", nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", nil, ErrInvalidToken
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return "", nil, ErrInvalidToken
	}

	userID := sub
	if idClaim, ok := claims["id"].(string); ok && idClaim != "" {
		userID = idClaim
	}

	if len(userID) != 36 {
		hash := []byte(sub)
		for len(hash) < 16 {
			hash = append(hash, hash...)
		}
		userID = fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			hash[0:4], hash[4:6], hash[6:8], hash[8:10], hash[10:16])
	}

	return userID, claims, nil
}
