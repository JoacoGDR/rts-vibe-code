package v1

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/adapter/httpapi/mw"
	"github.com/joaquing/clone-supremacy/internal/adapter/httpapi/render"
	"github.com/joaquing/clone-supremacy/internal/service/lobbysvc"
	"github.com/joaquing/clone-supremacy/pkg/api"
	"github.com/joaquing/clone-supremacy/pkg/errs"
)

// MatchController exposes the lobby routes (create / join / start / list /
// get).
type MatchController struct {
	svc *lobbysvc.Service
}

func NewMatchController(svc *lobbysvc.Service) *MatchController {
	return &MatchController{svc: svc}
}

func userIDFrom(r *http.Request) (uuid.UUID, error) {
	claims, ok := mw.ClaimsFrom(r.Context())
	if !ok {
		return uuid.Nil, errs.New(errs.Unauthorized, "missing token")
	}
	uid, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, errs.Wrap(err, errs.BadRequest, "bad subject")
	}
	return uid, nil
}

// Create handles POST /matches.
func (c *MatchController) Create(w http.ResponseWriter, r *http.Request) {
	uid, err := userIDFrom(r)
	if err != nil {
		render.Err(w, err)
		return
	}
	var req api.CreateMatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Err(w, errs.Wrap(err, errs.BadRequest, "invalid JSON"))
		return
	}
	view, err := c.svc.Create(r.Context(), lobbysvc.CreateInput{
		Name: req.Name, MapID: req.MapID, Slot: req.Slot, UserID: uid,
	})
	if err != nil {
		render.Err(w, err)
		return
	}
	render.JSON(w, http.StatusCreated, matchView(view))
}

// Join handles POST /matches/{matchID}/join.
func (c *MatchController) Join(w http.ResponseWriter, r *http.Request) {
	uid, err := userIDFrom(r)
	if err != nil {
		render.Err(w, err)
		return
	}
	matchID, err := uuid.Parse(chi.URLParam(r, "matchID"))
	if err != nil {
		render.Err(w, errs.Wrap(err, errs.BadRequest, "bad match id"))
		return
	}
	var req api.JoinMatchRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	out, err := c.svc.Join(r.Context(), lobbysvc.JoinInput{
		MatchID: matchID, UserID: uid, Slot: req.Slot,
	})
	if err != nil {
		render.Err(w, err)
		return
	}
	if out.AlreadyJoined {
		render.JSON(w, http.StatusOK, api.JoinMatchResponse{
			Status: "already_joined", Slot: out.JoinedSlot,
		})
		return
	}
	render.JSON(w, http.StatusOK, matchView(out.View))
}

// Start handles POST /matches/{matchID}/start.
func (c *MatchController) Start(w http.ResponseWriter, r *http.Request) {
	matchID, err := uuid.Parse(chi.URLParam(r, "matchID"))
	if err != nil {
		render.Err(w, errs.Wrap(err, errs.BadRequest, "bad match id"))
		return
	}
	view, err := c.svc.Start(r.Context(), matchID)
	if err != nil {
		render.Err(w, err)
		return
	}
	render.JSON(w, http.StatusOK, matchView(view))
}

// List handles GET /matches.
func (c *MatchController) List(w http.ResponseWriter, r *http.Request) {
	uid, err := userIDFrom(r)
	if err != nil {
		render.Err(w, err)
		return
	}
	views, err := c.svc.ListLobby(r.Context(), uid)
	if err != nil {
		render.Err(w, err)
		return
	}
	render.JSON(w, http.StatusOK, matchViews(views))
}

// Get handles GET /matches/{matchID}.
func (c *MatchController) Get(w http.ResponseWriter, r *http.Request) {
	matchID, err := uuid.Parse(chi.URLParam(r, "matchID"))
	if err != nil {
		render.Err(w, errs.Wrap(err, errs.BadRequest, "bad match id"))
		return
	}
	view, err := c.svc.Get(r.Context(), matchID)
	if err != nil {
		render.Err(w, err)
		return
	}
	render.JSON(w, http.StatusOK, matchView(view))
}
