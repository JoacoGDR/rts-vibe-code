package aibot

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/adapter/httpapi/mw"
	"github.com/joaquing/clone-supremacy/pkg/api"
)

// ErrNoBotsAvailable is returned by [CoreClient.PickBot] when the
// service-account pool is empty. The session loop treats it as a fatal
// configuration error.
var ErrNoBotsAvailable = errors.New("no bot accounts available")

// CoreClient is the HTTP wrapper the ai-bot mode uses to talk to
// core-api: list bot accounts, exchange them for JWTs, then mint
// short-lived WS tickets. Each bot session owns one CoreClient so the
// Authorization header can be swapped per identity.
type CoreClient struct {
	baseURL string
	apiKey  string
	http    *http.Client

	mu    *muRef
	token string
}

// NewCoreClient builds a client pointed at the given core-api base URL.
// The shared http.Client has a 10s timeout — bot calls are tiny REST
// hops, no point letting them hang forever.
func NewCoreClient(baseURL, apiKey string) *CoreClient {
	return &CoreClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		http:    &http.Client{Timeout: 10 * time.Second},
		mu:      &muRef{},
	}
}

// muRef is a tiny indirection so two CoreClient instances that share an
// underlying *http.Client can also share their token without races on
// the embedded mutex. Today we don't need it, but the indirection keeps
// the door open for connection pooling.
type muRef struct{}

// ListBots fetches the seed pool of bot identities. Goes through the
// bot-key-protected endpoint.
func (c *CoreClient) ListBots(ctx context.Context) ([]api.BotUserView, error) {
	var out api.ListBotsResponse
	if err := c.do(ctx, http.MethodGet, "/api/v1/auth/bots", nil, &out, withBotKey(c.apiKey)); err != nil {
		return nil, err
	}
	return out.Bots, nil
}

// LoginAsBot exchanges a bot identity for a JWT. The client stashes
// the token internally so subsequent calls can reuse it.
func (c *CoreClient) LoginAsBot(ctx context.Context, botID uuid.UUID) (api.TokenResponse, error) {
	body := api.LoginAsBotRequest{UserID: botID}
	var out api.TokenResponse
	if err := c.do(ctx, http.MethodPost, "/api/v1/auth/login-as-bot", body, &out, withBotKey(c.apiKey)); err != nil {
		return api.TokenResponse{}, err
	}
	c.token = out.AccessToken
	return out, nil
}

// MintWSTicket trades the cached JWT for a single-use WS ticket.
func (c *CoreClient) MintWSTicket(ctx context.Context) (string, error) {
	if c.token == "" {
		return "", errors.New("no JWT cached; call LoginAsBot first")
	}
	var out api.WSTicketResponse
	if err := c.do(ctx, http.MethodPost, "/api/v1/auth/ws-ticket", nil, &out, withBearer(c.token)); err != nil {
		return "", err
	}
	return out.Ticket, nil
}

// requestOpt mutates a request before it leaves. We use it to attach the
// right authentication header (bot-key vs Bearer) per call.
type requestOpt func(*http.Request)

func withBotKey(key string) requestOpt {
	return func(r *http.Request) { r.Header.Set(mw.BotKeyHeader, key) }
}

func withBearer(token string) requestOpt {
	return func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+token) }
}

func (c *CoreClient) do(ctx context.Context, method, path string, body, out any, opts ...requestOpt) error {
	var rd io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode body: %w", err)
		}
		rd = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, rd)
	if err != nil {
		return err
	}
	if rd != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for _, o := range opts {
		o(req)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("core-api %s %s: %d %s", method, path, resp.StatusCode, string(raw))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
