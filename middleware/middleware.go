package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/films/usecase"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/metrics"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/pkg/requests"
)

type contextKey string

const UserIDKey contextKey = "userId"

func AuthCheck(next http.Handler, core *usecase.Core, lg *slog.Logger, mt *metrics.Metrics) http.Handler {
	start := time.Now()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := r.Cookie("session_id")
		if errors.Is(err, http.ErrNoCookie) {
			next.ServeHTTP(w, r)
			return
		}

		userId, err := core.GetUserId(r.Context(), session.Value)
		if err != nil {
			lg.Error("auth check error", "err", err.Error())
			next.ServeHTTP(w, r)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), UserIDKey, userId))

		next.ServeHTTP(w, r)
		status := r.Context().Value(requests.StatusKey).(int)
		end := time.Since(start)
		mt.Time.WithLabelValues(strconv.Itoa(status), r.URL.Path).Observe(end.Seconds())
		mt.Hits.WithLabelValues(strconv.Itoa(status), r.URL.Path).Inc()
	})
}

func CollectMetrics(next http.Handler, lg *slog.Logger, mt *metrics.Metrics) http.Handler {
	start := time.Now()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		status := r.Context().Value(requests.StatusKey).(int)
		end := time.Since(start)
		mt.Time.WithLabelValues(strconv.Itoa(status), r.URL.Path).Observe(end.Seconds())
		mt.Hits.WithLabelValues(strconv.Itoa(status), r.URL.Path).Inc()
	})
}
