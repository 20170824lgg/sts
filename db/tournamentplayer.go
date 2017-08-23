package db

import (
	"encoding/json"
	"errors"

	"github.com/20170819lgg/sts/core"
	"github.com/Masterminds/squirrel"
	"github.com/go-sql-driver/mysql"
)

func TournPlayerSelect(q squirrel.Queryer, d queryDecorator) ([]core.TournPlayer, error) {
	query := d(squirrel.
		Select("tournament_id", "player_id", "fee", "data").
		From("tournament_player"))

	rows, err := squirrel.QueryWith(q, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tps []core.TournPlayer
	for rows.Next() {
		var tp core.TournPlayer
		var blob []byte
		if err := rows.Scan(&tp.TournamentID, &tp.PlayerID, &tp.Fee, &blob); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(blob, &tp.Backers); err != nil {
			return nil, err
		}
		tps = append(tps, tp)
	}
	return tps, nil
}

func TournPlayerGet(q squirrel.Queryer, tournamentID int, playerID string) (*core.TournPlayer, error) {
	tps, err := TournPlayerSelect(q, func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where(squirrel.Eq{
			"tournament_id": tournamentID,
			"player_id":     playerID,
		})
	})

	switch {
	case err != nil:
		return nil, err
	case len(tps) == 0:
		return nil, ErrNotFound
	default:
		return &tps[0], nil
	}
}

func TournPlayerInsert(e squirrel.Execer, tp *core.TournPlayer) error {
	blob, err := json.Marshal(&tp.Backers)
	if err != nil {
		return err
	}

	if len(blob) > TextMaxLength {
		return errors.New("db: backers slice is too big")
	}

	query := squirrel.
		Insert("tournament_player").
		SetMap(map[string]interface{}{
			"tournament_id": tp.TournamentID,
			"player_id":     tp.PlayerID,
			"fee":           tp.Fee,
			"data":          blob,
		})
	_, err = squirrel.ExecWith(e, query)
	if me, ok := err.(*mysql.MySQLError); ok && me.Number == 1062 {
		return ErrAlreadyExists
	}
	return err
}
