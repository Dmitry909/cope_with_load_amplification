package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"

	lru "github.com/hashicorp/golang-lru"
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
var nextShortID int64
var cache *lru.Cache

var maxConcurrentDBQueries = 5000
var semaphore = make(chan struct{}, maxConcurrentDBQueries)

func init() {
	var err error
	cache, err = lru.New(1000000)
	if err != nil {
		log.Fatalf("Failed to create LRU cache: %v", err)
	}
}

func insertData(db *sql.DB, short_id string, target_link string) {
	sqlStatement := "INSERT INTO pastes (short_id, target_link) VALUES ($1, $2)"

	semaphore <- struct{}{}
	_, err := db.Exec(sqlStatement, short_id, target_link)
	<-semaphore

	if err != nil {
		log.Fatalf("Unable to execute insert query. %v", err)
	}

	cache.Add(short_id, target_link)
}

func readData(db *sql.DB, short_id string) (string, bool) {
	if value, ok := cache.Get(short_id); ok {
		return value.(string), true
	}

	targetLink := ""
	sqlStatement := "SELECT target_link FROM pastes WHERE short_id=$1"

	semaphore <- struct{}{}
	row := db.QueryRow(sqlStatement, short_id)
	<-semaphore

	err := row.Scan(&targetLink)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", false
		}
		log.Fatalf("Unable to execute select query. %v", err)
	}

	cache.Add(short_id, targetLink)

	return targetLink, true
}

func writeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	targetLink := r.FormValue("target_link")
	if targetLink == "" {
		http.Error(w, "Missing target_link", http.StatusBadRequest)
		return
	}

	shortID := fmt.Sprintf("%d", atomic.AddInt64(&nextShortID, 1))
	insertData(db, shortID, targetLink)

	w.Header().Set("Content-Type", "application/json")
	response := map[string]string{"short_id": shortID}
	json.NewEncoder(w).Encode(response)
}

func readHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	shortID := r.URL.Query().Get("short_id")
	if shortID == "" {
		http.Error(w, "Missing short_id", http.StatusBadRequest)
		return
	}

	targetLink, found := readData(db, shortID)
	if !found {
		http.Error(w, "Short ID not found", http.StatusNotFound)
		return
	}

	fmt.Fprintf(w, "Target Link: %s\n", targetLink)
}

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

	http.HandleFunc("/write", writeHandler)
	http.HandleFunc("/read", readHandler)

	fmt.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
