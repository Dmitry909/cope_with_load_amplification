package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "dmitry"
	password = "pwd"
	dbname   = "db1"
)

var db *sql.DB

func insertData(db *sql.DB, short_id string, target_link string) {
	sqlStatement := "INSERT INTO pastes (short_id, target_link) VALUES ($1, $2)"
	_, err := db.Exec(sqlStatement, short_id, target_link)
	if err != nil {
		log.Fatalf("Unable to execute insert query. %v", err)
	}
}

func readData(db *sql.DB, short_id string) (string, bool) {
	targetLink := ""
	sqlStatement := "SELECT target_link FROM pastes WHERE short_id=$1"
	row := db.QueryRow(sqlStatement, short_id)
	err := row.Scan(&targetLink)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false
		}
		log.Fatalf("Unable to execute select query. %v", err)
	}
	return targetLink, true
}

var rowsInserted = 0

func main() {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	var err error
	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
}
