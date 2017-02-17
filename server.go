package main

import (
	"context"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/iris-contrib/graceful"
	"github.com/kataras/iris"
)

func runServer() {
	api := iris.New()
	api.Post("/_graphql", func(c *iris.Context) {
		var gqr GraphQLRequest
		err := c.ReadJSON(&gqr)
		if err != nil {
			log.Print("failed to read graphql request: ", err)
		}
		c.SetContentType("application/json")
		context := context.WithValue(context.TODO(), "loggeduser", s.LoggedAs)
		err = c.JSON(200, query(gqr, context))
		context.Done()
		if err != nil {
			log.Print("failed to marshal graphql response: ", err)
		}
	})

	api.Get("/t.gif", func(c *iris.Context) {
		code := c.FormValue("t")
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
		c.SetStatusCode(200)
	})

	api.Get("/", func(c *iris.Context) {
		c.ServeFile("client/index.html", false)
	})

	api.StaticServe("client")

	graceful.Run(":"+s.Port, time.Duration(10)*time.Second, api)
}
