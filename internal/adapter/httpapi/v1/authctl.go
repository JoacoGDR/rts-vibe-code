package v1

import (
	"encoding/json"
	"net/http"

	"github.com/joaquing/clone-supremacy/internal/adapter/httpapi/mw"
	"github.com/joaquing/clone-supremacy/internal/adapter/httpapi/render"
	"github.com/joaquing/clone-supremacy/internal/service/authsvc"
	"github.com/joaquing/clone-supremacy/pkg/api"
	"github.com/joaquing/clone-supremacy/pkg/errs"
)

// AuthController exposes the register / login / ws-ticket / me routes.
type AuthController struct {
	svc *authsvc.Service
}

func NewAuthController(svc *authsvc.Service) *AuthController {
	return &AuthController{svc: svc}
}

// Register handles POST /auth/register.
func (c *AuthController) Register(w http.ResponseWriter, r *http.Request) {
	var req api.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Err(w, errs.Wrap(err, errs.BadRequest, "invalid JSON"))
		return
	}
	out, err := c.svc.Register(r.Context(), authsvc.RegisterInput{
		Email: req.Email, Password: req.Password, DisplayName: req.DisplayName,
	})
	if err != nil {
		render.Err(w, err)
		return
	}
	render.JSON(w, http.StatusOK, api.TokenResponse{
		AccessToken: out.AccessToken,
		ExpiresAt:   out.ExpiresAt,
		User:        userView(out.User),
	})
}

// Login handles POST /auth/login.
func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	var req api.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Err(w, errs.Wrap(err, errs.BadRequest, "invalid JSON"))
		return
	}
	out, err := c.svc.Login(r.Context(), authsvc.LoginInput{
		Email: req.Email, Password: req.Password,
	})
	if err != nil {
		render.Err(w, err)
		return
	}
	render.JSON(w, http.StatusOK, api.TokenResponse{
		AccessToken: out.AccessToken,
		ExpiresAt:   out.ExpiresAt,
		User:        userView(out.User),
	})
}

// WSTicket handles POST /auth/ws-ticket. Authenticated.
func (c *AuthController) WSTicket(w http.ResponseWriter, r *http.Request) {
	claims, ok := mw.ClaimsFrom(r.Context())
	if !ok {
		render.Err(w, errs.New(errs.Unauthorized, "missing token"))
		return
	}
	ticket, err := c.svc.MintWSTicket(r.Context(), claims)
	if err != nil {
		render.Err(w, err)
		return
	}
	render.JSON(w, http.StatusOK, api.WSTicketResponse{Ticket: ticket})
}

// Me handles GET /auth/me. Authenticated.
func (c *AuthController) Me(w http.ResponseWriter, r *http.Request) {
	claims, ok := mw.ClaimsFrom(r.Context())
	if !ok {
		render.Err(w, errs.New(errs.Unauthorized, "missing token"))
		return
	}
	render.JSON(w, http.StatusOK, api.MeResponse{
		ID: claims.UserID, Email: claims.Email, DisplayName: claims.DisplayName,
	})
}

// LoginAsBot handles POST /auth/login-as-bot. Sits behind
// [mw.RequireBotKey] so only the ai-bot subsystem can call it.
func (c *AuthController) LoginAsBot(w http.ResponseWriter, r *http.Request) {
	var req api.LoginAsBotRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Err(w, errs.Wrap(err, errs.BadRequest, "invalid JSON"))
		return
	}
	out, err := c.svc.LoginAsBot(r.Context(), req.UserID)
	if err != nil {
		render.Err(w, err)
		return
	}
	render.JSON(w, http.StatusOK, api.TokenResponse{
		AccessToken: out.AccessToken,
		ExpiresAt:   out.ExpiresAt,
		User:        userView(out.User),
	})
}

// ListBots handles GET /auth/bots. Sits behind [mw.RequireBotKey] so
// only the ai-bot subsystem can enumerate the seed pool of bot
// identities. The response is intentionally minimal — full user
// records are not handed out across the trust boundary.
func (c *AuthController) ListBots(w http.ResponseWriter, r *http.Request) {
	bots, err := c.svc.ListBots(r.Context())
	if err != nil {
		render.Err(w, err)
		return
	}
	out := api.ListBotsResponse{Bots: make([]api.BotUserView, 0, len(bots))}
	for _, b := range bots {
		out.Bots = append(out.Bots, api.BotUserView{
			ID: b.ID, Email: b.Email, DisplayName: b.DisplayName,
		})
	}
	render.JSON(w, http.StatusOK, out)
}
