package core

type Tournament struct {
	ID           int
	EntryDeposit int64
	Active       bool
}

type Backer struct {
	PlayerID string
	Points   int64
}

type TournPlayer struct {
	TournamentID int
	PlayerID     string
	Fee          int64
	Backers      []Backer
}

type TournWinner struct {
	TournamentID int
	PlayerID     string
	Prize        int64
	Backers      []Backer
}

func hasDuplicates(ss []string) bool {
	cache := make(map[string]struct{})
	for _, s := range ss {
		if _, found := cache[s]; found {
			return true
		}
		cache[s] = struct{}{}
	}
	return false
}

func splitPoints(points int64, n int) []int64 {
	if n <= 0 {
		return []int64{}
	}

	parts := make([]int64, n)
	q := points / int64(n)
	r := points % int64(n)

	for i := range parts {
		switch {
		case r > 0:
			parts[i] = q + 1
			r--
		case r == 0:
			parts[i] = q
		case r < 0:
			parts[i] = q - 1
			r++
		}
	}
	return parts
}

// NewTournament creates a new tournament object.
func NewTournament(tournamentID int, deposit int64) (*Tournament, error) {
	if deposit <= 0 {
		return nil, ErrInvalidTournamentDeposit
	}
	return &Tournament{
		ID:           tournamentID,
		EntryDeposit: deposit,
		Active:       true,
	}, nil
}

// NewTournPlayer joins given player and its backers to the tournament by
// creating new tournament player object.
func (t *Tournament) NewTournPlayer(playerID string, backerIDs []string) (*TournPlayer, error) {
	ids := append([]string{playerID}, backerIDs...)
	if t.EntryDeposit < int64(len(ids)) {
		return nil, ErrTooManyBackers
	}
	if !t.Active {
		return nil, ErrTournamentFinished
	}
	if hasDuplicates(ids) {
		return nil, ErrDuplicateBackers
	}

	parts := splitPoints(t.EntryDeposit, len(ids))
	b := make([]Backer, len(parts))
	for i, pts := range parts {
		b[i] = Backer{
			PlayerID: ids[i],
			Points:   pts,
		}
	}
	return &TournPlayer{
		TournamentID: t.ID,
		PlayerID:     playerID,
		Fee:          t.EntryDeposit,
		Backers:      b,
	}, nil
}

// MarkFinished updates tournament to be marked as finished.
func (t *Tournament) MarkFinished() error {
	if !t.Active {
		return ErrTournamentFinished
	}
	t.Active = false
	return nil
}

// DeductDeposit updates player and its backers balances to pay for
// participating in a tournament. This function will mutate given players map.
func (tp *TournPlayer) DeductDeposit(players map[string]*Player) error {
	for _, b := range tp.Backers {
		p, ok := players[b.PlayerID]
		if !ok {
			return ErrPlayerNotFound
		}
		if p.Balance < b.Points {
			return ErrNegativePlayerBalance
		}
	}
	for _, b := range tp.Backers {
		players[b.PlayerID].Balance -= b.Points
	}
	return nil
}

// NewTournWinner creates a new tournament winner object for a given tournament
// player and tournament winner prize. Prize is distributed in equal parts for
// all participation backers with the same algorithm as participation fee.
func (tp *TournPlayer) NewTournWinner(prize int64) (*TournWinner, error) {
	if prize < 0 {
		return nil, ErrInvalidTournamentPrize
	}
	parts := splitPoints(prize, len(tp.Backers))
	b := make([]Backer, len(parts))
	for i, pts := range parts {
		b[i] = Backer{
			PlayerID: tp.Backers[i].PlayerID,
			Points:   pts,
		}
	}
	return &TournWinner{
		TournamentID: tp.TournamentID,
		PlayerID:     tp.PlayerID,
		Prize:        prize,
		Backers:      b,
	}, nil
}

// PayoutPrize updates tournament winner and its backers balances to receive
// winners prize. This function will mutate given player map.
func (tw *TournWinner) PayoutPrize(players map[string]*Player) error {
	for _, b := range tw.Backers {
		if _, ok := players[b.PlayerID]; !ok {
			return ErrPlayerNotFound
		}
	}
	for _, b := range tw.Backers {
		players[b.PlayerID].Balance += b.Points
	}
	return nil
}
