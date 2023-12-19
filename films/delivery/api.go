package delivery

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/configs"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/films/usecase"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/middleware"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/requests"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type API struct {
	core   usecase.ICore
	lg     *slog.Logger
	mx     *http.ServeMux
	adress string
}

func GetApi(c *usecase.Core, l *slog.Logger, cfg *configs.DbDsnCfg) *API {
	api := &API{
		core:   c,
		lg:     l.With("module", "api"),
		mx:     http.NewServeMux(),
		adress: cfg.ServerAdress,
	}

	api.mx.Handle("/metrics", promhttp.Handler())
	api.mx.HandleFunc("/api/v1/films", api.Films)
	api.mx.HandleFunc("/api/v1/film", api.Film)
	api.mx.HandleFunc("/api/v1/actor", api.Actor)
	api.mx.Handle("/api/v1/favorite/films", middleware.AuthCheck(http.HandlerFunc(api.FavoriteFilms), c, l))
	api.mx.Handle("/api/v1/favorite/film/add", middleware.AuthCheck(http.HandlerFunc(api.FavoriteFilmsAdd), c, l))
	api.mx.Handle("/api/v1/favorite/film/remove", middleware.AuthCheck(http.HandlerFunc(api.FavoriteFilmsRemove), c, l))
	api.mx.Handle("/api/v1/favorite/actors", middleware.AuthCheck(http.HandlerFunc(api.FavoriteActors), c, l))
	api.mx.Handle("/api/v1/favorite/actor/add", middleware.AuthCheck(http.HandlerFunc(api.FavoriteActorsAdd), c, l))
	api.mx.Handle("/api/v1/favorite/actor/remove", middleware.AuthCheck(http.HandlerFunc(api.FavoriteActorsRemove), c, l))
	api.mx.HandleFunc("/api/v1/find", api.FindFilm)
	api.mx.HandleFunc("/api/v1/search/actor", api.FindActor)
	api.mx.HandleFunc("/api/v1/calendar", api.Calendar)
	api.mx.Handle("/api/v1/rating/add", middleware.AuthCheck(http.HandlerFunc(api.AddRating), c, l))
	api.mx.HandleFunc("/api/v1/add/film", api.AddFilm)

	return api
}

func (a *API) ListenAndServe() {
	err := http.ListenAndServe(a.adress, a.mx)
	if err != nil {
		a.lg.Error("listen and serve error", "err", err.Error())
	}
}

func (a *API) Films(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}

	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		requests.SendResponse(w, response, a.lg)
		return
	}

	page, err := strconv.ParseUint(r.URL.Query().Get("page"), 10, 64)
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.ParseUint(r.URL.Query().Get("page_size"), 10, 64)
	if err != nil {
		pageSize = 8
	}

	genreId, err := strconv.ParseUint(r.URL.Query().Get("collection_id"), 10, 64)
	if err != nil {
		genreId = 0
	}

	var films []models.FilmItem

	films, genre, err := a.core.GetFilmsAndGenreTitle(genreId, uint64((page-1)*pageSize), pageSize)
	if err != nil {
		a.lg.Error("get films error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		requests.SendResponse(w, response, a.lg)
		return
	}

	filmsResponse := requests.FilmsResponse{
		Page:           page,
		PageSize:       pageSize,
		Total:          uint64(len(films)),
		CollectionName: genre,
		Films:          films,
	}
	response.Body = filmsResponse
    requests.SendResponse(w, response, a.lg)
}

func (a *API) Film(w http.ResponseWriter, r *http.Request) {
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

	film, err := a.core.GetFilmInfo(filmId)
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			response.Status = http.StatusNotFound
			requests.SendResponse(w, response, a.lg)
			return
		}
		a.lg.Error("film error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		requests.SendResponse(w, response, a.lg)
		return
	}

	response.Body = film
    requests.SendResponse(w, response, a.lg)
}

