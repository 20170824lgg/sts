package db

import (
	"github.com/20170819lgg/sts/core"
	"github.com/Masterminds/squirrel"
	"github.com/go-sql-driver/mysql"
)

func tournamentSelect(q squirrel.Queryer, d queryDecorator) ([]core.Tournament, error) {
	query := d(squirrel.
		Select("tournament_id", "entry_deposit", "active").
		From("tournament"))

	rows, err := squirrel.QueryWith(q, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ts []core.Tournament
	for rows.Next() {
		var t core.Tournament
		if err := rows.Scan(&t.ID, &t.EntryDeposit, &t.Active); err != nil {
			return nil, err
		}
		ts = append(ts, t)
	}
	return ts, nil
}

func TournamentGet(q squirrel.Queryer, tournamentID int) (*core.Tournament, error) {
	ts, err := tournamentSelect(q, func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where("tournament_id = ?", tournamentID)
	})
	switch {
	case err != nil:
		return nil, err
	case len(ts) == 0:
		return nil, ErrNotFound
	default:
		return &ts[0], nil
	}
}

func TournamentGetForUpdate(q squirrel.Queryer, tournamentID int) (*core.Tournament, error) {
	ts, err := tournamentSelect(q, func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where("tournament_id = ?", tournamentID).Suffix("FOR UPDATE")
	})
	switch {
	case err != nil:
		return nil, err
	case len(ts) == 0:
		return nil, ErrNotFound
	default:
		return &ts[0], nil
	}
}

func TournamentUpdate(e squirrel.Execer, t *core.Tournament) error {
	query := squirrel.
		Update("tournament").
		SetMap(map[string]interface{}{
			"entry_deposit": t.EntryDeposit,
			"active":        t.Active,
		}).
		Where("tournament_id = ?", t.ID)

	_, err := squirrel.ExecWith(e, query)
	return err
}

func TournamentInsert(e squirrel.Execer, t *core.Tournament) error {
	query := squirrel.
		Insert("tournament").
		SetMap(map[string]interface{}{
			"tournament_id": t.ID,
			"entry_deposit": t.EntryDeposit,
			"active":        t.Active,
		})
	_, err := squirrel.ExecWith(e, query)
	if me, ok := err.(*mysql.MySQLError); ok && me.Number == 1062 {
		return ErrAlreadyExists
	}
	return err
}
