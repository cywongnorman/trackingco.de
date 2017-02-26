package main

import (
	"log"

	"github.com/ogier/pflag"
)

func daily() {
	var day string
	pflag.StringVar(&day, "day",
		presentDay().Format(DATEFORMAT),
		"compile stats from which day?")
	pflag.Parse()

	log.Print("-- running daily routine for ", day, ".")

	var sites []Site
	if err = pg.Model(Site{}).Scan(&sites); err != nil {
		log.Fatal("error fetching list of sites from postgres: ", err)
	}

	for _, site := range sites {
		log.Print("-------------")
		log.Print(" > site ", site.Code, " (", site.Name, "), from ", site.UserId, ":")

		stats := compendiumFromRedis(site.Code, day)
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

func monthly() {}
