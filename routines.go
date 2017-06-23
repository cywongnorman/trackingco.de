package main

import (
	"log"
	"sort"
	"strconv"
	"time"

	couchdb "github.com/fjl/go-couchdb"
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
}

func every8days() {
	downgradeAccountsInDebtForMoreThanAWeek()
	generateInvoices()
	notifyAccountsInDebt()
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

	sites, err := fetchPayingSites()
	if err != nil {
		log.Fatal("error fetching list of sites from postgres: ", err)
	}

	for _, site := range sites {
		log.Print("-------------")
		log.Print(" > site ", site.Code, " (", site.Name, "), from ", site.UserEmail, ":")

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

	sites, err := fetchPayingSites()
	if err != nil {
		log.Fatal("error fetching list of sites from postgres: ", err)
	}

	for _, site := range sites {
		log.Print("-------------")
		log.Print(" > site ", site.Code, " (", site.Name, "), from ", site.UserEmail, ":")

		// make a couchdb document representing a month, with data from day couchdb documents
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
	}
}

func fetchPayingSites() (sites []Site, err error) {
	err = pg.Raw(`
SELECT sites.* FROM sites
INNER JOIN users ON users.email = sites.user_email
WHERE plan > 0`).
		Scan(&sites)
	return sites, err
}

func deleteOlderDayStats() {
	today := presentDay()
	a100daysago := today.AddDate(0, 0, -100)
	a130daysago := today.AddDate(0, 0, -130)
	sites, err := fetchPayingSites()

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

func generateInvoices() {
	var res []struct {
		UserEmail string    `json:"user_email"`
		ExpiresAt time.Time `json:"expires_at"`
		Plan      float64   `json:"plan"`
	}

	// find all expired invoices
	err := pg.Raw(`
SELECT DISTINCT ON (user_email)
  user_email,
  time + due AS expires_at,
  plan
FROM balances
  INNER JOIN users ON user_email = users.email
WHERE
  due IS NOT NULL AND (time + due) <= now()
ORDER BY user_email, time DESC`).
		Scan(&res)

	if err != nil {
		log.Print("failed to fetch account data for generating invoices: ", err)
		return
	}

	// create new ones starting at the day those expired
	for _, row := range res {
		planValue := planValues[row.Plan]

		err := pg.Exec(`
INSERT INTO balances (user_email, time, delta, due)
VALUES (?, ?, ?, '1 month')
        `,
			row.UserEmail,
			row.ExpiresAt,
			-planValue)
		if err != nil {
			log.Print("failed to create invoice at account ", row, ": ", err)
		}
	}
}

func notifyAccountsInDebt() {
	var res []struct {
		UserEmail string  `json:"user_email"`
		Balance   float64 `json:"balance"`
	}

	// find any account in debt
	err := pg.Raw(`
SELECT user_email, balance FROM (
  SELECT user_email, sum(delta) AS balance
  FROM balances
  GROUP BY user_email
)s WHERE balance < 0
    `).Scan(&res)

	if err != nil {
		log.Print("failed to fetch accounts in debt: ", err)
		return
	}

	for _, row := range res {
		log.Print("notifying ", row.UserEmail, " for a debt balance of ", row.Balance)
		if err = sendMessage(
			row.UserEmail,
			"Payment reminder at tracking.code",
			`
Dear user `+row.UserEmail+`,

According to our records you have an outstanding unpaid balance of 
$ `+strconv.Itoa(int(row.Balance))+` on your account at trackingco.de.

We ask you to carry out the payment of the aforementioned amount using
one of the our payment options found at https://trackingco.de/account or
reply to this email if something is wrong or you want to say anything.

If the balance continues in an unpaid state for 7 days your account will
be automatically cancelled.

---

Giovanni T. Parra
trackingco.de
            `,
		); err != nil {
			log.Print("failed to send downgrade email: ", err)
		}
	}
}

func downgradeAccountsInDebtForMoreThanAWeek() {
	var res []struct {
		UserEmail string `json:"user_email"`
	}

	err := pg.Raw(`
SELECT email FROM (
  SELECT
    email,
    plan,
    (SELECT sum(delta) FROM balances WHERE user_email = users.email GROUP BY user_email) AS balance,
    (
      SELECT time + due FROM balances
      WHERE user_email = users.email AND due IS NOT NULL ORDER BY time DESC LIMIT 1
    ) AS expires_at
  FROM users
)d
WHERE plan > 0 AND (expires_at + '8 days') < now()
    `).Scan(&res)

	if err != nil {
		log.Print("failed to fetch accounts with debt for more than 8 days.") // tell the user 7 days.
		return
	}

	for _, row := range res {
		log.Print("downgrading account ", row.UserEmail, " to plan 0")
		tx := pg.Begin()
		tx.Exec("UPDATE users SET plan = 0 WHERE users.email = ?", row.UserEmail)
		tx.Exec(`
UPDATE balances SET delta = 0
WHERE user_email = ?
  AND id = (
    SELECT id FROM balances
    WHERE user_email = ? AND due IS NOT NULL
    ORDER BY time DESC LIMIT 1
  )i
        `, row.UserEmail, row.UserEmail)
		err := tx.Commit()
		if err != nil {
			log.Print("failed to downgrade account ", row.UserEmail)
			continue
		}

		if err = sendMessage(
			row.UserEmail,
			"Your account at trackingco.de was downgraded.",
			`
Dear user `+row.UserEmail+`,

Your account at https://trackingco.de/ was downgraded since we haven't
seen your last payment which was due 7 days ago.

Your analytics data stored at our servers wasn't yet deleted, but may
be deleted at any time from now.

---

Giovanni T. Parra
trackingco.de
            `,
		); err != nil {
			log.Print("failed to send downgrade email: ", err)
		}
	}
}
