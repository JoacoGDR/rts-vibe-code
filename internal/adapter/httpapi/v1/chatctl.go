package v1

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/joaquing/clone-supremacy/internal/adapter/httpapi/render"
	"github.com/joaquing/clone-supremacy/internal/domain/chatdom"
	"github.com/joaquing/clone-supremacy/internal/service/chatsvc"
	"github.com/joaquing/clone-supremacy/pkg/api"
	"github.com/joaquing/clone-supremacy/pkg/errs"
)

// ChatController exposes the REST chat surface: send (fallback when WS
// is not available) and history.
type ChatController struct {
	svc *chatsvc.Service
}

func NewChatController(svc *chatsvc.Service) *ChatController {
	return &ChatController{svc: svc}
}

// Send handles POST /matches/{matchID}/chat. The WS path is preferred —
// this exists so curl, integration tests and chat-only clients can
// participate without a WebSocket session.
func (c *ChatController) Send(w http.ResponseWriter, r *http.Request) {
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
	var req api.SendChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		render.Err(w, errs.Wrap(err, errs.BadRequest, "invalid JSON"))
		return
	}
	msg, err := c.svc.Send(r.Context(), chatsvc.SendInput{
		MatchID:    matchID,
		AuthorID:   uid,
		AuthorSlot: req.AuthorSlot,
		Scope:      req.Scope,
		Body:       req.Body,
	})
	if err != nil {
		render.Err(w, err)
		return
	}
	render.JSON(w, http.StatusCreated, chatMessageView(msg))
}

// History handles GET /matches/{matchID}/chat?scope=...&before=...&limit=...
func (c *ChatController) History(w http.ResponseWriter, r *http.Request) {
	matchID, err := uuid.Parse(chi.URLParam(r, "matchID"))
	if err != nil {
		render.Err(w, errs.Wrap(err, errs.BadRequest, "bad match id"))
		return
	}
	scope := r.URL.Query().Get("scope")
	if scope == "" {
		render.Err(w, errs.New(errs.BadRequest, "scope query param required"))
		return
	}

	q := chatdom.HistoryQuery{MatchID: matchID, Scope: scope, Limit: parseLimit(r)}
	if before := r.URL.Query().Get("before"); before != "" {
		t, err := time.Parse(time.RFC3339Nano, before)
		if err != nil {
			render.Err(w, errs.Wrap(err, errs.BadRequest, "bad before timestamp"))
			return
		}
		q.Before = t
	}

	msgs, err := c.svc.History(r.Context(), q)
	if err != nil {
		render.Err(w, err)
		return
	}
	out := make([]api.ChatMessageView, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, chatMessageView(m))
	}
	render.JSON(w, http.StatusOK, out)
}

func parseLimit(r *http.Request) int {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return 0
	}
	return n
}

func chatMessageView(m chatdom.Message) api.ChatMessageView {
	return api.ChatMessageView{
		ID: m.ID, MatchID: m.MatchID,
		Scope:      m.Scope,
		AuthorID:   m.AuthorID,
		AuthorSlot: m.AuthorSlot,
		Body:       m.Body,
		SentAt:     m.SentAt,
	}
}
