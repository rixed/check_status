package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

type DB struct {
	conn *sql.DB
}

func InitDB(connStr string) (*DB, error) {
	conn, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("db ping: %w", err)
	}
	conn.SetMaxOpenConns(10)
	conn.SetMaxIdleConns(5)
	conn.SetConnMaxLifetime(30 * time.Minute)
	return &DB{conn: conn}, nil
}

func (d *DB) FetchWebsites() ([]Website, error) {
	rows, err := d.conn.Query(`SELECT id, url, name FROM websites ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sites []Website
	for rows.Next() {
		var w Website
		if err := rows.Scan(&w.ID, &w.URL, &w.Name); err != nil {
			return nil, err
		}
		sites = append(sites, w)
	}
	return sites, rows.Err()
}

func (d *DB) InsertCheck(websiteID int, isUp bool, statusCode, responseMs int) error {
	_, err := d.conn.Exec(
		`INSERT INTO checks (website_id, is_up, status_code, response_ms)
		 VALUES ($1, $2, $3, $4)`,
		websiteID, isUp, statusCode, responseMs,
	)
	return err
}

// FetchUptimeStats returns the number of up and total checks in the last 31 days.
// Each check represents one minute of observation.
func (d *DB) FetchUptimeStats(websiteID int) (upMins, totalMins int, err error) {
	row := d.conn.QueryRow(
		`SELECT
		    COUNT(*) FILTER (WHERE is_up) AS up_count,
		    COUNT(*)                      AS total_count
		 FROM checks
		 WHERE website_id = $1
		   AND checked_at >= NOW() - INTERVAL '31 days'`,
		websiteID,
	)
	err = row.Scan(&upMins, &totalMins)
	return
}

func (d *DB) FetchRecentChecks(websiteID, limit int) ([]Check, error) {
	rows, err := d.conn.Query(
		`SELECT id, website_id, checked_at, is_up, status_code, response_ms
		   FROM checks
		  WHERE website_id = $1
		  ORDER BY checked_at DESC
		  LIMIT $2`,
		websiteID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []Check
	for rows.Next() {
		var c Check
		if err := rows.Scan(&c.ID, &c.WebsiteID, &c.CheckedAt,
			&c.IsUp, &c.StatusCode, &c.ResponseMs); err != nil {
			return nil, err
		}
		checks = append(checks, c)
	}
	return checks, rows.Err()
}
