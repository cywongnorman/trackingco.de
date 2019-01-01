package main

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lucsky/cuid"
	"github.com/valyala/fasthttp"
)

func track(c *fasthttp.RequestCtx, session string) {
	// cors
	c.Response.Header.Add("Vary", "Origin")

	origin := c.Request.Header.Peek("Origin")
	if len(origin) > 0 {
		c.Response.Header.AddBytesV("Access-Control-Allow-Origin", origin)
	} else {
		c.Response.Header.Add("Access-Control-Allow-Origin", "*")
	}

	// no cache
	c.Response.Header.Add("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Response.Header.Add("Pragma", "no-cache")
	c.Response.Header.Add("Expires", "0")

	logger := log.With().Logger()

	upage, err := url.Parse(string(c.Referer()))
	if err != nil {
		logger.Warn().Err(err).Str("ref", string(c.Referer())).
			Msg("invalid referer")
		c.Error("invalid Referer: "+string(c.Referer())+" - "+err.Error(), 400)
		return
	}

	// domain
	domain := strings.TrimPrefix(upage.Hostname(), "www.")
	logger = logger.With().Str("domain", domain).Logger()

	// event
	var event interface{}

	if points, err := strconv.Atoi(string(c.FormValue("p"))); err != nil {
		// if a call to tc() is made with no arguments,
		// it means "p" is blank, so track a pageview (equivalent to 1 point).
		page := strings.TrimRight(upage.Path, "/")
		if page == "" {
			page = "/"
		}
		if upage.RawQuery != "" {
			page = page + condenseQuery(upage.Query())
		}
		logger = logger.With().Str("page", page).Logger()

		event = page
	} else {
		// if tc() is called with a number of points as an argument,
		// "p" will have a value, which will be stored at `points`.
		// that means we shouldn't track a pageview.
		// pageviews are only tracked from blank tc() calls.
		logger = logger.With().Int("points", points).Logger()

		event = points
	}

	// plumbing
	threedays := time.Hour * 72
	today := presentDay().Format(DATEFORMAT)
	keyfn := redisKeyFactory(domain, today)

	// temp
	code := string(c.FormValue("c"))
	_, err = pg.Exec(`INSERT INTO temp_migration VALUES ($1, $2) ON CONFLICT (domain, code) DO NOTHING`, domain, code)
	if err != nil {
		log.Error().Err(err).Msg("temp")
	}

	// referrer
	referrer := string(c.FormValue("r")) // may be "". means <direct>.
	if referrer != "" {
		uref, err := url.Parse(referrer)
		if err == nil {
			// verify if referrer is on blacklist
			if _, blacklisted := blacklist[uref.Host]; blacklisted {
				log.Info().Str("ref", uref.Host).Msg("referrer on blacklist")

				// send fake/invalid cuid to spammer
				session = "z" + cuid.New()
				goto end
			}

			// process (turns https://x.com/plic/?xyz=q&uel=2 into x.com/plic?{xyz,uel})
			uref.Path = strings.TrimRight(uref.Path, "/") // strip ending slashes
			if uref.Path == "" {
				uref.Path = "/"
			}
			referrer = uref.Hostname() + uref.Path
			if uref.RawQuery != "" {
				referrer = referrer + condenseQuery(uref.Query())
			}
		}
	}

	logger = logger.With().
		Str("ref", referrer).
		Str("session", session).Logger()

	if session[0] != 'c' || strings.Index(session, "-") != -1 {
		// not a valid cuid, means it's the first visit of session
		// create session code
		session = cuid.New()
		// send the referrer first
		err = rds.RPush(keyfn(session), referrer, event).Err()
	} else {
		err = rds.RPushX(keyfn(session), event).Err()
	}

	// expire session data
	rds.Expire(keyfn(session), threedays)

	// add this domain to the list of domains that should be compiled today
	rds.SAdd("compile:"+today, domain)
	rds.Expire("compile:"+today, threedays)

	if err != nil {
		logger.Warn().Err(err).Msg("error tracking")
		c.Error("error tracking: "+err.Error(), 500)
	}

end:
	// send session cuid to user
	c.SetStatusCode(200)
	c.SetBody([]byte(session))

	logger.Info().Msg("tracked")
}
