package main

import (
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

// checkSite performs a single HTTP GET. A 3xx stops at the first response
// (the redirect target's health is irrelevant). 5xx and network errors are
// considered down; everything else is up.
func checkSite(url string) (isUp bool, statusCode int, responseMs int) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	start := time.Now()
	resp, err := client.Get(url)
	elapsed := time.Since(start)

	if err != nil {
		return false, 0, 0
	}
	defer resp.Body.Close()
	io.CopyN(io.Discard, resp.Body, 1<<20)

	statusCode = resp.StatusCode
	responseMs = int(elapsed.Milliseconds())
	isUp = statusCode >= 200 && statusCode < 500
	return
}

func RunChecker(db *DB, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	runChecks(db)

	for range ticker.C {
		runChecks(db)
	}
}

func runChecks(db *DB) {
	sites, err := db.FetchWebsites()
	if err != nil {
		log.Printf("checker: FetchWebsites: %v", err)
		return
	}

	var wg sync.WaitGroup
	for _, site := range sites {
		site := site
		wg.Add(1)
		go func() {
			defer wg.Done()
			isUp, code, ms := checkSite(site.URL)
			if err := db.InsertCheck(site.ID, isUp, code, ms); err != nil {
				log.Printf("checker: InsertCheck for %s: %v", site.URL, err)
			}
		}()
	}
	wg.Wait()
}
