package main

import (
	"strconv"
	"time"

	"github.com/ogier/pflag"
)

func daily() {
	var day string
	pflag.StringVar(&day, "day",
		presentDay().Format(DATEFORMAT),
		"which day is today? (will compile for yesterday)")
	pflag.Parse()

	log.Print("# running daily routine for day ", day, ".")

	parsed, err := time.Parse(DATEFORMAT, day)
	if err != nil {
		log.Print("  # failed to parse day ", day)
		return
	}
	yesterday := parsed.AddDate(0, 0, -1).Format(DATEFORMAT)
	compileDayStats(yesterday)

	moreThan90 := parsed.AddDate(0, -1, -90).Format(DATEFORMAT)
	deleteDaysOlderThan(moreThan90)
}

func monthly() {
	var month string
	pflag.StringVar(&month, "month",
		presentDay().Format(MONTHFORMAT),
		"which month are we in? (will compile for previous month)")
	pflag.Parse()

	log.Print("# running monthly routine for month ", month, ".")

	parsed, err := time.Parse(MONTHFORMAT, month)
	if err != nil {
		log.Print("  # failed to parse month ", month)
		return
	}
	lastmonth := parsed.AddDate(0, -1, 0).Format(MONTHFORMAT)
	compileMonthStats(lastmonth)
}

func compileDayStats(day string) {
	log.Print("-- running compileDayStats routine for ", day, ".")

	domains, err := rds.SMembers("compile:" + day).Result()
	if err != nil {
		log.Fatal().Err(err).Str("today", day).
			Msg("error todays domains from redis.")
	}

	for _, domain := range domains {
		log.Print("-------------")
		log.Print(" > site ", domain)

		// grab all data from redis
		day := dayFromRedis(domain, day)
		log.Print(day)

		// check for zero-day (to save disk space we won't store these)
		if len(day.RawSessions) < 3 {
			log.Print("   : skipped saving because everything is zero.")
			continue
		}

		// save on postgres
		if _, err = pg.Exec(`
INSERT INTO days
  (domain, day, sessions)
VALUES ($1, $2, $3)
        `, domain, day.Day, day.RawSessions); err != nil {
			log.Print("   : failed to save day on postgres: ", err)
			continue
		}
		log.Print("   : saved on postgres.")
	}
}

func compileMonthStats(month string) {
	log.Print("-- running compileMonthStats routine for ", month, ".")
	monthstart := month + "01"
	monthend := month + "31"

	var domains []string
	err := pg.Select(&domains, `SELECT DISTINCT domain FROM days`)
	if err != nil {
		log.Fatal().Err(err).Str("month", month).
			Msg("error fetching domains from postgres.")
	}

	for _, domain := range domains {
		log.Print("-------------")
		log.Print(" > site ", domain)

		_, err := pg.Exec(`
WITH sessions AS (
  SELECT jsonb_array_elements(sessions) AS session
  FROM days
  WHERE domain = $1 AND day >= $2 AND day <= $3
), events AS (
  SELECT
    session->>'referrer' AS referrer,
    jsonb_array_elements(session->'events') AS event
  FROM sessions
), referrers AS (
  SELECT session->>'referrer' AS referrer FROM sessions
), nbounces AS (
  SELECT count(*) AS nbounces
  FROM sessions
  WHERE jsonb_array_length(session->'events') = 1
    AND (jsonb_typeof(session->'events'->0) = 'string' OR (session->'events'->0)::text::int = 1)
), pages AS (
  SELECT event#>>'{}' AS page
  FROM events
  WHERE jsonb_typeof(event) = 'string'
), score AS (
  SELECT
    sum(CASE WHEN jsonb_typeof(event) = 'number' THEN event::text::int ELSE 1 END)
      AS score
  FROM events
), top_pages AS (
  SELECT jsonb_object_agg(page, count) AS top_pages FROM (
    SELECT page, count(*)
    FROM pages
    GROUP BY page
    ORDER BY count DESC
    LIMIT 10
  )x
), top_referrers AS (
  SELECT jsonb_object_agg(referrer, count) AS top_referrers FROM (
    SELECT referrer, count(*)
    FROM referrers
    GROUP BY referrer
    ORDER BY count DESC
    LIMIT 10
  )x
), top_referrers_scores AS (
  SELECT jsonb_object_agg(referrer, sum) AS top_referrers_scores FROM (
    SELECT referrer, sum(CASE WHEN jsonb_typeof(event) = 'number' THEN event::text::int ELSE 1 END)
    FROM events
    GROUP BY referrer
    ORDER BY sum DESC
    LIMIT 10
  )x
), agg AS (
  SELECT
    (SELECT score FROM score) AS score,
    (SELECT nbounces FROM nbounces) AS nbounces,
    (SELECT count(*) FROM sessions) AS nsessions,
    (SELECT count(*) FROM pages) AS npageviews,
    (SELECT coalesce(top_referrers, '{}') FROM top_referrers) AS top_referrers,
    (SELECT coalesce(top_referrers_scores, '{}') FROM top_referrers_scores) AS top_referrers_scores,
    (SELECT coalesce(top_pages, '{}') FROM top_pages) AS top_pages
)

INSERT INTO months
  (domain, month, score, nbounces, nsessions, npageviews, top_referrers, top_referrers_scores, top_pages)
  SELECT
    $1, $4, score, nbounces, nsessions, npageviews, top_referrers, top_referrers_scores, top_pages
  FROM agg
        `, domain, monthstart, monthend, month)
		if err != nil {
			log.Print("   : failed to build monthly stats: ", err)
			continue
		}
		log.Print("   : monthly stats built.")

	}
}

func deleteDaysOlderThan(dayInThePast string) {
	log.Print("-- deleting days older than " + dayInThePast)
	r, err := pg.Exec(`
DELETE FROM days
WHERE day <= $1
    `, dayInThePast)
	if err != nil {
		log.Print("  : failed to delete old days: ", err)
		return
	}

	rows, err := r.RowsAffected()
	if err != nil {
		log.Print("  : failed to get number of affected rows: ", err)
		return
	}

	log.Print("  : deleted " + strconv.Itoa(int(rows)) + " old days.")
}
