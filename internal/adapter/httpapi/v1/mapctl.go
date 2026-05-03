package v1

import (
	"net/http"

	"github.com/joaquing/clone-supremacy/internal/adapter/httpapi/render"
	"github.com/joaquing/clone-supremacy/pkg/errs"
	"github.com/joaquing/clone-supremacy/pkg/shared/maps"
)

// MapController exposes GET /maps.
type MapController struct{}

func NewMapController() *MapController { return &MapController{} }

// List handles GET /maps.
func (c *MapController) List(w http.ResponseWriter, _ *http.Request) {
	all, err := maps.All()
	if err != nil {
		render.Err(w, errs.Wrap(err, errs.Internal, "listing maps"))
		return
	}
	render.JSON(w, http.StatusOK, all)
}
