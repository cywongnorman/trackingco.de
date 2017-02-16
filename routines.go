package main

import (
	"log"
	"strconv"
	"time"

	"github.com/kr/pretty"
	"github.com/ogier/pflag"
)

func daily() {
	var day string
	pflag.StringVar(&day, "day",
		time.Now().AddDate(0, 0, -1).Format(DATEFORMAT),
		"compile stats from which day?")
	pflag.Parse()

	log.Print("-- running daily routine for ", day, ".")

	var sites []Site
	if err = pg.Model(Site{}).Scan(&sites); err != nil {
		log.Fatal("error fetching list of sites from postgres: ", err)
	}

	for _, site := range sites {
		log.Print(" > site ", site.Code, " (", site.Name, "), from ", site.UserId, ":")
		key := redisKeyFactory(site.Code, day)

		stats := Compendium{
			Id:        makeBaseKey(site.Code, day),
			Referrers: make(map[string]int),
			Pages:     make(map[string]int),
		}

		// grab stats from redis
		if val, err := rds.Get(key(SESSIONS)).Int64(); err == nil {
			stats.Sessions = int(val)
		}
		if val, err := rds.Get(key(PAGEVIEWS)).Int64(); err == nil {
			stats.Pageviews = int(val)
		}
		if val, err := rds.HGetAll(key(REFERRERS)).Result(); err == nil {
			for k, v := range val {
				if count, err := strconv.Atoi(v); err == nil {
					stats.Referrers[k] = count
				}
			}
		}
		if val, err := rds.HGetAll(key(PAGES)).Result(); err == nil {
			for k, v := range val {
				if count, err := strconv.Atoi(v); err == nil {
					stats.Pages[k] = count
				}
			}
		}

		pretty.Log(stats)

		// check for zero-stats (to save disk space we won't store these)
		if stats.Sessions == 0 && stats.Pageviews == 0 && len(stats.Referrers) == 0 && len(stats.Pages) == 0 {
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

func monthly() {}
