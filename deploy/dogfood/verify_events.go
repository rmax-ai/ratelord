package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "deploy/dogfood/ratelord.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var total int
	err = db.QueryRow("SELECT count(*) FROM events").Scan(&total)
	if err != nil {
		log.Fatal(err)
	}

	var pollObserved int
	err = db.QueryRow("SELECT count(*) FROM events WHERE event_type = 'provider_poll_observed'").Scan(&pollObserved)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total events: %d\n", total)
	fmt.Printf("Provider poll observed events: %d\n", pollObserved)
}
