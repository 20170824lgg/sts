package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/20170819lgg/sts/db"
	"github.com/stretchr/testify/assert"

	dockertest "gopkg.in/ory-am/dockertest.v3"
)

const (
	dbName = "testing"
	dbUser = "root"
	dbPass = ":secret"
)

var dbDSN string

func TestMain(m *testing.M) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	resource, err := pool.Run("mariadb", "latest", []string{
		fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s", dbPass),
		fmt.Sprintf("MYSQL_DATABASE=%s", dbName),
	})
	if err != nil {
		log.Fatalf("Could not start MariaDB: %s", err)
	}

	dbDSN = fmt.Sprintf("%s:%s@(localhost:%s)/%s", dbUser, dbPass, resource.GetPort("3306/tcp"), dbName)
	conf.DSN = dbDSN
	if err = pool.Retry(func() error {
		db, err := sql.Open("mysql", dbDSN)
		if err != nil {
			return err
		}
		defer db.Close()
		return db.Ping()
	}); err != nil {
		log.Fatalf("Waiting for MariaDB to boot-up: %s", err)
	}

	code := m.Run()
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Cleaning up MariaDB: %s", err)
	}
	os.Exit(code)
}

func newServer(t *testing.T) (*sql.DB, string, func()) {
	if err := db.RecreateDB(dbDSN); err != nil {
		t.Fatal(err)
	}
	dbh, err := db.Connect(dbDSN)
	if err != nil {
		t.Fatal("connecting to DB")
	}
	server := httptest.NewServer(mainRouter(newApplication(dbh)))

	return dbh, server.URL, func() {
		server.Close()
		dbh.Close()
	}
}

func get(url string) (string, int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}
	return string(body), resp.StatusCode, nil
}

func post(url string, data string) (string, int, error) {
	resp, err := http.Post(url, "application/json", bytes.NewBufferString(data))
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}
	return string(body), resp.StatusCode, nil
}

func TestFundTakeBalanceReset(t *testing.T) {
	_, url, cleanup := newServer(t)
	defer cleanup()

	t.Run("fund P1", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/fund?playerId=P1&points=100", url))
		assert.NoError(t, err)
		assert.Empty(t, body)
		assert.Equal(t, http.StatusNoContent, status, body)
	})

	t.Run("take P1", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/take?playerId=P1&points=20", url))
		assert.NoError(t, err)
		assert.Empty(t, body)
		assert.Equal(t, http.StatusNoContent, status, body)
	})

	t.Run("balance P1", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/balance?playerId=P1", url))
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		expected := `{"playerId": "P1", "balance": 80}`
		assert.JSONEq(t, expected, body)
	})

	t.Run("reset", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/reset", url))
		assert.NoError(t, err)
		assert.Empty(t, body)
		assert.Equal(t, http.StatusNoContent, status, body)
	})

	t.Run("balance P1 after reset", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/balance?playerId=P1", url))
		assert.NoError(t, err)
		assert.Equal(t, http.StatusNotFound, status, body)
	})
}

func TestFullUseCase(t *testing.T) {
	_, url, cleanup := newServer(t)
	defer cleanup()

	t.Run("fund P1 300", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/fund?playerId=P1&points=300", url))
		assert.NoError(t, err)
		assert.Empty(t, body)
		assert.Equal(t, http.StatusNoContent, status, body)
	})
	t.Run("fund P2 300", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/fund?playerId=P2&points=300", url))
		assert.NoError(t, err)
		assert.Empty(t, body)
		assert.Equal(t, http.StatusNoContent, status, body)
	})
	t.Run("fund P3 300", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/fund?playerId=P3&points=300", url))
		assert.NoError(t, err)
		assert.Empty(t, body)
		assert.Equal(t, http.StatusNoContent, status, body)
	})
	t.Run("fund P4 500", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/fund?playerId=P4&points=500", url))
		assert.NoError(t, err)
		assert.Empty(t, body)
		assert.Equal(t, http.StatusNoContent, status, body)
	})
	t.Run("fund P5 1000", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/fund?playerId=P5&points=1000", url))
		assert.NoError(t, err)
		assert.Empty(t, body)
		assert.Equal(t, http.StatusNoContent, status, body)
	})
	t.Run("announce T1 for 1000", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/announceTournament?tournamentId=1&deposit=1000", url))
		assert.NoError(t, err)
		assert.Empty(t, body)
		assert.Equal(t, http.StatusNoContent, status, body)
	})
	t.Run("join P5", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/joinTournament?tournamentId=1&playerId=P5", url))
		assert.NoError(t, err)
		assert.Empty(t, body)
		assert.Equal(t, http.StatusNoContent, status, body)
	})
	t.Run("join P1 with P2, P3, P4", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/joinTournament?tournamentId=1&playerId=P1&backerId=P2&backerId=P3&backerId=P4", url))
		assert.NoError(t, err)
		assert.Empty(t, body)
		assert.Equal(t, http.StatusNoContent, status, body)
	})
	t.Run("result T1 with P1", func(t *testing.T) {
		data := `{"tournamentId": 1, "winners": [{"playerId": "P1", "prize": 2000}]}`
		body, status, err := post(fmt.Sprintf("%s/resultTournament", url), data)
		assert.NoError(t, err)
		assert.Empty(t, body)
		assert.Equal(t, http.StatusNoContent, status, body)
	})
	t.Run("balance P1", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/balance?playerId=P1", url))
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		expected := `{"playerId": "P1", "balance": 550}`
		assert.JSONEq(t, expected, body)
	})
	t.Run("balance P2", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/balance?playerId=P2", url))
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		expected := `{"playerId": "P2", "balance": 550}`
		assert.JSONEq(t, expected, body)
	})
	t.Run("balance P3", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/balance?playerId=P3", url))
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		expected := `{"playerId": "P3", "balance": 550}`
		assert.JSONEq(t, expected, body)
	})
	t.Run("balance P4", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/balance?playerId=P4", url))
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		expected := `{"playerId": "P4", "balance": 750}`
		assert.JSONEq(t, expected, body)
	})
	t.Run("balance P5", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/balance?playerId=P5", url))
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		expected := `{"playerId": "P5", "balance": 0}`
		assert.JSONEq(t, expected, body)
	})
}

