package db

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/Masterminds/squirrel"
	"github.com/go-sql-driver/mysql"
)

type queryDecorator func(squirrel.SelectBuilder) squirrel.SelectBuilder

// Maximum sizes for MySQL/MariaDB text fields
const (
	TinyTextMaxLength   int = (1 << 8) - 1  // Max size of TINYTEXT, TINYBLOB fields
	TextMaxLength       int = (1 << 16) - 1 // Max size of TEXT, BLOB fields
	MediumTextMaxLength int = (1 << 24) - 1 // Max size of MEDIUMTEXT, MEDIUMBLOB fields
)

var ErrNotFound = errors.New("db: not found")
var ErrAlreadyExists = errors.New("db: already exists")

func Connect(dsn string) (*sql.DB, error) {
	dbh, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	if err := CreateSchema(dbh); err != nil {
		dbh.Close()
		return nil, err
	}
	return dbh, nil
}

func RecreateDB(dsn string) error {
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return nil
	}

	name := cfg.DBName
	cfg.DBName = ""

	dbh, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		return err
	}
	defer dbh.Close()

	_, err = dbh.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", name))
	if err != nil {
		return err
	}
	_, err = dbh.Exec(fmt.Sprintf("CREATE DATABASE %s", name))
	if err != nil {
		return err
	}
	return nil
}

func CreateSchema(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS player (
			player_id VARCHAR(64) NOT NULL,
			balance BIGINT UNSIGNED NOT NULL DEFAULT 0,
			PRIMARY KEY (player_id)
		)`,
		`CREATE TABLE IF NOT EXISTS tournament (
			tournament_id INT UNSIGNED NOT NULL AUTO_INCREMENT,
			entry_deposit BIGINT UNSIGNED NOT NULL DEFAULT 0,
			active BOOL NOT NULL DEFAULT 1,
			PRIMARY KEY (tournament_id)
		)`,
		`CREATE TABLE IF NOT EXISTS tournament_player (
			tournament_id INT UNSIGNED NOT NULL,
			player_id VARCHAR(64) NOT NULL,
			fee BIGINT NOT NULL DEFAULT 0,
			data BLOB NOT NULL DEFAULT "{}",
			PRIMARY KEY (tournament_id, player_id),
			KEY player_id (player_id),
			FOREIGN KEY tournament_player_fk_tournament_id (tournament_id) REFERENCES tournament (tournament_id),
			FOREIGN KEY tournament_player_fk_player_id (player_id) REFERENCES player (player_id)
		)`,
		`CREATE TABLE IF NOT EXISTS tournament_winner (
			tournament_id INT UNSIGNED NOT NULL,
			player_id VARCHAR(64) NOT NULL,
			prize BIGINT NOT NULL DEFAULT 0,
			data BLOB NOT NULL DEFAULT "{}",
			PRIMARY KEY (tournament_id, player_id),
			KEY player_id (player_id),
			FOREIGN KEY tournament_winner_fk_tournament_id_player_id (tournament_id, player_id) REFERENCES tournament_player (tournament_id, player_id)
		)`,
		/*
			`CREATE TABLE IF NOT EXISTS transfer_log (
				transfer_log_id INT UNSIGNED NOT NULL AUTO_INCREMENT,
				tstamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				player_id VARCHAR(64) NOT NULL,
				points BIGINT NOT NULL,
				op BINARY(1) NOT NULL,
				tournament_id INT UNSIGNED NULL,
				PRIMARY KEY (transfer_log_id),
				KEY (tstamp),
				KEY (player_id),
				KEY (tournament_id),
				FOREIGN KEY transfer_log_fk_player_id (player_id) REFERENCES account (player_id),
				FOREIGN KEY transfer_log_fk_tournament_id (tournament_id) REFERENCES tournament (tournament_id) ON DELETE SET NULL
			)`,
		*/
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			return err
		}
	}
	return nil
}

func Transaction(db *sql.DB, body func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := body(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
