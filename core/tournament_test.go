package core

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHasDuplicates(t *testing.T) {
	tests := []struct {
		strings    []string
		duplicates bool
	}{
		{strings: nil, duplicates: false},
		{strings: []string{}, duplicates: false},
		{strings: []string{"a", "b"}, duplicates: false},
		{strings: []string{"a", "b", "c"}, duplicates: false},
		{strings: []string{"a", "b", "a"}, duplicates: true},
	}

	for _, test := range tests {
		duplicates := hasDuplicates(test.strings)
		msg := fmt.Sprintf("%v", test.strings)
		assert.Equal(t, test.duplicates, duplicates, msg)
	}
}

func TestSplitPoints(t *testing.T) {
	tests := []struct {
		points int64
		n      int
		result []int64
	}{
		{
			points: 4,
			n:      5,
			result: []int64{1, 1, 1, 1, 0},
		},
		{
			points: 9,
			n:      5,
			result: []int64{2, 2, 2, 2, 1},
		},
		{
			points: 100,
			n:      3,
			result: []int64{34, 33, 33},
		},
		{
			points: 0,
			n:      2,
			result: []int64{0, 0},
		},
		{
			points: -3,
			n:      2,
			result: []int64{-2, -1},
		},
		{
			points: 5,
			n:      0,
			result: []int64{},
		},
	}

	for _, test := range tests {
		result := splitPoints(test.points, test.n)
		msg := fmt.Sprintf("splitPoints(%d, %d)", test.points, test.n)
		assert.Equal(t, test.result, result, msg)

		sum := int64(0)
		for _, n := range result {
			sum += n
		}
		if test.n != 0 {
			assert.Equal(t, test.points, sum, msg)
		}
	}
}

func TestTournamentMarkFinished(t *testing.T) {
	tests := []struct {
		in  *Tournament
		out *Tournament
		err error
	}{
		{
			in:  &Tournament{Active: true},
			out: &Tournament{Active: false},
		},
		{
			in:  &Tournament{Active: false},
			out: &Tournament{Active: false},
			err: ErrTournamentFinished,
		},
	}

	for _, test := range tests {
		err := test.in.MarkFinished()
		assert.Equal(t, test.err, err)
		assert.Equal(t, test.out, test.in)
	}
}

func TestNewTournPlayer(t *testing.T) {
	tests := []struct {
		msg       string
		t         Tournament
		playerID  string
		backerIDs []string
		tp        *TournPlayer
		err       error
	}{
		{
			msg: "too low deposit",
			t: Tournament{
				ID:           123,
				EntryDeposit: 3,
				Active:       true,
			},
			playerID:  "P1",
			backerIDs: []string{"P2", "P3", "P4"},
			err:       ErrTooManyBackers,
		},
		{
			msg: "inactive tournament",
			t: Tournament{
				ID:           123,
				EntryDeposit: 1,
				Active:       false,
			},
			playerID: "P1",
			err:      ErrTournamentFinished,
		},
		{
			msg: "single player",
			t: Tournament{
				ID:           123,
				EntryDeposit: 1,
				Active:       true,
			},
			playerID: "P1",
			tp: &TournPlayer{
				TournamentID: 123,
				PlayerID:     "P1",
				Fee:          1,
				Backers: []Backer{
					{PlayerID: "P1", Points: 1},
				},
			},
		},
		{
			msg: "player and two backers",
			t: Tournament{
				ID:           123,
				EntryDeposit: 100,
				Active:       true,
			},
			playerID:  "P1",
			backerIDs: []string{"P2", "P3"},
			tp: &TournPlayer{
				TournamentID: 123,
				PlayerID:     "P1",
				Fee:          100,
				Backers: []Backer{
					{PlayerID: "P1", Points: 34},
					{PlayerID: "P2", Points: 33},
					{PlayerID: "P3", Points: 33},
				},
			},
		},
		{
			msg: "duplicate backers",
			t: Tournament{
				ID:           123,
				EntryDeposit: 100,
				Active:       true,
			},
			playerID:  "P1",
			backerIDs: []string{"P2", "P2"},
			err:       ErrDuplicateBackers,
		},
		{
			msg: "duplicate player backer",
			t: Tournament{
				ID:           123,
				EntryDeposit: 100,
				Active:       true,
			},
			playerID:  "P1",
			backerIDs: []string{"P1", "P2"},
			err:       ErrDuplicateBackers,
		},
	}

	for _, test := range tests {
		tp, err := test.t.NewTournPlayer(test.playerID, test.backerIDs)
		assert.Equal(t, test.err, err, test.msg)
		assert.Equal(t, test.tp, tp, test.msg)
	}
}

