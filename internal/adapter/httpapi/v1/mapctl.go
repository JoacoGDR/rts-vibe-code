package v1

import (
	"net/http"
	"strings"

	"github.com/joaquing/clone-supremacy/internal/adapter/httpapi/render"
	"github.com/joaquing/clone-supremacy/pkg/errs"
	"github.com/joaquing/clone-supremacy/pkg/shared/maps"
)

// MapController exposes GET /maps.
type MapController struct {
	AssetsBaseURL string
}

func NewMapController(assetsBaseURL string) *MapController {
	return &MapController{AssetsBaseURL: strings.TrimRight(assetsBaseURL, "/")}
}

// List handles GET /maps.
func (c *MapController) List(w http.ResponseWriter, _ *http.Request) {
	all, err := maps.All()
	if err != nil {
		render.Err(w, errs.Wrap(err, errs.Internal, "listing maps"))
		return
	}
	for i := range all {
		enrichMapAssets(&all[i], c.AssetsBaseURL)
	}
	render.JSON(w, http.StatusOK, all)
}

func enrichMapAssets(m *maps.Map, base string) {
	if m.SVGURL != "" {
		return
	}
	path := "/maps/" + m.ID + ".svg"
	if base != "" {
		m.SVGURL = base + path
	} else {
		m.SVGURL = path
	}
}
