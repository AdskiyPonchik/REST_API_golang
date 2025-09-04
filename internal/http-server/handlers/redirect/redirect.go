package redirect

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"go.uber.org/zap"

	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/storage"
)

// URLGetter is an interface for getting url by alias.
//
//go:generate go run github.com/vektra/mockery/v2@v2.28.2 --name=URLGetter
type URLGetter interface {
	GetURL(alias string) (string, error)
}

func New(log *zap.Logger, urlGetter URLGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.redirect.New"

		logger := log.With(
			zap.String("op", op),
			zap.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")
		if alias == "" {
			logger.Info("alias is empty")

			render.JSON(w, r, resp.Error("invalid request"))
			return
		}

		resURL, err := urlGetter.GetURL(alias)
		if errors.Is(err, storage.ErrUrlNotFound) {
			logger.Info("url not found", zap.String("alias", alias))

			render.JSON(w, r, resp.Error("not found"))
			return
		}
		if err != nil {
			logger.Error("failed to get url", zap.Error(err))

			render.JSON(w, r, resp.Error("internal error"))
			return
		}

		logger.Info("got url", zap.String("url", resURL))
		http.Redirect(w, r, resURL, http.StatusFound)
	}
}
