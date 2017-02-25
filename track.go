package main

import (
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gopkg.in/kataras/iris.v6"
)

func track(c *iris.Context) {
	// cors
	c.SetHeader("Vary", "Origin")
	c.SetHeader("Access-Control-Allow-Origin", c.RequestHeader("Origin"))

	// no cache
	c.SetHeader("Cache-Control", "no-cache, no-store, must-revalidate")
	c.SetHeader("Pragma", "no-cache")
	c.SetHeader("Expires", "0")

	// tracking code
	code := c.FormValue("c")
	if code == "" {
		log.Print("didn't send tracking code.")
		sendBlank(c)
		return
	}

	// points
	points, err := strconv.Atoi(c.FormValue("p"))
	if err != nil || points < 1 {
		points = 1
	}

	// page
	upage, err := url.Parse(c.RequestHeader("Referer"))
	if err != nil {
		log.Print("invalid Referer: ", c.RequestHeader("Referer"), " - ", err)
		sendBlank(c)
		return
	}
	page := strings.TrimRight(upage.Path, "/")
	if page == "" {
		page = "/"
	}
	if upage.RawQuery != "" {
		page = page + "?" + upage.RawQuery
	}

	// referrer
	referrer := c.FormValue("r") // may be "". means <direct>.
	if referrer != "" {
		uref, err := url.Parse(referrer)
		if err == nil {
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

	// try to decode (at first it may be an invalid string)
	if offsetarr, err := hso.DecodeWithError(hi); err == nil && len(offsetarr) == 2 {
		// success decoding, it is a _valid_ existing session
		offset = offsetarr[0]
		// this session code will be used to fetch the referrer for this session
		sessioncode = offsetarr[1]
		referrer = rds.Get("rs:" + strconv.Itoa(sessioncode)).Val()
	} else {
		// new session
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
		log.Print("error executing track.lua: ", err)
		sendBlank(c)
		return
	} else {
		offset = int(val.(int64))
	}

	// send session to user
	hi, err = hso.Encode([]int{offset, sessioncode})
	if err != nil {
		log.Print("error encoding hashid for session offset ", offset, ": ", err)
		sendBlank(c)
		return
	}
	c.SetStatusCode(200)
	c.ResponseWriter.WriteString(hi)

	log.Print("tracked ", code, " ", referrer, " ", offset, " ", page)
}

func sendBlank(c *iris.Context) {
	c.SetStatusCode(204)
	c.ResponseWriter.WriteString("")
}
