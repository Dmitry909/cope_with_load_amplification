package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
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

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	db, err = sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	sqlStatement := "SELECT max(short_id) FROM pastes"
	row := db.QueryRow(sqlStatement)
	if err := row.Scan(&nextShortID); err != nil {
		log.Fatalf("Unable to take last short_id from table. %v", err)
	}
	fmt.Println("found last short_id:", nextShortID)
}

func insertData(db *sql.DB, short_id int64, target_link string) bool {
	sqlStatement := "INSERT INTO pastes (short_id, target_link) VALUES ($1, $2)"

	select {
	case semaphore <- struct{}{}:
	default:
		return false
	}
	_, err := db.Exec(sqlStatement, short_id, target_link)
	<-semaphore

	if err != nil {
		log.Fatalf("Unable to execute insert query. %v", err)
	}

	cache.Add(short_id, target_link)
	return true
}

func readData(db *sql.DB, short_id int64) (string, bool) {
	if value, ok := cache.Get(short_id); ok {
		return value.(string), true
	}

	targetLink := ""
	sqlStatement := "SELECT target_link FROM pastes WHERE short_id=$1"

	select {
	case semaphore <- struct{}{}:
	default:
		return "", false
	}
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

	short_id := atomic.AddInt64(&nextShortID, 1)
	if !insertData(db, short_id, targetLink) {
		http.Error(w, "Too much DB load", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	response := map[string]int64{"short_id": short_id}
	json.NewEncoder(w).Encode(response)
}

func readHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	short_id, err := strconv.Atoi(r.URL.Query().Get("short_id"))
	if err != nil {
		http.Error(w, "Missing short_id", http.StatusBadRequest)
		return
	}

	targetLink, found := readData(db, int64(short_id))
	if !found {
		http.Error(w, "Short ID not found", http.StatusNotFound)
		return
	}

	fmt.Fprintf(w, "Target Link: %s\n", targetLink)
}

func main() {
	defer db.Close()

	http.HandleFunc("/write", writeHandler)
	http.HandleFunc("/read", readHandler)

	fmt.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
