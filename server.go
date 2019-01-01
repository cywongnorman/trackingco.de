package main

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/valyala/fasthttp"
)

func runServer() {
	log.Print("listening at :" + s.Port)
	panic(fasthttp.ListenAndServe(":"+s.Port, fastHTTPHandler))
}

func fastHTTPHandler(c *fasthttp.RequestCtx) {
	path := string(c.Path())

	dotspl := strings.Split(path, ".")
	if len(dotspl) == 2 && dotspl[1] == "xml" {
		cuid := dotspl[0][1:]
		track(c, cuid)
		return
	}

	switch path {
	case "/":
		c.SendFile("static/landing.html")
	case "/favicon.ico":
		c.SendFile("static/logo.png")
	default:
		if strings.HasPrefix(path, "/query/") {
			handleQuery(path, c)
			return
		}

		if strings.HasPrefix(path, "/static/") {
			fasthttp.FSHandler(".", 0)(c)
			return
		}

		serveClient(c)
	}
}

func serveClient(c *fasthttp.RequestCtx) {
	c.SendFile("static/index.html")
}

func handleQuery(path string, c *fasthttp.RequestCtx) {
	ctx := context.TODO()

	var params Params
	if err = json.Unmarshal(c.Request.Body(), &params); err != nil {
		c.Error("failed to read request: "+err.Error(), 400)
		return
	}

	var result interface{}

	switch path {
	case "/query/days":
		result, err = queryDays(params)
		break
	case "/query/months":
		result, err = queryMonths(params)
		break
	case "/query/today":
		result, err = queryToday(params)
		break
	}

	if err != nil {
		c.Error("query failure: "+err.Error(), 500)
		return
	}

	if jsonresult, err := json.Marshal(result); err != nil {
		c.Error("failed to marshal graphql response: "+err.Error(), 500)
		return
	} else {
		c.SetContentType("application/json")
		c.SetBody(jsonresult)
	}

	ctx.Done()
}
