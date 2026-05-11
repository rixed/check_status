package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
)

// computePeriods groups consecutive same-status checks into periods.
// Input: checks newest-first (as returned by FetchRecentChecks).
// Output: periods oldest-first, capped at maxPeriods.
func computePeriods(checks []Check, maxPeriods int) []Period {
	if len(checks) == 0 {
		return nil
	}

	var periods []Period
	current := Period{
		IsUp:  checks[0].IsUp,
		Since: checks[0].CheckedAt,
		// Until is zero: this period is ongoing
	}

	for i := 1; i < len(checks); i++ {
		c := checks[i]
		if c.IsUp == current.IsUp {
			current.Since = c.CheckedAt
		} else {
			periods = append(periods, current)
			if len(periods) >= maxPeriods {
				break
			}
			current = Period{
				IsUp:  c.IsUp,
				Since: c.CheckedAt,
				Until: checks[i-1].CheckedAt,
			}
		}
	}
	if len(periods) < maxPeriods {
		periods = append(periods, current)
	}

	// Reverse so display reads left-to-right chronologically.
	for i, j := 0, len(periods)-1; i < j; i, j = i+1, j-1 {
		periods[i], periods[j] = periods[j], periods[i]
	}
	return periods
}

func formatMins(m int) string {
	d, h, min := m/1440, (m%1440)/60, m%60
	var parts []string
	if d > 0 {
		parts = append(parts, fmt.Sprintf("%dd", d))
	}
	if h > 0 {
		parts = append(parts, fmt.Sprintf("%dh", h))
	}
	if min > 0 {
		parts = append(parts, fmt.Sprintf("%dm", min))
	}
	if len(parts) == 0 {
		return "0m"
	}
	return strings.Join(parts, " ")
}

func uptimeClass(pct float64) string {
	switch {
	case pct >= 99.0:
		return "up"
	case pct >= 95.0:
		return "warn"
	default:
		return "dn"
	}
}

type pageData struct {
	RenderedAt string
	Sites      []SiteStatus
}

var dashboardTmpl = template.Must(
	template.New("dashboard").
		Funcs(template.FuncMap{
			"isZero":      func(t time.Time) bool { return t.IsZero() },
			"formatMins":  formatMins,
			"uptimeClass": uptimeClass,
		}).
		Parse(tmpl),
)

const tmpl = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta http-equiv="refresh" content="60">
  <title>Status</title>
  <style>
    body  { font-family: sans-serif; max-width: 1100px; margin: 2rem auto; padding: 0 1rem; color: #222; }
    h1    { font-size: 1.4rem; margin-bottom: 0.2rem; }
    small { color: #888; }
    table { border-collapse: collapse; width: 100%; margin-top: 1rem; }
    th, td { text-align: left; padding: 0.45rem 0.7rem; border-bottom: 1px solid #e0e0e0; vertical-align: top; }
    th    { background: #f5f5f5; font-weight: 600; }
    .up   { color: #2a7a2a; }
    .dn   { color: #c02020; }
    .warn { color: #b07000; }
    .dot  { font-size: 1.05rem; margin-right: 0.25rem; }
    .tl   { font-size: 0.82rem; }
    .sep  { color: #bbb; margin: 0 0.25rem; }
  </style>
</head>
<body>
  <h1>Service Status</h1>
  <p><small>Checks run every 60 s &mdash; page rendered at {{.RenderedAt}} UTC</small></p>
  <table>
    <thead>
      <tr>
        <th>Site</th>
        <th>Status</th>
        <th>Last checked (UTC)</th>
        <th>Timeline (oldest &rarr; newest)</th>
        <th>Up / Tested (last 31d)</th>
        <th>Uptime %</th>
      </tr>
    </thead>
    <tbody>
      {{range .Sites}}
      <tr>
        <td><a href="{{.Site.URL}}">{{.Site.Name}}</a></td>
        <td>
          {{if isZero .LastChecked}}
            <span class="dot" style="color:#bbb">&#9679;</span><span style="color:#bbb">no data</span>
          {{else if .CurrentlyUp}}
            <span class="dot up">&#9679;</span><span class="up">UP</span>
          {{else}}
            <span class="dot dn">&#9679;</span><span class="dn">DOWN</span>
          {{end}}
        </td>
        <td>
          {{if not (isZero .LastChecked)}}{{.LastChecked.Format "2006-01-02 15:04:05"}}{{else}}&mdash;{{end}}
        </td>
        <td class="tl">
          {{range $i, $p := .Periods}}
            {{if $i}}<span class="sep">&rarr;</span>{{end}}
            <span class="{{if $p.IsUp}}up{{else}}dn{{end}}">
              {{if $p.IsUp}}UP{{else}}DOWN{{end}}
              since {{$p.Since.Format "2006-01-02 15:04"}}{{if not (isZero $p.Until)}} until {{$p.Until.Format "2006-01-02 15:04"}}{{end}}
            </span>
          {{else}}
            <em style="color:#bbb">no data yet</em>
          {{end}}
        </td>
        <td style="white-space:nowrap">
          {{if eq .TestedMins 0}}&mdash;{{else}}{{formatMins .UptimeMins}} / {{formatMins .TestedMins}}{{end}}
        </td>
        <td style="white-space:nowrap">
          {{if eq .TestedMins 0}}&mdash;{{else}}<span class="{{uptimeClass .UptimePct}}">{{printf "%.1f%%" .UptimePct}}</span>{{end}}
        </td>
      </tr>
      {{end}}
    </tbody>
  </table>
</body>
</html>`

func RunServer(db *DB, addr string) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", makeHandler(db))
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func makeHandler(db *DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		sites, err := db.FetchWebsites()
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			log.Printf("handler: FetchWebsites: %v", err)
			return
		}

		statuses := make([]SiteStatus, 0, len(sites))
		for _, site := range sites {
			checks, err := db.FetchRecentChecks(site.ID, 40)
			if err != nil {
				log.Printf("handler: FetchRecentChecks(%d): %v", site.ID, err)
				statuses = append(statuses, SiteStatus{Site: site})
				continue
			}

			ss := SiteStatus{Site: site}
			if len(checks) > 0 {
				ss.CurrentlyUp = checks[0].IsUp
				ss.LastChecked = checks[0].CheckedAt
			}
			ss.Periods = computePeriods(checks, 20)

			upMins, totalMins, err := db.FetchUptimeStats(site.ID)
			if err != nil {
				log.Printf("handler: FetchUptimeStats(%d): %v", site.ID, err)
			} else {
				ss.UptimeMins = upMins
				ss.TestedMins = totalMins
				if totalMins > 0 {
					ss.UptimePct = float64(upMins) / float64(totalMins) * 100
				}
			}
			statuses = append(statuses, ss)
		}

		data := pageData{
			RenderedAt: time.Now().UTC().Format("2006-01-02 15:04:05"),
			Sites:      statuses,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := dashboardTmpl.Execute(w, data); err != nil {
			log.Printf("handler: template execute: %v", err)
		}
	}
}
