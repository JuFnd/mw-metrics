package delivery

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/comments/usecase"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/configs"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/requests"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type API struct {
	core   usecase.ICore
	lg     *slog.Logger
	mx     *http.ServeMux
	adress string
}

func GetApi(c *usecase.Core, l *slog.Logger, cfg *configs.CommentCfg) *API {
	
	api := &API{
		core:   c,
		lg:     l.With("module", "api"),
		mx:     http.NewServeMux(),
		adress: cfg.ServerAdress,
	}

	api.mx.Handle("/metrics", promhttp.Handler())
	api.mx.HandleFunc("/api/v1/comment", api.Comment)
	api.mx.HandleFunc("/api/v1/comment/add", api.AddComment)

	return api
}

func (a *API) ListenAndServe() {
	err := http.ListenAndServe(a.adress, a.mx)
	if err != nil {
		a.lg.Error("listen and serve error", "err", err.Error())
	}
}

func (a *API) Comment(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	if r.Method != http.MethodGet {
		
		response.Status = http.StatusMethodNotAllowed
		requests.SendResponse(w, response, a.lg)
		return
	}

	filmId, err := strconv.ParseUint(r.URL.Query().Get("film_id"), 10, 64)
	if err != nil {
		
		response.Status = http.StatusBadRequest
		requests.SendResponse(w, response, a.lg)
		return
	}
	page, err := strconv.ParseUint(r.URL.Query().Get("page"), 10, 64)
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.ParseUint(r.URL.Query().Get("per_page"), 10, 64)
	if err != nil {
		pageSize = 10
	}

	comments, err := a.core.GetFilmComments(filmId, (page-1)*pageSize, pageSize)
	if err != nil {
		a.lg.Error("Comment", "err", err.Error())
		
		response.Status = http.StatusInternalServerError
		requests.SendResponse(w, response, a.lg)
		return
	}

	commentsResponse := requests.CommentResponse{Comments: comments}

	
	response.Body = commentsResponse
    requests.SendResponse(w, response, a.lg)
}

func (a *API) AddComment(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	if r.Method != http.MethodPost {
		
		response.Status = http.StatusMethodNotAllowed
		requests.SendResponse(w, response, a.lg)
		return
	}

	session, err := r.Cookie("session_id")
	if err == http.ErrNoCookie {
		
		response.Status = http.StatusUnauthorized
		requests.SendResponse(w, response, a.lg)
		return
	}
	if err != nil {
		a.lg.Error("Add comment error", "err", err.Error())
		
		response.Status = http.StatusInternalServerError
		requests.SendResponse(w, response, a.lg)
		return
	}

	userId, err := a.core.GetUserId(r.Context(), session.Value)
	if err != nil {
		a.lg.Error("Add comment error", "err", err.Error())
		
		response.Status = http.StatusInternalServerError
		requests.SendResponse(w, response, a.lg)
		return
	}

	var commentRequest requests.CommentRequest

	body, err := io.ReadAll(r.Body)
	if err != nil {
		
		response.Status = http.StatusBadRequest
		requests.SendResponse(w, response, a.lg)
		return
	}

	if err = json.Unmarshal(body, &commentRequest); err != nil {
		
		response.Status = http.StatusBadRequest
		requests.SendResponse(w, response, a.lg)
		return
	}

	found, err := a.core.AddComment(commentRequest.FilmId, userId, commentRequest.Rating, commentRequest.Text)
	if err != nil {
		a.lg.Error("Add Comment error", "err", err.Error())
		response.Status = http.StatusInternalServerError
	}
	if found {
		
		response.Status = http.StatusNotAcceptable
		requests.SendResponse(w, response, a.lg)
		return
	}
    requests.SendResponse(w, response, a.lg)
}
