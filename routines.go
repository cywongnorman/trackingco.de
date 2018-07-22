package main

import (
	"sort"
	"time"

	"github.com/ogier/pflag"
	"github.com/timjacobi/go-couchdb"
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
	deleteOlderDayStats()
}

func compileDayStats(day string) {
	log.Print("-- running compileDayStats routine for ", day, ".")

	sites, err := fetchSites()
	if err != nil {
		log.Fatal().Err(err).Msg("error fetching list of sites from postgres")
	}

	for _, site := range sites {
		log.Print("-------------")
		log.Print(" > site ", site.Code, " (", site.Name, "), from ", site.Owner, ":")

		// make a couchdb document representing a day, with data from redis
		stats := dayFromRedis(site.Code, day)
		log.Print(stats)

		// check for zero-stats (to save disk space we won't store these)
		if len(stats.Sessions) == 0 && len(stats.Pages) == 0 {
			log.Print("   : skipped saving because everything is zero.")
			continue
		}

		// save on couch
		if _, err = couch.Put(stats.Id, stats, ""); err != nil {
			log.Print("   : failed to save stats on couch: ", err)
			continue
		}
		log.Print("   : saved on couch.")
	}
}

func compileMonthStats(month string) {
	log.Print("-- running compileMonthStats routine for ", month, ".")

	sites, err := fetchSites()
	if err != nil {
		log.Fatal().Err(err).Msg("error fetching list of sites from postgres")
	}

	for _, site := range sites {
		log.Print("-------------")
		log.Print(" > site ", site.Code, " (", site.Name, "), from ", site.Owner, ":")

		// make a couchdb document representing a month,
		// with data from day couchdb documents
		stats := Month{
			Id:           makeMonthKey(site.Code, month),
			TopReferrers: make(map[string]int, 10),
			TopPages:     make(map[string]int, 10),
		}

		// first fetch the data from couchdb
		res := CouchDBDayResults{}
		err := couch.AllDocs(&res, couchdb.Options{
			"startkey":     makeBaseKey(site.Code, month+"01"),
			"endkey":       makeBaseKey(site.Code, month+"31"),
			"include_docs": true,
		})
		if err != nil {
			log.Print("   : failed to fetch days from couchdb: ", err)
			continue
		}
		days := res.toDayList()

		// reduce everything
		allpages := make(map[string]int)
		allreferrers := make(map[string]int)
		sessionswithscore1 := 0
		for _, day := range days {
			for page, count := range day.Pages {
				allpages[page] += count
				stats.Pageviews += count
			}
			for referrer, scoremap := range day.Sessions {
				sessions := sessionsFromScoremap(scoremap)
				stats.Sessions += len(sessions)
				allreferrers[referrer] += len(sessions)
				for _, score := range sessions {
					stats.Score += score
					if score == 1 {
						sessionswithscore1++
					}
				}
			}
		}
		if stats.Sessions > 0 {
			stats.BounceRate = 10000 * sessionswithscore1 / stats.Sessions
		}

		pageEntries := EntriesFromMap(allpages)
		if len(pageEntries) > 10 {
			sort.Sort(sort.Reverse(EntrySort(pageEntries)))
			pageEntries = pageEntries[:10]
		}
		stats.TopPages = MapFromEntries(pageEntries)

		referrerEntries := EntriesFromMap(allreferrers)
		if len(referrerEntries) > 10 {
			sort.Sort(sort.Reverse(EntrySort(referrerEntries)))
			referrerEntries = referrerEntries[:10]
		}
		stats.TopReferrers = MapFromEntries(referrerEntries)

		log.Print(stats)

		// check for zero-stats (to save disk space we won't store these)
		if stats.Sessions == 0 && stats.Pageviews == 0 {
			log.Print("   : skipped saving because everything is zero.")
			continue
		}

		// save on couch
		if _, err = couch.Put(stats.Id, stats, ""); err != nil {
			log.Print("   : failed to save stats on couch: ", err)
			continue
		}
		log.Print("   : saved on couch.")

		// track this user as having used the service this month
		_, err = pg.Exec(`
UPDATE users
SET months_using = array_append(array_remove(months_using, $2), $2)
WHERE id = $1
        `, site.Owner, month)
		if err != nil {
			log.Print("   : failed to set months_using: ", err)
		}
	}
}

func fetchSites() (sites []Site, err error) {
	err = pg.Select(&sites, `SELECT * FROM sites`)
	return sites, err
}

func deleteOlderDayStats() {
	today := presentDay()
	a100daysago := today.AddDate(0, 0, -100)
	a130daysago := today.AddDate(0, 0, -130)
	sites, err := fetchSites()

	if err != nil {
		log.Print("error fetching all sites for deleting older day stats: ", err)
		return
	}

	for _, site := range sites {
		cur := a100daysago
		for {
			deleteDayFromRedis(site.Code, cur.Format(DATEFORMAT))
			if cur.Before(a130daysago) {
				break
			}
			cur = cur.AddDate(0, 0, -1)
		}
	}
}
