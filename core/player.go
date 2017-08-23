package core

// Player is an object for player account.
type Player struct {
	PlayerID string `json:"playerId"`
	Balance  int64  `json:"balance"`
}

func (p *Player) AddBalance(delta int64) error {
	if p.Balance+delta < 0 {
		return ErrNegativePlayerBalance
	}
	p.Balance += delta
	return nil
}
