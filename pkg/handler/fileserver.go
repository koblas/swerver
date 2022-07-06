package handler

import (
	"strings"

	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/koblas/swerver/pkg/swhttp"
)

func (state HandlerState) sendFile(root http.Dir) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, swhttp.FileServer(root, state.RenderSingle, !state.NoDirectoryListing))
		fs.ServeHTTP(w, r)
	}
}
