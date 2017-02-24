package main

import (
	"bytes"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"gopkg.in/kataras/iris.v6"
)

func track(c *iris.Context) {
	defer sendImage(c)

	// tracking code
	code := c.FormValue("c")
	if code == "" {
		log.Print("didn't send tracking code.")
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
	if hi := c.GetCookie("_tcs"); hi != "" {
		// existing session
		if offsetarr, err := hso.DecodeWithError(hi); err != nil || len(offsetarr) != 2 {
			log.Print("error decoding session hashid ", hi, ": ", err, " (treating as new session)")
			offset = -1
		} else {
			// _valid_ existing session
			offset = offsetarr[0]
			// this session code will be used to fetch the referrer for this session
			sessioncode = offsetarr[1]
			referrer = "@"
		}
	} else {
		// new session
		offset = -1
		// this session code will be used to store the referrer for this session
		sessioncode = randomNumber(999999999)
	}

	// store data to redis
	twodays := int(time.Hour * 48)
	key := redisKeyFactory(code, time.Now().Format(DATEFORMAT))

	result := rds.Eval(
		tracklua,
		[]string{
			key("p"),    // KEYS[1]
			key("s"),    // KEYS[2]
			key("rfsc"), // KEYS[3]
		},
		page,        // ARGV[1]
		referrer,    // ARGV[2]
		offset,      // ARGV[3]
		twodays,     // ARGV[4]
		sessioncode, // ARGV[5]
		points,      // ARGV[6]
	)

	if val, err := result.Result(); err != nil {
		log.Print("error executing track.lua: ", err)
		return
	} else {
		offset = int(val.(int64))
	}

	// send session to user
	hi, err := hso.Encode([]int{offset, sessioncode})
	if err != nil {
		log.Print("error encoding hashid for session offset ", offset, ": ", err)
		return
	}
	c.SetCookieKV("_tcs", hi)

	log.Print("tracked ", code, " ", referrer, " ", offset, " ", page)
}

func sendImage(c *iris.Context) {
	// no cache
	c.SetHeader("Cache-Control", "no-cache, no-store, must-revalidate")
	c.SetHeader("Pragma", "no-cache")
	c.SetHeader("Expires", "0")

	var buffer *bytes.Buffer

	switch c.Param("extension") {
	case "gif":
		buffer = gifimage
		c.SetContentType("image/gif")
		break
	case "jpeg", "jpg":
		buffer = jpgimage
		c.SetContentType("image/jpeg")
		break
	case "png":
		buffer = pngimage
		c.SetContentType("image/png")
		break
	}

	c.SetHeader("Content-Length", strconv.Itoa(len(buffer.Bytes())))

	if _, err = c.ResponseWriter.Write(buffer.Bytes()); err != nil {
		log.Print("failed to serve image: ", err)
		c.HTML(500, "<p>Erro!</p>")
	}
}
