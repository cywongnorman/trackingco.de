package main

import (
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/qiangxue/fasthttp-routing"
)

func track(c *routing.Context) error {
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

	// tracking code
	code := string(c.FormValue("c"))
	if code == "" {
		return HTTPError{400, "didn't send tracking code."}
	}

	// points
	points, err := strconv.Atoi(string(c.FormValue("p")))
	if err != nil || points < 1 {
		points = 1
	}

	// page
	upage, err := url.Parse(string(c.Referer()))
	if err != nil {
		return HTTPError{
			400,
			"invalid Referer: " + string(c.Referer()) + " - " + err.Error(),
		}
	}
	page := strings.TrimRight(upage.Path, "/")
	if page == "" {
		page = "/"
	}
	if upage.RawQuery != "" {
		page = page + "?" + upage.RawQuery
	}

	// referrer
	referrer := string(c.FormValue("r")) // may be "". means <direct>.
	if referrer != "" {
		uref, err := url.Parse(referrer)
		if err == nil {
			// verify if referrer is on blacklist
			if _, blacklisted := blacklist[uref.Host]; blacklisted {
				log.Print("referrer on blacklist: ", uref.Host)

				// send fake/invalid hashid to spammer
				if hi, err := hso.Encode([]int{-1, randomNumber(999), randomNumber(99), 37}); err == nil {
					c.SetStatusCode(200)
					c.SetBody([]byte(hi))
					return nil
				} else {
					return HTTPError{500, "error encoding fake hashid: " + err.Error()}
				}
			}

			uref.Path = strings.TrimRight(uref.Path, "/") // strip ending slashes
			if uref.Path == "" {
				uref.Path = "/"
			}
			referrer = uref.String()
		}
	}

	// session (a hashid that translates to a number, which is the offset for the array of points)
	var offset int
	var sessioncode int
	hi := c.Param("sessionhashid")

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
		return HTTPError{500, "error executing track.lua: " + err.Error()}
	} else {
		offset = int(val.(int64))
	}

	// send session to user
	hi, err = hso.Encode([]int{offset, sessioncode})
	if err != nil {
		return HTTPError{
			500,
			"error encoding hashid for session offset " + string(offset) + ": " + err.Error(),
		}
	}
	c.SetStatusCode(200)
	c.SetBody([]byte(hi))

	log.Print("tracked ", code, " ", referrer, " ", hi, " ", offset, " ", page)
	return nil
}