func TestDeductDeposit(t *testing.T) {
	tests := []struct {
		msg string
		tp  TournPlayer
		in  map[string]*Player
		out map[string]*Player
		err error
	}{
		{
			msg: "empty backers",
			tp:  TournPlayer{},
			in:  map[string]*Player{},
			out: map[string]*Player{},
		},
		{
			msg: "missing player",
			tp: TournPlayer{
				Backers: []Backer{
					{PlayerID: "P1", Points: 1},
				},
			},
			in:  map[string]*Player{},
			out: map[string]*Player{},
			err: ErrPlayerNotFound,
		},
		{
			msg: "negative balance",
			tp: TournPlayer{
				Backers: []Backer{
					{PlayerID: "P1", Points: 200},
				},
			},
			in: map[string]*Player{
				"P1": &Player{Balance: 100},
			},
			out: map[string]*Player{
				"P1": &Player{Balance: 100},
			},
			err: ErrNegativePlayerBalance,
		},
		{
			msg: "valid payins",
			tp: TournPlayer{
				Backers: []Backer{
					{PlayerID: "P1", Points: 1},
					{PlayerID: "P2", Points: 5},
					{PlayerID: "P3", Points: -150},
					{PlayerID: "P4", Points: 0},
				},
			},
			in: map[string]*Player{
				"P1": &Player{Balance: 100},
				"P2": &Player{Balance: 100},
				"P3": &Player{Balance: 100},
				"P4": &Player{Balance: 100},
			},
			out: map[string]*Player{
				"P1": &Player{Balance: 99},
				"P2": &Player{Balance: 95},
				"P3": &Player{Balance: 250},
				"P4": &Player{Balance: 100},
			},
		},
	}

	for _, test := range tests {
		err := test.tp.DeductDeposit(test.in)
		assert.Equal(t, test.err, err, test.msg)
		assert.Equal(t, test.out, test.in, test.msg)
	}
}

func TestNewTournWinner(t *testing.T) {
	tests := []struct {
		msg   string
		tp    TournPlayer
		prize int64
		tw    *TournWinner
		err   error
	}{
		{
			msg:   "negative prize",
			tp:    TournPlayer{},
			prize: -100,
			err:   ErrInvalidTournamentPrize,
		},
		{
			msg:   "zero tourn player",
			tp:    TournPlayer{},
			prize: 100,
			tw: &TournWinner{
				Prize:   100,
				Backers: []Backer{},
			},
		},
		{
			msg: "valid case",
			tp: TournPlayer{
				TournamentID: 123,
				PlayerID:     "P1",
				Fee:          100,
				Backers: []Backer{
					{PlayerID: "P1", Points: 34},
					{PlayerID: "P3", Points: 33},
					{PlayerID: "P3", Points: 33},
				},
			},
			prize: 500,
			tw: &TournWinner{
				TournamentID: 123,
				PlayerID:     "P1",
				Prize:        500,
				Backers: []Backer{
					{PlayerID: "P1", Points: 167},
					{PlayerID: "P3", Points: 167},
					{PlayerID: "P3", Points: 166},
				},
			},
		},
	}

	for _, test := range tests {
		tw, err := test.tp.NewTournWinner(test.prize)
		assert.Equal(t, test.err, err, test.msg)
		assert.Equal(t, test.tw, tw, test.msg)
	}
}

func TestPayoutPrize(t *testing.T) {
	tests := []struct {
		msg string
		tw  TournWinner
		in  map[string]*Player
		out map[string]*Player
		err error
	}{
		{
			msg: "empty winner",
			tw:  TournWinner{},
			in:  map[string]*Player{},
			out: map[string]*Player{},
		},
		{
			msg: "missing player",
			tw: TournWinner{
				Backers: []Backer{
					{PlayerID: "P1", Points: 1},
				},
			},
			in:  map[string]*Player{},
			out: map[string]*Player{},
			err: ErrPlayerNotFound,
		},
		{
			msg: "valid payouts",
			tw: TournWinner{
				Backers: []Backer{
					{PlayerID: "P1", Points: 1},
					{PlayerID: "P2", Points: 5},
					{PlayerID: "P3", Points: -150},
					{PlayerID: "P4", Points: 0},
				},
			},
			in: map[string]*Player{
				"P1": &Player{Balance: 100},
				"P2": &Player{Balance: 100},
				"P3": &Player{Balance: 100},
				"P4": &Player{Balance: 100},
			},
			out: map[string]*Player{
				"P1": &Player{Balance: 101},
				"P2": &Player{Balance: 105},
				"P3": &Player{Balance: -50},
				"P4": &Player{Balance: 100},
			},
		},
	}

	for _, test := range tests {
		err := test.tw.PayoutPrize(test.in)
		assert.Equal(t, test.err, err, test.msg)
		assert.Equal(t, test.out, test.in, test.msg)
	}
}