func TestFundConcurrency(t *testing.T) {
	_, url, cleanup := newServer(t)
	defer cleanup()

	t.Run("fund P1 0", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/fund?playerId=P1&points=0", url))
		assert.NoError(t, err)
		assert.Empty(t, body)
		assert.Equal(t, http.StatusNoContent, status, body)
	})

	t.Run("fund P1 10 concurrent", func(t *testing.T) {
		var wg sync.WaitGroup
		n := 10
		wg.Add(10)
		for i := 0; i < n; i++ {
			go func() {
				defer wg.Done()
				body, status, err := get(fmt.Sprintf("%s/fund?playerId=P1&points=10", url))
				assert.NoError(t, err)
				assert.Empty(t, body)
				assert.Equal(t, http.StatusNoContent, status, body)
			}()
		}
		wg.Wait()
	})

	t.Run("balance P1", func(t *testing.T) {
		body, status, err := get(fmt.Sprintf("%s/balance?playerId=P1", url))
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, status)
		expected := `{"playerId": "P1", "balance": 100}`
		assert.JSONEq(t, expected, body)
	})
}

func TestResultConcurrency(t *testing.T) {
	_, url, cleanup := newServer(t)
	defer cleanup()

	n := 4
	entryFee := 100
	initialBalance := n * entryFee

	// Create n players and n tournaments
	for i := 1; i <= n; i++ {
		body, status, err := get(fmt.Sprintf("%s/fund?playerId=P%d&points=%d", url, i, initialBalance))
		assert.NoError(t, err)
		assert.Empty(t, body)
		assert.Equal(t, http.StatusNoContent, status, body)

		body, status, err = get(fmt.Sprintf("%s/announceTournament?tournamentId=%d&deposit=%d", url, i, entryFee))
		assert.NoError(t, err)
		assert.Empty(t, body)
		assert.Equal(t, http.StatusNoContent, status, body)
	}

	// join all players to all tournaments
	for i := 1; i <= n; i++ {
		for j := 1; j <= n; j++ {
			body, status, err := get(fmt.Sprintf("%s/joinTournament?tournamentId=%d&playerId=P%d", url, i, j))
			assert.NoError(t, err)
			assert.Empty(t, body)
			assert.Equal(t, http.StatusNoContent, status, body)
		}
	}

	var winnersAsc []string
	var winnersDesc []string
	for i := 1; i <= n; i++ {
		winnersAsc = append(winnersAsc, fmt.Sprintf(`{"playerId": "P%d", "prize": %d}`, i, entryFee))
		winnersDesc = append(winnersDesc, fmt.Sprintf(`{"playerId": "P%d", "prize": %d}`, n-i+1, entryFee))
	}

	// announce winners in conflicting order to trigger deadlocks, example:
	//
	// tournament 1 winners P1, P2, P3
	// tournament 2 winners P3, P2, P1
	// tournament 3 winners P1, P2, P3
	// ...
	var results []string
	for i := 1; i <= n; i++ {
		var winners string
		if i%2 == 0 {
			winners = strings.Join(winnersAsc, ",")
		} else {
			winners = strings.Join(winnersDesc, ",")
		}
		results = append(results, fmt.Sprintf(`{"tournamentId": %d, "winners": [%s]}`, i, winners))
	}

	// execute tournaments resulting concurrently
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range results {
		go func(n int) {
			defer wg.Done()
			body, status, err := post(fmt.Sprintf("%s/resultTournament", url), results[n])
			assert.NoError(t, err)
			assert.Empty(t, body)
			assert.Equal(t, http.StatusNoContent, status, body)
		}(i)
	}
	wg.Wait()
}
