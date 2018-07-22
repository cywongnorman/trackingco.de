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
		hashid := dotspl[0][1:]
		c.SetUserValue("sessionhashid", hashid)
		track(c)
		return
	}

	spl := strings.Split(path, "/")
	if len(spl) == 3 && (spl[1] == "sites" || spl[1] == "public") {
		serveClient(c)
		return
	}

	switch path {
	case "/":
		c.SendFile("client/landing.html")
	case "/_graphql":
		user := extractUserFromJWT(c)
		if user == "" {
			user = s.LoggedAs
			log.Print("forced auth as ", user)
		}
		context := context.WithValue(
			context.TODO(),
			"loggeduser", user,
		)

		var gqr GraphQLRequest
		if err = json.Unmarshal(c.Request.Body(), &gqr); err != nil {
			c.Error("failed to read graphql request: "+err.Error(), 400)
			return
		}
		result := query(gqr, context)
		if jsonresult, err := json.Marshal(result); err != nil {
			c.Error("failed to marshal graphql response: "+err.Error(), 500)
			return
		} else {
			c.SetContentType("application/json")
			c.SetBody(jsonresult)
		}
		context.Done()
	case "/_/webhooks/strike":
		handleStrikeWebhook(c)
	case "/favicon.ico":
		c.SendFile("client/logo.png")
	case "/sites", "/account":
		serveClient(c)
	default:
		fasthttp.FSHandler(".", 0)(c)
	}
}

func serveClient(c *fasthttp.RequestCtx) {
	c.SendFile("client/index.html")
}

func extractUserFromJWT(ctx *fasthttp.RequestCtx) string {
	token := string(ctx.Request.Header.Peek("Authorization"))
	user, err := acd.VerifyAuth(token)
	if err != nil {
		ctx.Error("wrong authorization token: "+token, 401)
		return ""
	}
	return user
}
