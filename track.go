package main

import (
	"log"
	"net/url"
	"strings"
	"time"

	"gopkg.in/kataras/iris.v6"
)

func track(c *iris.Context) {
	code := c.FormValue("t")
	log.Print(code)
	if code == "" {
		c.SetStatusCode(400)
		return
	}

	// store data to redis
	twodays := time.Hour * 48
	key := redisKeyFactory(code, time.Now().Format(DATEFORMAT))

	if c.GetCookie("_tcs") == "" {
		// new session, count it
		c.SetCookieKV("_tcs", "1")

		rds.Incr(key(SESSIONS))
		rds.Expire(key(SESSIONS), twodays)

		// save referrer only on new sessions
		uref, err := url.Parse(c.FormValue("r"))
		if err == nil {
			uref.Path = strings.TrimRight(uref.Path, "/") // strip ending slashes
			rds.HIncrBy(key(REFERRERS), uref.String(), 1)
			rds.Expire(key(REFERRERS), twodays)
		}
	}

	// count a pageview
	rds.Incr(key(PAGEVIEWS))
	rds.Expire(key(PAGEVIEWS), twodays)

	// save visited page
	upage, err := url.Parse(c.RequestHeader("Referer"))
	if err != nil {
		log.Print("invalid Referer: ", c.RequestHeader("Referer"), " - ", err)
		c.SetStatusCode(400)
		return
	}
	page := strings.TrimRight(upage.Path, "/")
	if upage.RawQuery != "" {
		page = page + "?" + upage.RawQuery
	}
	rds.HIncrBy(key(PAGES), page, 1)
	rds.Expire(key(PAGES), twodays)

	log.Print("tracked " + code)

	// no cache
	c.SetHeader("Cache-Control", "no-cache, no-store, must-revalidate")
	c.SetHeader("Pragma", "no-cache")
	c.SetHeader("Expires", "0")

	c.SetContentType("image/gif")
	c.SetStatusCode(200)
}
