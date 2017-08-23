package main

import (
	"database/sql"
	"net/http"
	"sort"

	"github.com/20170819lgg/sts/core"
	"github.com/20170819lgg/sts/db"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
)

type apiResponse struct {
	status int
	msg    string
}

type application struct {
	db *sql.DB
}

func respOK() *apiResponse                 { return &apiResponse{status: http.StatusNoContent} }
func respConflict(msg string) *apiResponse { return &apiResponse{status: http.StatusConflict, msg: msg} }

func newApplication(db *sql.DB) *application {
	return &application{
		db: db,
	}
}

func (a *application) addPoints(playerID string, points int64) (*apiResponse, error) {
	tx, err := a.db.Begin()
	if err != nil {
		return nil, errors.WithMessage(err, "starting transaction")
	}
	defer tx.Rollback()

	player, err := db.PlayerGetForUpdate(tx, playerID)
	switch err {
	case nil:
		// OK
	case db.ErrNotFound:
		player = &core.Player{
			PlayerID: playerID,
			Balance:  0,
		}
		if err := db.PlayerInsert(tx, player); err != nil {
			return nil, errors.WithMessage(err, "inserting player")
		}
	default:
		return nil, errors.WithMessage(err, "getting player for update")
	}

	if err := player.AddBalance(points); err != nil {
		return respConflict(err.Error()), nil
	}

	if err := db.PlayerUpdate(tx, player); err != nil {
		return nil, errors.WithMessage(err, "updating player")
	}
	if err := tx.Commit(); err != nil {
		return nil, errors.WithMessage(err, "committing transaction")
	}
	return respOK(), nil
}

func (a *application) balance(playerID string) (*core.Player, error) {
	player, err := db.PlayerGet(a.db, playerID)
	switch err {
	case nil:
		return player, nil
	case db.ErrNotFound:
		return nil, nil
	default:
		return nil, errors.WithMessage(err, "getting player")
	}
}

func (a *application) announceTournament(tournamentID int, deposit int64) (*apiResponse, error) {
	tournament, err := core.NewTournament(tournamentID, deposit)
	if err != nil {
		return respConflict(err.Error()), nil
	}
	switch db.TournamentInsert(a.db, tournament) {
	case nil:
		return respOK(), nil
	case db.ErrAlreadyExists:
		return respConflict(core.ErrDuplicateTournament.Error()), nil
	default:
		return nil, errors.WithMessage(err, "inserting tournament")
	}
}

func (a *application) joinTournament(tournamentID int, playerID string, backerIDs []string) (*apiResponse, error) {
	tx, err := a.db.Begin()
	if err != nil {
		return nil, errors.WithMessage(err, "starting transaction")
	}
	defer tx.Rollback()

	tournament, err := db.TournamentGet(tx, tournamentID)
	switch err {
	case nil:
		// OK
	case db.ErrNotFound:
		return respConflict(core.ErrTournamentNotFound.Error()), nil
	default:
		return nil, errors.WithMessage(err, "getting tournament")
	}
	playerIDs := append([]string{playerID}, backerIDs...)
	players, err := db.PlayerSelectForUpdate(tx, playerIDs)
	if err != nil {
		return nil, errors.WithMessage(err, "getting players for update")
	}

	tp, err := tournament.NewTournPlayer(playerID, backerIDs)
	if err != nil {
		return respConflict(err.Error()), nil
	}

	if err := tp.DeductDeposit(players); err != nil {
		return respConflict(err.Error()), nil
	}

	err = db.TournPlayerInsert(tx, tp)
	switch err {
	case nil:
		// OK
	case db.ErrAlreadyExists:
		return respConflict(core.ErrDuplicateTournPlayer.Error()), nil
	default:
		return nil, errors.WithMessage(err, "inserting tournament player")
	}

	for _, acc := range players {
		if err := db.PlayerUpdate(tx, acc); err != nil {
			return nil, errors.WithMessage(err, "updating player balance")
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.WithMessage(err, "committing transaction")
	}
	return respOK(), nil
}

func (a *application) resultTroutnament(tournamentID int, winners map[string]int64) (*apiResponse, error) {
	tx, err := a.db.Begin()
	if err != nil {
		return nil, errors.WithMessage(err, "starting transaction")
	}
	defer tx.Rollback()

	tournament, err := db.TournamentGetForUpdate(tx, tournamentID)
	switch err {
	case nil:
		// OK
	case db.ErrNotFound:
		return respConflict(core.ErrTournamentNotFound.Error()), nil
	default:
		return nil, errors.WithMessage(err, "getting tournament for update")
	}

	if err := tournament.MarkFinished(); err != nil {
		return respConflict(err.Error()), nil
	}
	if err := db.TournamentUpdate(tx, tournament); err != nil {
		return nil, errors.WithMessage(err, "updating tournament")
	}

	tws := make([]*core.TournWinner, 0, len(winners))
	playerIDs := sort.StringSlice{}
	for playerID, prize := range winners {
		tp, err := db.TournPlayerGet(tx, tournamentID, playerID)
		switch err {
		case nil:
			// OK
		case db.ErrNotFound:
			return respConflict(core.ErrTournPlayerNotFound.Error()), nil
		default:
			return nil, errors.WithMessage(err, "getting tournament player")
		}

		tw, err := tp.NewTournWinner(prize)
		if err != nil {
			return respConflict(err.Error()), nil
		}

		for _, b := range tw.Backers {
			playerIDs = append(playerIDs, b.PlayerID)
		}
		tws = append(tws, tw)
	}

	// retrieve all player accounts in single query to prevent deadlocks
	// between multiple tournament resulting requests
	players, err := db.PlayerSelectForUpdate(tx, playerIDs)
	if err != nil {
		return nil, errors.WithMessage(err, "getting player accounts")
	}

	for _, tw := range tws {
		if err := tw.PayoutPrize(players); err != nil {
			return respConflict(err.Error()), nil
		}
		if err := db.TournamentWinnerInsert(tx, tw); err != nil {
			return nil, errors.WithMessage(err, "inserting tournament winner")
		}
	}
	for _, acc := range players {
		if err := db.PlayerUpdate(tx, acc); err != nil {
			return nil, errors.WithMessage(err, "updating player account")
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.WithMessage(err, "committing transaction")
	}
	return respOK(), nil
}

func (a *application) reset(dsn string) error {
	if err := db.RecreateDB(dsn); err != nil {
		return err
	}
	if err := db.CreateSchema(a.db); err != nil {
		return err
	}
	return nil
}
