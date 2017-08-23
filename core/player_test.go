package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlayerAddBalance(t *testing.T) {
	tests := []struct {
		msg   string
		delta int64
		in    Player
		out   Player
		err   error
	}{
		{
			msg:   "identity operation",
			delta: 0,
			in:    Player{},
			out:   Player{},
		},
		{
			msg:   "fund op",
			delta: 10,
			in:    Player{Balance: 100},
			out:   Player{Balance: 110},
		},
		{
			msg:   "take op",
			delta: -10,
			in:    Player{Balance: 100},
			out:   Player{Balance: 90},
		},
		{
			msg:   "negative balance",
			delta: -110,
			in:    Player{Balance: 100},
			out:   Player{Balance: 100},
			err:   ErrNegativePlayerBalance,
		},
	}

	for _, test := range tests {
		err := test.in.AddBalance(test.delta)
		assert.Equal(t, test.err, err, test.msg)
		assert.Equal(t, test.out, test.in, test.msg)
	}
}
