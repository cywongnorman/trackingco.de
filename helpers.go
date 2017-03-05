package main

import (
	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/speps/go-hashids"
	"github.com/valyala/fasthttp"
)

const DATEFORMAT = "20060102"

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

func makeCodeForUser(userId string) string {
	userNumber := 0
	for _, char := range userId {
		userNumber += int(char)
	}

	hd := hashids.NewData()
	hd.MinLength = 5
	hd.Alphabet = "abcdefghijklmnopqrstuvwxyz1234567890"
	h := hashids.NewWithData(hd)
	r, _ := h.Encode([]int{userNumber, randomNumber(9999)})
	return r
}

func randomNumber(r int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(r)
}

func compendiumFromRedis(code, day string) Compendium {
	key := redisKeyFactory(code, day)

	compendium := Compendium{
		Id:       makeBaseKey(code, day),
		Sessions: make(map[string]string),
		Pages:    make(map[string]int),
	}

	if val, err := rds.HGetAll(key("s")).Result(); err == nil {
		for k, v := range val {
			compendium.Sessions[k] = v
		}
	}
	if val, err := rds.HGetAll(key("p")).Result(); err == nil {
		for k, v := range val {
			if count, err := strconv.Atoi(v); err == nil {
				compendium.Pages[k] = count
			}
		}
	}

	return compendium
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
	lines += doRequest("https://raw.githubusercontent.com/ddofborg/analytics-ghost-spam-list/master/adwordsrobot.com-spam-list.txt")
	lines += "\n"
	lines += doRequest("https://raw.githubusercontent.com/piwik/referrer-spam-blacklist/master/spammers.txt")

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

func doRequest(u string) string {
	req := fasthttp.AcquireRequest()
	req.SetRequestURI(u)

	resp := fasthttp.AcquireResponse()
	client := &fasthttp.Client{
		Name: "Mozilla/5.0 (X11; Linux i686) AppleWebKit/537.36 (KHTML, like Gecko) Ubuntu Chromium/56.0.2924.76 Chrome/56.0.2924.76 Safari/537.36",
	}
	err := client.DoTimeout(req, resp, time.Second*25)
	if err != nil {
		return ""
	}

	return string(resp.Body())
}
