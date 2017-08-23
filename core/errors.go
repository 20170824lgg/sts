package core

import "errors"

var (
	ErrPlayerNotFound           = errors.New("player not found")
	ErrTournamentNotFound       = errors.New("tournament not found")
	ErrTournPlayerNotFound      = errors.New("tournament player not found")
	ErrNegativePlayerBalance    = errors.New("operation would result in negative player balance")
	ErrDuplicateTournament      = errors.New("duplicate tournament")
	ErrDuplicateTournPlayer     = errors.New("duplicate tournament player")
	ErrTournamentFinished       = errors.New("tournament is finished")
	ErrInvalidTournamentDeposit = errors.New("invalid tournament deposit value, must greater than 0")
	ErrInvalidTournamentPrize   = errors.New("invalid tournament prize value, must be greater than 0")
	ErrTooManyBackers           = errors.New("too many player backers")
	ErrDuplicateBackers         = errors.New("duplicate backers")
)
