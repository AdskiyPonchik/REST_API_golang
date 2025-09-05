package delete

import (
	"errors"
	"net/http"

	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/storage"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type Request struct {
	URL   string `json:"url" validate:"required,url"`
	Alias string `json:"alias,omitempty"`
}

type Response struct {
	resp.Response
	Alias string `json:"alias,omitempty"`
}

// TODO: move to config
const aliasLength = 6

// INFO: it doesn't work. Mock files were generated manually
//
//go:generate go run github.com/vektra/mockery/v2@v2.28.2 --name=URLDeleter

type DeleteURL interface {
	DeleteURL(alias string) error
}

func New(log *zap.Logger, urlDeleter DeleteURL) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.delete.New"

		log = log.With(
			zap.String("op", op),
			zap.String("request_id", middleware.GetReqID(r.Context())),
		)

		var req Request
		err := render.DecodeJSON(r.Body, &req)
		if err != nil {
			log.Error("failed to decode request body", zap.Error(err))
			render.JSON(w, r, resp.Error("failed to decode request"))
			return
		}
		log.Info("request body decoded", zap.Any("request", req))

		if err := validator.New().Struct(req); err != nil {
			validateErr := err.(validator.ValidationErrors)

			log.Error("invalid request", zap.Error(err))

			render.JSON(w, r, resp.Error("invalid request"))
			render.JSON(w, r, resp.ValidationError(validateErr))

			return
		}

		alias := req.Alias
		if alias == "" {
			log.Error("alias can't be empty", zap.Error(err))
			render.JSON(w, r, resp.Error("alias can't be empty"))
		}

		err = urlDeleter.DeleteURL(alias)

		if errors.Is(err, storage.ErrUrlNotFound) {
			log.Info("url not found", zap.String("url", req.URL))
			render.JSON(w, r, resp.Error("url not found"))
			return
		}

		if err != nil {
			log.Error("failed to delete url", zap.Error(err))

			render.JSON(w, r, resp.Error("failed to delete url"))

			return
		}
		log.Info("url deleted", zap.String("alias: ", alias))
		responseOK(w, r, alias)
	}
}

func responseOK(w http.ResponseWriter, r *http.Request, alias string) {
	render.JSON(w, r, Response{
		Response: resp.OK(),
		Alias:    alias,
	})
}