func (a *API) Actor(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		requests.SendResponse(w, response, a.lg)
		return
	}

	actorId, err := strconv.ParseUint(r.URL.Query().Get("actor_id"), 10, 64)
	if err != nil {
		a.lg.Error("actor error", "err", err.Error())
		response.Status = http.StatusBadRequest
		requests.SendResponse(w, response, a.lg)
		return
	}

	actor, err := a.core.GetActorInfo(actorId)
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			response.Status = http.StatusNotFound
			requests.SendResponse(w, response, a.lg)
			return
		}
		a.lg.Error("actor error", "err", err.Error())
		response.Status = http.StatusInternalServerError

		requests.SendResponse(w, response, a.lg)
		return
	}
	response.Body = actor
    requests.SendResponse(w, response, a.lg)
}

func (a *API) FindFilm(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	if r.Method != http.MethodPost {
		response.Status = http.StatusMethodNotAllowed
		requests.SendResponse(w, response, a.lg)
		return
	}

	var request requests.FindFilmRequest

	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.lg.Error("find film error", "err", err.Error())
		response.Status = http.StatusBadRequest
		requests.SendResponse(w, response, a.lg)
		return
	}

	if err = json.Unmarshal(body, &request); err != nil {
		a.lg.Error("find film error", "err", err.Error())
		response.Status = http.StatusBadRequest
		requests.SendResponse(w, response, a.lg)
		return
	}

	films, err := a.core.FindFilm(request.Title, request.DateFrom, request.DateTo, request.RatingFrom, request.RatingTo, request.Mpaa, request.Genres, request.Actors)
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			response.Status = http.StatusNotFound
			requests.SendResponse(w, response, a.lg)
			return
		}

		a.lg.Error("find film error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		requests.SendResponse(w, response, a.lg)
		return
	}

	filmsResponse := requests.FilmsResponse{
		Films: films,
		Total: uint64(len((films))),
	}
	response.Body = filmsResponse
    requests.SendResponse(w, response, a.lg)
}

func (a *API) FavoriteFilmsAdd(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		requests.SendResponse(w, response, a.lg)
		return
	}

	userId := r.Context().Value(middleware.UserIDKey).(uint64)

	filmId, err := strconv.ParseUint(r.URL.Query().Get("film_id"), 10, 64)
	if err != nil {
		response.Status = http.StatusBadRequest
		requests.SendResponse(w, response, a.lg)
		return
	}

	err = a.core.FavoriteFilmsAdd(userId, filmId)
	if err != nil {
		if errors.Is(err, usecase.ErrFoundFavorite) {
			response.Status = http.StatusNotAcceptable
			requests.SendResponse(w, response, a.lg)
			return
		}

		a.lg.Error("favorite films error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		requests.SendResponse(w, response, a.lg)
		return
	}
    requests.SendResponse(w, response, a.lg)
}

func (a *API) FavoriteFilmsRemove(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		requests.SendResponse(w, response, a.lg)
		return
	}

	userId := r.Context().Value(middleware.UserIDKey).(uint64)

	filmId, err := strconv.ParseUint(r.URL.Query().Get("film_id"), 10, 64)
	if err != nil {
		response.Status = http.StatusBadRequest
		requests.SendResponse(w, response, a.lg)
		return
	}

	err = a.core.FavoriteFilmsRemove(userId, filmId)
	if err != nil {
		a.lg.Error("favorite films error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		requests.SendResponse(w, response, a.lg)
		return
	}
    requests.SendResponse(w, response, a.lg)
}

func (a *API) FavoriteFilms(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		requests.SendResponse(w, response, a.lg)
		return
	}

	userId := r.Context().Value(middleware.UserIDKey).(uint64)

	page, err := strconv.ParseUint(r.URL.Query().Get("page"), 10, 64)
	if err != nil {
		page = 1
	}

	pageSize, err := strconv.ParseUint(r.URL.Query().Get("per_page"), 10, 64)
	if err != nil {
		pageSize = 8
	}

	films, err := a.core.FavoriteFilms(userId, uint64((page-1)*pageSize), pageSize)
	if err != nil {
		a.lg.Error("favorite films error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		requests.SendResponse(w, response, a.lg)
		return
	}
	response.Body = films
    requests.SendResponse(w, response, a.lg)
}

