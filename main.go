package main

import (
	"log"
	"os"
	"time"
)

type Website struct {
	ID   int
	URL  string
	Name string
}

type Check struct {
	ID         int64
	WebsiteID  int
	CheckedAt  time.Time
	IsUp       bool
	StatusCode int
	ResponseMs int
}

// Period represents a consecutive run of the same up/down status.
// Until is zero for the most recent (ongoing) period.
type Period struct {
	IsUp  bool
	Since time.Time
	Until time.Time
}

type SiteStatus struct {
	Site        Website
	CurrentlyUp bool
	LastChecked time.Time
	Periods     []Period
	UptimeMins  int
	TestedMins  int
	UptimePct   float64
}

func main() {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := InitDB(connStr)
	if err != nil {
		log.Fatalf("cannot connect to database: %v", err)
	}

	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		addr = ":80"
	}

	go RunChecker(db, time.Minute)

	log.Printf("listening on %s", addr)
	RunServer(db, addr)
}
