package main

import (
	"encoding/json"
	"math/rand"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/jmoiron/sqlx/types"
	"github.com/valyala/fasthttp"
)

const (
	DATEFORMAT  = "20060102"
	MONTHFORMAT = "200601"
)

func presentDay() time.Time {
	now := time.Now().UTC()
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location())
}

func makeBaseKey(code, day string) string { return code + ":" + day }
func redisKeyFactory(code, day string) func(string) string {
	basekey := makeBaseKey(code, day)
	return func(subkey string) string {
		return basekey + ":" + subkey
	}
}
func makeMonthKey(code, month string) string { return code + "." + month }

func randomNumber(r int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(r)
}

func dayFromRedis(domain, day string) Day {
	var sessions []Session
	scankey := redisKeyFactory(domain, day)("*")

	iter := rds.Scan(0, scankey, 100).Iterator()
	for iter.Next() {
		sessionkey := iter.Val()
		events, err := rds.LRange(sessionkey, 0, -1).Result()
		if err != nil {
			log.Error().Str("skey", sessionkey).Err(err).
				Msg("error reading session from redis")
			continue
		}

		session := Session{
			Referrer: events[0],
		}
		for _, event := range events[1:] {
			if points, err := strconv.Atoi(event); err == nil {
				session.Events = append(session.Events, points)
			} else {
				session.Events = append(session.Events, event)
			}
		}
		sessions = append(sessions, session)
	}
	if err := iter.Err(); err != nil {
		log.Error().Str("key", scankey).Err(err).Msg("error scanning from redis")
	}

	var rawsessions types.JSONText
	rawsessions, _ = json.Marshal(sessions)

	return Day{
		Day:         day,
		RawSessions: rawsessions,
		sessions:    sessions,
	}
}

func deleteDayFromRedis(domain, day string) error {
	scankey := redisKeyFactory(domain, day)("*")

	var sessionkeys []string

	iter := rds.Scan(0, scankey, 100).Iterator()
	for iter.Next() {
		sessionkey := iter.Val()
		sessionkeys = append(sessionkeys, sessionkey)
	}

	return rds.Del(sessionkeys...).Err()
}

func condenseQuery(query url.Values) string {
	// if there's any querystring we'll keep it, but not its value
	// it will be something like /user?{id,page}, so in case there's an
	// adwords or similar stuff happening we'll see just /?{utm_source}
	querykeys := make([]string, len(query))
	var i = 0
	for qk, _ := range query {
		querykeys[i] = qk
		i++
	}
	sort.Strings(querykeys)
	return "?" + "{" + strings.Join(querykeys, ",") + "}"
}

func buildReferrerBlacklist() map[string]bool {
	refmap := make(map[string]bool)

	lines := ""
	client := &fasthttp.Client{Name: "Mozilla/5.0 (X11; Linux i686) AppleWebKit/537.36 (KHTML, like Gecko) Ubuntu Chromium/56.0.2924.76 Chrome/56.0.2924.76 Safari/537.36"}
	for _, u := range []string{
		"https://raw.githubusercontent.com/piwik/referrer-spam-blacklist/master/spammers.txt",
		"https://raw.githubusercontent.com/ddofborg/analytics-ghost-spam-list/master/adwordsrobot.com-spam-list.txt",
	} {
		r := fasthttp.AcquireRequest()
		r.SetRequestURI(u)

		w := fasthttp.AcquireResponse()
		err := client.DoTimeout(r, w, time.Second*25)
		if err != nil {
			continue
		}

		lines += string(w.Body())
		lines += "\n"
	}

	for _, line := range strings.Split(lines, "\n") {
		refmap[strings.TrimSpace(line)] = true
	}

	if doc, err := goquery.NewDocument("https://referrerspamblocker.com/blacklist"); err == nil {
		doc.Find(".blacklist li").Each(func(i int, s *goquery.Selection) {
			refmap[strings.TrimSpace(s.Text())] = true
		})
	}

	return refmap
}
