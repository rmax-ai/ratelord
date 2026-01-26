package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "./ratelord.db")
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

	var forecastComputed int
	err = db.QueryRow("SELECT count(*) FROM events WHERE event_type = 'forecast_computed'").Scan(&forecastComputed)
	if err != nil {
		log.Fatal(err)
	}

	var usageObserved int
	err = db.QueryRow("SELECT count(*) FROM events WHERE event_type = 'usage_observed'").Scan(&usageObserved)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Total events: %d\n", total)
	fmt.Printf("Provider poll observed events: %d\n", pollObserved)
	fmt.Printf("Usage observed events: %d\n", usageObserved)
	fmt.Printf("Forecast computed events: %d\n", forecastComputed)
}
