package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"github.com/gothi/vouchrs/src/internal/domain/port"
)

type googleOAuth struct {
	config *oauth2.Config
}

// NewGoogleOAuth creates a Google OAuth service for admin login.
func NewGoogleOAuth(clientID, clientSecret, redirectURL string) port.OAuthService {
	cfg := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "email", "profile"},
		Endpoint:     google.Endpoint,
	}
	return &googleOAuth{config: cfg}
}

func (g *googleOAuth) GetAuthURL(state string) string {
	return g.config.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

func (g *googleOAuth) ExchangeCode(ctx context.Context, code string) (*port.OAuthUser, error) {
	token, err := g.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("google oauth exchange: %w", err)
	}

	client := g.config.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("get google userinfo: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo error %d: %s", resp.StatusCode, body)
	}

	var info struct {
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}

	return &port.OAuthUser{
		Email:   info.Email,
		Name:    info.Name,
		Picture: info.Picture,
	}, nil
}
