package v1

import (
	"github.com/joaquing/clone-supremacy/internal/domain/userdom"
	"github.com/joaquing/clone-supremacy/internal/service/lobbysvc"
	"github.com/joaquing/clone-supremacy/pkg/api"
)

// userView maps a domain user onto the public API projection.
func userView(u userdom.User) api.UserView {
	return api.UserView{
		ID: u.ID, Email: u.Email, DisplayName: u.DisplayName, Color: u.Color,
	}
}

// matchView maps a service-level lobby view onto the public API projection.
func matchView(view lobbysvc.MatchView) api.MatchView {
	out := api.MatchView{
		ID: view.Match.ID, Name: view.Match.Name, MapID: view.Match.MapID,
		Status:    string(view.Match.Status),
		CreatedAt: view.Match.CreatedAt,
		StartedAt: view.Match.StartedAt,
		EndedAt:   view.Match.EndedAt,
		WinnerID:  view.Match.WinnerUserID,
	}
	out.Players = make([]api.PlayerView, 0, len(view.Players))
	for _, p := range view.Players {
		out.Players = append(out.Players, api.PlayerView{
			UserID: p.UserID, Slot: p.Slot, Color: p.Color, Alive: p.Alive,
			ControlledByAI: p.ControlledByAI,
		})
	}
	return out
}

// matchViews maps a slice in one go.
func matchViews(views []lobbysvc.MatchView) []api.MatchView {
	out := make([]api.MatchView, 0, len(views))
	for _, v := range views {
		out = append(out, matchView(v))
	}
	return out
}