func (a *API) Calendar(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		requests.SendResponse(w, response, a.lg)
		return
	}

	calendar, err := a.core.GetCalendar()
	if err != nil {
		a.lg.Error("calendar error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		requests.SendResponse(w, response, a.lg)
		return
	}
	response.Body = calendar
    requests.SendResponse(w, response, a.lg)
}

func (a *API) FindActor(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	if r.Method != http.MethodPost {
		response.Status = http.StatusMethodNotAllowed
		requests.SendResponse(w, response, a.lg)
		return
	}

	var request requests.FindActorRequest

	body, err := io.ReadAll(r.Body)
	if err != nil {
		a.lg.Error("find actor error", "err", err.Error())
		response.Status = http.StatusBadRequest
		requests.SendResponse(w, response, a.lg)
		return
	}

	if err = json.Unmarshal(body, &request); err != nil {
		a.lg.Error("find actor error", "err", err.Error())
		response.Status = http.StatusBadRequest
		requests.SendResponse(w, response, a.lg)
		return
	}

	actors, err := a.core.FindActor(request.Name, request.BirthDate, request.Films, request.Career, request.Country)
	if err != nil {
		if errors.Is(err, usecase.ErrNotFound) {
			response.Status = http.StatusNotFound
			requests.SendResponse(w, response, a.lg)
			return
		}

		a.lg.Error("find actor error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		requests.SendResponse(w, response, a.lg)
		return
	}

	actorsResponse := requests.ActorsResponse{
		Actors: actors,
	}
	response.Body = actorsResponse
    requests.SendResponse(w, response, a.lg)
}

func (a *API) AddRating(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	if r.Method != http.MethodPost {
		response.Status = http.StatusMethodNotAllowed
		requests.SendResponse(w, response, a.lg)
		return
	}

	userId := r.Context().Value(middleware.UserIDKey).(uint64)

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

	found, err := a.core.AddRating(commentRequest.FilmId, userId, commentRequest.Rating)
	if err != nil {
		a.lg.Error("add rating error", "err", err.Error())
		response.Status = http.StatusInternalServerError
	}
	if found {
		response.Status = http.StatusNotAcceptable
		requests.SendResponse(w, response, a.lg)
		return
	}
    requests.SendResponse(w, response, a.lg)
}

func (a *API) AddFilm(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	if r.Method != http.MethodPost {
		response.Status = http.StatusMethodNotAllowed
		requests.SendResponse(w, response, a.lg)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		a.lg.Error("add film error", "err", err.Error())
		response.Status = http.StatusBadRequest
		requests.SendResponse(w, response, a.lg)
		return
	}

	title := r.FormValue("title")
	info := r.FormValue("info")
	date := r.FormValue("date")
	country := r.FormValue("country")

	genresString := r.FormValue("genre")
	var genres []uint64
	prev := 0
	for i := 0; i < len(genresString); i++ {
		if genresString[i] == ',' {
			genreUint, err := strconv.ParseUint(genresString[prev:i], 10, 64)
			if err != nil {
				a.lg.Error("add film error", "err", err.Error())
				response.Status = http.StatusBadRequest
				requests.SendResponse(w, response, a.lg)
				return
			}
			genres = append(genres, genreUint)
			prev = i + 1
		}
	}
	genreUint, err := strconv.ParseUint(genresString[prev:], 10, 64)
	if err != nil {
		a.lg.Error("add film error", "err", err.Error())
		response.Status = http.StatusBadRequest
		requests.SendResponse(w, response, a.lg)
		return
	}
	genres = append(genres, genreUint)
	prev = 0

	actorsString := r.FormValue("actors")
	var actors []uint64
	for i := 0; i < len(actorsString); i++ {
		if actorsString[i] == ',' {
			actorUint, err := strconv.ParseUint(actorsString[prev:i], 10, 64)
			if err != nil {
				a.lg.Error("add film error", "err", err.Error())
				response.Status = http.StatusBadRequest
				requests.SendResponse(w, response, a.lg)
				return
			}
			actors = append(actors, actorUint)
			prev = i + 1
		}
	}
	actorUint, err := strconv.ParseUint(actorsString[prev:], 10, 64)
	if err != nil {
		a.lg.Error("add film error", "err", err.Error())
		response.Status = http.StatusBadRequest
		requests.SendResponse(w, response, a.lg)
		return
	}
	actors = append(actors, actorUint)

	fmt.Println(actors, genres)
	poster, handler, err := r.FormFile("photo")
	if err != nil && !errors.Is(err, http.ErrMissingFile) {
		a.lg.Error("add film error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		requests.SendResponse(w, response, a.lg)
		return
	}

	filename := "/icons/" + handler.Filename
	if err != nil && handler != nil && poster != nil {
		a.lg.Error("Post profile error", "err", err.Error())
		response.Status = http.StatusBadRequest
		requests.SendResponse(w, response, a.lg)
		return
	}

	filePhoto, err := os.OpenFile("/home/ubuntu/frontend-project"+filename, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		a.lg.Error("add film error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		requests.SendResponse(w, response, a.lg)
		return
	}
	defer filePhoto.Close()

	_, err = io.Copy(filePhoto, poster)
	if err != nil {
		a.lg.Error("add film error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		requests.SendResponse(w, response, a.lg)
		return
	}

	film := models.FilmItem{
		Title:       title,
		Info:        info,
		Poster:      filename,
		ReleaseDate: date,
		Country:     country,
	}

	err = a.core.AddFilm(film, genres, actors)
	if err != nil {
		a.lg.Error("add film error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		requests.SendResponse(w, response, a.lg)
		return
	}
    requests.SendResponse(w, response, a.lg)
}

func (a *API) FavoriteActorsAdd(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		requests.SendResponse(w, response, a.lg)
		return
	}

	userId := r.Context().Value(middleware.UserIDKey).(uint64)

	actorId, err := strconv.ParseUint(r.URL.Query().Get("actor_id"), 10, 64)
	if err != nil {
		response.Status = http.StatusBadRequest
		requests.SendResponse(w, response, a.lg)
		return
	}

	err = a.core.FavoriteActorsAdd(userId, actorId)
	if err != nil {
		if errors.Is(err, usecase.ErrFoundFavorite) {
			response.Status = http.StatusNotAcceptable
			requests.SendResponse(w, response, a.lg)
			return
		}
		a.lg.Error("favorite actors error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		requests.SendResponse(w, response, a.lg)
		return
	}
    requests.SendResponse(w, response, a.lg)
}

func (a *API) FavoriteActorsRemove(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		requests.SendResponse(w, response, a.lg)
		return
	}

	userId := r.Context().Value(middleware.UserIDKey).(uint64)

	actorId, err := strconv.ParseUint(r.URL.Query().Get("actor_id"), 10, 64)
	if err != nil {
		response.Status = http.StatusBadRequest
		requests.SendResponse(w, response, a.lg)
		return
	}

	err = a.core.FavoriteActorsRemove(userId, actorId)
	if err != nil {
		a.lg.Error("favorite actors error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		requests.SendResponse(w, response, a.lg)
		return
	}
    requests.SendResponse(w, response, a.lg)
}

func (a *API) FavoriteActors(w http.ResponseWriter, r *http.Request) {
	response := requests.Response{Status: http.StatusOK, Body: nil}
	if r.Method != http.MethodGet {
		response.Status = http.StatusMethodNotAllowed
		requests.SendResponse(w, response, a.lg)
		return
	}

	userId := r.Context().Value(middleware.UserIDKey).(uint64)

	page, err := strconv.ParseUint(r.URL.Query().Get("page"), 10, 64)
	if err != nil {
		page = 1
	}
	pageSize, err := strconv.ParseUint(r.URL.Query().Get("per_page"), 10, 64)
	if err != nil {
		pageSize = 8
	}

	actors, err := a.core.FavoriteActors(userId, uint64((page-1)*pageSize), pageSize)
	if err != nil {
		a.lg.Error("favorite actors error", "err", err.Error())
		response.Status = http.StatusInternalServerError
		requests.SendResponse(w, response, a.lg)
		return
	}

	actorsResponse := requests.ActorsResponse{
		Actors: actors,
	}
	
	response.Body = actorsResponse
    requests.SendResponse(w, response, a.lg)
}
