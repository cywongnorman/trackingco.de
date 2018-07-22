package main

import (
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

func track(c *fasthttp.RequestCtx) {
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

	// tracking code
	code := string(c.FormValue("c"))
	if code == "" {
		logger.Warn().Msg("didn't send tracking code")
		c.Error("didn't send tracking code.", 400)
		return
	}

	logger = logger.With().Str("code", code).Logger()

	// page
	upage, err := url.Parse(string(c.Referer()))
	if err != nil {
		logger.Warn().Err(err).Str("ref", string(c.Referer())).
			Msg("invalid referer")
		c.Error("invalid Referer: "+string(c.Referer())+" - "+err.Error(), 400)
		return
	}
	page := strings.TrimRight(upage.Path, "/")
	if page == "" {
		page = "/"
	}
	if upage.RawQuery != "" {
		page = page + "?" + upage.RawQuery
	}

	logger = logger.With().Str("page", page).Logger()

	// points
	points, err := strconv.Atoi(string(c.FormValue("p")))
	if err != nil {
		// if a call to tc() is made with no arguments,
		// it means "p" is blank, so track as if point were 1.
		points = 1
	} else {
		// if tc() is called with a number of points as an argument,
		// "p" will have a value, which will be stored at `points`.
		// that means we shouldn't track a pageview.
		// pageviews are only tracked from blank tc() calls.
		page = ""
	}

	logger = logger.With().Int("points", points).Logger()

	// referrer
	referrer := string(c.FormValue("r")) // may be "". means <direct>.
	if referrer != "" {
		uref, err := url.Parse(referrer)
		if err == nil {
			// verify if referrer is on blacklist
			if _, blacklisted := blacklist[uref.Host]; blacklisted {
				log.Info().Str("ref", uref.Host).Msg("referrer on blacklist")

				// send fake/invalid hashid to spammer
				if hi, err := hso.Encode([]int{-1, randomNumber(999), randomNumber(99), 37}); err == nil {
					c.SetStatusCode(200)
					c.SetBody([]byte(hi))
					return
				} else {
					logger.Warn().Err(err).Msg("error encoding fake hashid")
					c.Error("error encoding fake hashid: "+err.Error(), 500)
					return
				}
			}

			uref.Path = strings.TrimRight(uref.Path, "/") // strip ending slashes
			if uref.Path == "" {
				uref.Path = "/"
			}
			referrer = uref.String()
		}
	}

	logger = logger.With().Str("ref", referrer).Logger()

	// session (a hashid that translates to a number, which is the offset for the array of points)
	var offset int
	var sessioncode int
	hi := c.UserValue("sessionhashid").(string)

	logger = logger.With().Str("session", hi).Logger()

	// try to decode (at first it should be an invalid string)
	if offsetarr, err := hso.DecodeWithError(hi); err == nil && len(offsetarr) == 2 {
		// success decoding, it is a _valid_ existing session
		offset = offsetarr[0]
		// this session code will be used to fetch the referrer for this session
		sessioncode = offsetarr[1]
		referrer = rds.Get("rs:" + strconv.Itoa(sessioncode)).Val()
	} else {
		// error decoding, so it is a new session
		offset = -1
		// this session code will be used to store the referrer for this session
		sessioncode = randomNumber(999999999)
		rds.Set("rs:"+strconv.Itoa(sessioncode), referrer, time.Hour*5)
	}

	logger = logger.With().Int("offset", offset).Logger()

	// store data to redis
	twodays := int(time.Hour * 48)
	key := redisKeyFactory(code, presentDay().Format(DATEFORMAT))

	result := rds.Eval(
		tracklua,
		[]string{
			key("p"), // KEYS[1]
			key("s"), // KEYS[2]
		},
		page,     // ARGV[1]
		referrer, // ARGV[2]
		offset,   // ARGV[3]
		twodays,  // ARGV[4]
		points,   // ARGV[5]
	)

	if val, err := result.Result(); err != nil {
		logger.Warn().Err(err).Msg("error executing track.lua")
		c.Error("error executing track.lua: "+err.Error(), 500)
	} else {
		offset = int(val.(int64))
	}

	// send session to user
	hi, err = hso.Encode([]int{offset, sessioncode})
	if err != nil {
		logger.Warn().Err(err).Msg("error encoding hashid for session offset")
		c.Error(
			"error encoding hashid for session offset "+string(offset)+": "+
				err.Error(),
			500,
		)
		return
	}
	c.SetStatusCode(200)
	c.SetBody([]byte(hi))

	logger.Info().Msg("tracked")
}
