package main

import (
	"encoding/json"
	"math/rand"
	"net/url"
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

func sessionsFromRedis(domain, day string) Day {
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

func urlHost(full string) string {
	if u, err := url.Parse(full); err == nil {
		return u.Host
	}
	if full == "<direct>" {
		return ""
	}
	return full
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

func herokuDomains(method, subpath string, data interface{}) (resp *herokuDomainResponse, err error) {
	client := &fasthttp.Client{}

	r := fasthttp.AcquireRequest()
	r.SetRequestURI("https://api.heroku.com/apps/" + s.HerokuAppName + "/domains" + subpath)
	r.Header.SetMethod(method)
	r.Header.Set("Accept", "application/vnd.heroku+json; version=3")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+s.HerokuToken)

	if data != nil {
		body, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}
		r.SetBody(body)
	}

	w := fasthttp.AcquireResponse()
	err = client.DoTimeout(r, w, time.Second*25)
	if err != nil {
		return nil, err
	}

	resp = &herokuDomainResponse{}
	err = json.Unmarshal(w.Body(), resp)
	return resp, err
}

type herokuDomainResponse struct {
	Id       string `json:"id"`
	Message  string `json:"message"`
	Hostname string `json:"hostname"`
	Status   string `json:"string"`
}

func sessionsFromScoremap(scoremap string) []int {
	l := len(scoremap)
	nsessions := (l - 1) / 2
	sessions := make([]int, nsessions)
	for s := 0; s < nsessions; s++ {
		if l >= s*2+3 {
			score, _ := strconv.Atoi(scoremap[s*2+1 : s*2+3])
			sessions[s] = score
		}
	}
	return sessions
}
