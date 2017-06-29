package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/dgrijalva/jwt-go"
	"github.com/qiangxue/fasthttp-routing"
	"github.com/valyala/fasthttp"
)

func runServer() {
	router := routing.New()
	router.Get("/", func(c *routing.Context) error {
		c.SendFile("client/landing.html")
		return nil
	})

	router.Get("/favicon.ico", func(c *routing.Context) error {
		c.SendFile("client/logo.png")
		return nil
	})
	router.Get("/billing/bitcoinpay/done/", BitcoinPayDone)
	router.Post("/billing/bitcoinpay/ipn/", BitcoinPayIPN)
	router.Get("/sites", serveClient)
	router.Get("/account", serveClient)
	router.Get("/sites/<code>", serveClient)
	router.Get("/public/<code>", serveClient)

	router.Post("/_graphql", func(c *routing.Context) error {
		email := extractEmailFromJWT(c.RequestCtx)
		if email == "" {
			email = s.LoggedAs
			log.Print("forced auth as ", email)
		}
		context := context.WithValue(
			context.TODO(),
			"loggeduser", email,
		)

		var gqr GraphQLRequest
		if err = json.Unmarshal(c.Request.Body(), &gqr); err != nil {
			return HTTPError{400, "failed to read graphql request: " + err.Error()}
		}
		result := query(gqr, context)
		if jsonresult, err := json.Marshal(result); err != nil {
			return HTTPError{500, "failed to marshal graphql response: " + err.Error()}
		} else {
			c.SetContentType("application/json")
			c.SetBody(jsonresult)
		}
		context.Done()
		return nil
	})

	fsHandler := fasthttp.FSHandler(".", 0)
	router.Get("/client/*", func(c *routing.Context) error {
		fsHandler(c.RequestCtx)
		return nil
	})

	router.Get("/<sessionhashid:[0-9a-zA-Z-~^]+>.xml", track)

	log.Print("listening at :" + s.Port)
	panic(fasthttp.ListenAndServe(":"+s.Port, router.HandleRequest))
}

func serveClient(c *routing.Context) error {
	c.SendFile("client/index.html")
	return nil
}

type HTTPError struct {
	status  int
	message string
}

func (h HTTPError) StatusCode() int { return h.status }
func (h HTTPError) Error() string {
	log.Print(h.message)
	return h.message
}

func extractEmailFromJWT(ctx *fasthttp.RequestCtx) string {
	jwtbytes := ctx.Request.Header.Peek("Authorization")
	if len(jwtbytes) == 0 {
		jwtbytes = ctx.Request.Header.Peek("authorization")
		if len(jwtbytes) == 0 {
			log.Print("no jwt.")
			return ""
		}
	}

	token, err := jwt.Parse(string(jwtbytes), func(token *jwt.Token) (interface{}, error) {
		return []byte(s.Auth0Secret), nil
	})
	if err != nil {
		log.Print("failed to parse the jwt '", string(jwtbytes), "': ", err)
		return ""
	}

	if token.Method != jwt.SigningMethodHS256 {
		log.Print("expected jwt signed with HS256, but got ", token.Method)
		return ""
	}

	if !token.Valid {
		log.Print("parsed jwt is invalid.")
		return ""
	}

	email, _ := token.Claims["https://trackingco.de/user/email"].(string)
	return email
}
