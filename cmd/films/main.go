package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/go-park-mail-ru/2023_2_Vkladyshi/configs"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/films/delivery"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/films/repository/calendar"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/films/repository/crew"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/films/repository/film"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/films/repository/genre"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/films/repository/profession"
	"github.com/go-park-mail-ru/2023_2_Vkladyshi/films/usecase"
)

func main() {
	var path string
	flag.StringVar(&path, "films_log_path", "films_log.log", "Путь к логу фильмов")
	logFile, _ := os.Create(path)
	lg := slog.New(slog.NewJSONHandler(logFile, nil))

	config, err := configs.ReadFilmConfig()
	if err != nil {
		lg.Error("read config error", "err", err.Error())
		return
	}

	var (
		films       film.IFilmsRepo
		genres      genre.IGenreRepo
		actors      crew.ICrewRepo
		professions profession.IProfessionRepo
		news        calendar.ICalendarRepo
	)
	switch config.Films_db {
	case "postgres":
		films, err = film.GetFilmRepo(config, lg)
	}
	if err != nil {
		lg.Error("cant create repo")
		return
	}

	switch config.Genres_db {
	case "postgres":
		genres, err = genre.GetGenreRepo(config, lg)
	}
	if err != nil {
		lg.Error("cant create repo")
		return
	}

	switch config.Crew_db {
	case "postgres":
		actors, err = crew.GetCrewRepo(config, lg)
	}
	if err != nil {
		lg.Error("cant create repo")
		return
	}

	switch config.Profession_db {
	case "postgres":
		professions, err = profession.GetProfessionRepo(config, lg)
	}
	if err != nil {
		lg.Error("cant create repo")
		return
	}

	switch config.Calendar_db {
	case "postgres":
		news, err = calendar.GetCalendarRepo(config, lg)
	}
	if err != nil {
		lg.Error("cant creare calendar repo")
		return
	}

	core := usecase.GetCore(config, lg, films, genres, actors, professions, news)
	api := delivery.GetApi(core, lg, config)

	api.ListenAndServe()
}
