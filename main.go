package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/20170819lgg/sts/db"
	"github.com/Sirupsen/logrus"
	"github.com/fln/pcors"
	_ "github.com/go-sql-driver/mysql"
	"github.com/go-zoo/bone"
	"github.com/vrischmann/envconfig"
)

// conf strores application configuration, it is controlled via environment
// variables on application startup.
var conf struct {
	Listen string `envconfig:"default=:8080"`
	DSN    string
}

func respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logrus.WithError(err).Error("marshaling JSON response")
	}
}

func respondStatus(w http.ResponseWriter, r apiResponse) {
	switch r.status {
	case http.StatusNoContent:
		w.WriteHeader(r.status)
	default:
		http.Error(w, r.msg, r.status)
	}
}

func mainRouter(app *application) http.Handler {
	mux := bone.New()

	mux.GetFunc("/take", func(w http.ResponseWriter, r *http.Request) {
		playerID := r.URL.Query().Get("playerId")
		if playerID == "" {
			http.Error(w, "missing playerId parameter", http.StatusBadRequest)
			return
		}
		points, err := strconv.ParseInt(r.URL.Query().Get("points"), 10, 64)
		if err != nil || points < 0 {
			http.Error(w, "invalid points parameter", http.StatusBadRequest)
			return
		}

		resp, err := app.addPoints(playerID, -points)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"playerID": playerID,
				"points":   points,
			}).WithError(err).Error("reducing player account balance")
			http.Error(w, "unexpected error", http.StatusInternalServerError)
			return
		}
		respondStatus(w, *resp)
	})

	mux.GetFunc("/fund", func(w http.ResponseWriter, r *http.Request) {
		playerID := r.URL.Query().Get("playerId")
		if playerID == "" {
			http.Error(w, "missing playerId parameter", http.StatusBadRequest)
			return
		}
		points, err := strconv.ParseInt(r.URL.Query().Get("points"), 10, 64)
		if err != nil || points < 0 {
			http.Error(w, "invalid points parameter", http.StatusBadRequest)
			return
		}

		resp, err := app.addPoints(playerID, points)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"playerID": playerID,
				"points":   points,
			}).WithError(err).Error("increasing player account balance")
			http.Error(w, "unexpected error", http.StatusInternalServerError)
			return
		}
		respondStatus(w, *resp)

	})

	mux.GetFunc("/balance", func(w http.ResponseWriter, r *http.Request) {
		playerID := r.URL.Query().Get("playerId")
		if playerID == "" {
			http.Error(w, "missing playerId parameter", http.StatusBadRequest)
			return
		}
		player, err := app.balance(playerID)
		if err != nil {
			logrus.WithField("playerID", playerID).WithError(err).Error("getting player balance")
			http.Error(w, "unexpected error", http.StatusInternalServerError)
			return
		}
		if player == nil {
			http.Error(w, "player account not found", http.StatusNotFound)
			return
		}
		respondJSON(w, player)
	})

	mux.GetFunc("/announceTournament", func(w http.ResponseWriter, r *http.Request) {
		tournamentID, err := strconv.Atoi(r.URL.Query().Get("tournamentId"))
		if err != nil {
			http.Error(w, "invalid tournamentId parameter", http.StatusBadRequest)
			return
		}
		deposit, err := strconv.ParseInt(r.URL.Query().Get("deposit"), 10, 64)
		if err != nil || deposit <= 0 {
			http.Error(w, "invalid deposit parameter", http.StatusBadRequest)
			return
		}

		resp, err := app.announceTournament(tournamentID, deposit)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"tournamentID": tournamentID,
				"deposit":      deposit,
			}).WithError(err).Error("creating tournament")
			http.Error(w, "unexpected error", http.StatusInternalServerError)
			return
		}
		respondStatus(w, *resp)
	})

	mux.GetFunc("/joinTournament", func(w http.ResponseWriter, r *http.Request) {
		tournamentID, err := strconv.Atoi(r.URL.Query().Get("tournamentId"))
		if err != nil {
			http.Error(w, "invalid tournamentId parameter", http.StatusBadRequest)
			return
		}
		playerID := r.URL.Query().Get("playerId")
		if playerID == "" {
			http.Error(w, "missing playerId parameter", http.StatusBadRequest)
			return
		}
		backerIDs := r.URL.Query()["backerId"]

		resp, err := app.joinTournament(tournamentID, playerID, backerIDs)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"tournamentID": tournamentID,
				"playerID":     playerID,
				"backedIDs":    backerIDs,
			}).WithError(err).Error("joining player to tournament")
			http.Error(w, "unexpected error", http.StatusInternalServerError)
			return
		}
		respondStatus(w, *resp)
	})

	mux.PostFunc("/resultTournament", func(w http.ResponseWriter, r *http.Request) {
		var data struct {
			ID      int `json:"tournamentId"`
			Winners []struct {
				PlayerID string `json:"playerId"`
				Prize    int64  `json:"prize"`
			} `json:"winners"`
		}
		if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		winners := make(map[string]int64)
		for _, wn := range data.Winners {
			winners[wn.PlayerID] = wn.Prize
		}

		resp, err := app.resultTroutnament(data.ID, winners)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"tournamentID": data.ID,
				"winners":      winners,
			}).WithError(err).Error("resulting tournament")
			http.Error(w, "unexpected error", http.StatusInternalServerError)
			return
		}
		respondStatus(w, *resp)
	})

	mux.GetFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		if err := app.reset(conf.DSN); err != nil {
			logrus.WithError(err).Error("resetting database")
			http.Error(w, "unexpected error", http.StatusInternalServerError)
			return
		}
		respondStatus(w, *respOK())
	})
	return mux
}

func main() {
	if err := envconfig.InitWithPrefix(&conf, "STS"); err != nil {
		logrus.WithError(err).Fatal("parsing environment variables")
	}

	// Establish main database connection
	dbh, err := db.Connect(conf.DSN)
	if err != nil {
		logrus.WithField("error", err).Fatal("connecting to DB")
	}
	defer dbh.Close()

	app := newApplication(dbh)

	server := &http.Server{
		Addr:    conf.Listen,
		Handler: pcors.Default(mainRouter(app)),
	}
	stopped := make(chan struct{})
	go func() {
		logrus.Info("starting web server")
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			logrus.WithError(err).Error("http server stopped with error")
		}
		close(stopped)
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	<-c
	if err := server.Shutdown(context.TODO()); err != nil {
		logrus.WithError(err).Error("calling shutdown on http server")
	}
	<-stopped
	logrus.Info("graceful shutdown complete")
}
