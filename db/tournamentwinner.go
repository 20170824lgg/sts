package db

import (
	"encoding/json"
	"errors"

	"github.com/20170819lgg/sts/core"
	"github.com/Masterminds/squirrel"
	"github.com/go-sql-driver/mysql"
)

func TournamentWinnerInsert(e squirrel.Execer, tp *core.TournWinner) error {
	blob, err := json.Marshal(&tp.Backers)
	if err != nil {
		return err
	}

	if len(blob) > TextMaxLength {
		return errors.New("db: backers slice is too big")
	}

	query := squirrel.
		Insert("tournament_winner").
		SetMap(map[string]interface{}{
			"tournament_id": tp.TournamentID,
			"player_id":     tp.PlayerID,
			"prize":         tp.Prize,
			"data":          blob,
		})
	_, err = squirrel.ExecWith(e, query)
	if me, ok := err.(*mysql.MySQLError); ok && me.Number == 1062 {
		return ErrAlreadyExists
	}
	return err
}
