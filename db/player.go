package db

import (
	"github.com/20170819lgg/sts/core"
	"github.com/Masterminds/squirrel"
	"github.com/go-sql-driver/mysql"
)

// playerSelect is generic funcion for querying player table.
func playerSelect(q squirrel.Queryer, d queryDecorator) ([]core.Player, error) {
	query := d(squirrel.
		Select("player_id", "balance").
		From("player"))

	rows, err := squirrel.QueryWith(q, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ps []core.Player
	for rows.Next() {
		var p core.Player
		if err := rows.Scan(&p.PlayerID, &p.Balance); err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	return ps, nil
}

func PlayerSelectForUpdate(q squirrel.Queryer, playerIDs []string) (map[string]*core.Player, error) {
	ps, err := playerSelect(q, func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.
			Where(squirrel.Eq{
				"player_id": playerIDs,
			}).
			Suffix("FOR UPDATE")
	})
	m := make(map[string]*core.Player)
	for i, p := range ps {
		m[p.PlayerID] = &ps[i]
	}
	return m, err
}

func PlayerGet(q squirrel.Queryer, playerID string) (*core.Player, error) {
	ps, err := playerSelect(q, func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where("player_id = ?", playerID)
	})
	switch {
	case err != nil:
		return nil, err
	case len(ps) == 0:
		return nil, ErrNotFound
	default:
		return &ps[0], nil
	}
}

func PlayerGetForUpdate(q squirrel.Queryer, playerID string) (*core.Player, error) {
	ps, err := playerSelect(q, func(b squirrel.SelectBuilder) squirrel.SelectBuilder {
		return b.Where("player_id = ?", playerID).Suffix("FOR UPDATE")
	})
	switch {
	case err != nil:
		return nil, err
	case len(ps) == 0:
		return nil, ErrNotFound
	default:
		return &ps[0], nil
	}
}

func PlayerInsert(e squirrel.Execer, player *core.Player) error {
	query := squirrel.
		Insert("player").
		SetMap(map[string]interface{}{
			"player_id": player.PlayerID,
			"balance":   player.Balance,
		})
	_, err := squirrel.ExecWith(e, query)
	if me, ok := err.(*mysql.MySQLError); ok && me.Number == 1062 {
		return ErrAlreadyExists
	}
	return err
}

func PlayerUpdate(e squirrel.Execer, player *core.Player) error {
	query := squirrel.
		Update("player").
		SetMap(map[string]interface{}{
			"balance": player.Balance,
		}).
		Where("player_id = ?", player.PlayerID)
	_, err := squirrel.ExecWith(e, query)
	return err
}
