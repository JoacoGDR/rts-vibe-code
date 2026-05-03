package v1

import (
	"net/http"
	"strconv"
	"time"

	"github.com/joaquing/clone-supremacy/internal/adapter/httpapi/render"
	"github.com/joaquing/clone-supremacy/internal/domain/notifydom"
	"github.com/joaquing/clone-supremacy/internal/service/notifysvc"
	"github.com/joaquing/clone-supremacy/pkg/api"
	"github.com/joaquing/clone-supremacy/pkg/errs"
)

// NotificationController exposes the in-app notification bell endpoints.
// The worker writes to the same `notifications` table; this controller
// only reads and marks-as-read on behalf of the authenticated user.
type NotificationController struct {
	svc *notifysvc.Service
}

func NewNotificationController(svc *notifysvc.Service) *NotificationController {
	return &NotificationController{svc: svc}
}

// List handles GET /notifications?after=<rfc3339>&limit=<n>.
func (c *NotificationController) List(w http.ResponseWriter, r *http.Request) {
	uid, err := userIDFrom(r)
	if err != nil {
		render.Err(w, err)
		return
	}
	q := notifydom.ListQuery{UserID: uid, Limit: parseNotifyLimit(r)}
	if after := r.URL.Query().Get("after"); after != "" {
		t, err := time.Parse(time.RFC3339Nano, after)
		if err != nil {
			render.Err(w, errs.Wrap(err, errs.BadRequest, "bad after timestamp"))
			return
		}
		q.After = t
	}
	items, err := c.svc.List(r.Context(), q)
	if err != nil {
		render.Err(w, err)
		return
	}
	unread, err := c.svc.UnreadCount(r.Context(), uid)
	if err != nil {
		render.Err(w, err)
		return
	}
	out := api.ListNotificationsResponse{
		Items:       make([]api.NotificationView, 0, len(items)),
		UnreadCount: unread,
	}
	for _, n := range items {
		out.Items = append(out.Items, notificationView(n))
	}
	render.JSON(w, http.StatusOK, out)
}

// UnreadCount handles GET /notifications/unread-count.
func (c *NotificationController) UnreadCount(w http.ResponseWriter, r *http.Request) {
	uid, err := userIDFrom(r)
	if err != nil {
		render.Err(w, err)
		return
	}
	n, err := c.svc.UnreadCount(r.Context(), uid)
	if err != nil {
		render.Err(w, err)
		return
	}
	render.JSON(w, http.StatusOK, api.UnreadCountResponse{Count: n})
}

// MarkRead handles POST /notifications/mark-read.
func (c *NotificationController) MarkRead(w http.ResponseWriter, r *http.Request) {
	uid, err := userIDFrom(r)
	if err != nil {
		render.Err(w, err)
		return
	}
	n, err := c.svc.MarkAllRead(r.Context(), uid)
	if err != nil {
		render.Err(w, err)
		return
	}
	render.JSON(w, http.StatusOK, api.MarkReadResponse{Marked: int(n)})
}

func parseNotifyLimit(r *http.Request) int {
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

func notificationView(n notifydom.Notification) api.NotificationView {
	payload := n.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	return api.NotificationView{
		ID: n.ID, UserID: n.UserID, MatchID: n.MatchID,
		Kind:      string(n.Kind),
		Payload:   payload,
		CreatedAt: n.CreatedAt,
		SentAt:    n.SentAt,
		ReadAt:    n.ReadAt,
	}
}
