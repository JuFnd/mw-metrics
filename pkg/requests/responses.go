package requests

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/metrics"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/models"
)

type Response struct {
	Status int `json:"status"`
	Body   any `json:"body"`
}

type FilmsResponse struct {
	Page           uint64            `json:"current_page"`
	PageSize       uint64            `json:"page_size"`
	CollectionName string            `json:"collection_name"`
	Total          uint64            `json:"total"`
	Films          []models.FilmItem `json:"films"`
}

type FilmResponse struct {
	Film       models.FilmItem    `json:"film"`
	Genres     []models.GenreItem `json:"genre"`
	Rating     float64            `json:"rating"`
	Number     uint64             `json:"number"`
	Directors  []models.CrewItem  `json:"directors"`
	Scenarists []models.CrewItem  `json:"scenarists"`
	Characters []models.Character `json:"actors"`
}

type ActorResponse struct {
	Name      string                  `json:"name"`
	Photo     string                  `json:"poster_href"`
	Career    []models.ProfessionItem `json:"career"`
	BirthDate string                  `json:"birthday"`
	Country   string                  `json:"country"`
	Info      string                  `json:"info_text"`
}

type ActorsResponse struct {
	Actors []models.Character `json:"actors"`
}

type CommentResponse struct {
	Comments []models.CommentItem `json:"comment"`
}

type ProfileResponse struct {
	Email     string `json:"email"`
	Name      string `json:"name"`
	Login     string `json:"login"`
	Photo     string `json:"photo"`
	BirthDate string `json:"birthday"`
}

type AuthCheckResponse struct {
	Login string `json:"login"`
	Role  string `json:"role"`
}

type CalendarResponse struct {
	MonthName  string           `json:"monthName"`
	MonthText  string           `json:"monthText"`
	CurrentDay uint8            `json:"currentDay"`
	Days       []models.DayItem `json:"days"`
}

func sendMetrics(mt *metrics.Metrics, path string, status int, start time.Time) {
	end := time.Since(start)
	mt.Time.WithLabelValues(strconv.Itoa(status), path).Observe(end.Seconds())
	mt.Hits.WithLabelValues(strconv.Itoa(status), path).Inc()
}

func SendResponse(w http.ResponseWriter, path string, response Response, lg *slog.Logger, mt *metrics.Metrics, start time.Time) {
	jsonResponse, err := json.Marshal(response)
	sendMetrics(mt, path, response.Status, start)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		sendMetrics(mt, path, http.StatusInternalServerError, start)
		lg.Error("failed to pack json", "err", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonResponse)
	if err != nil {
		lg.Error("failed to send response", "err", err.Error())
		return
	}
}
