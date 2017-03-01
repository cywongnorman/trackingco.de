package main

import (
	"context"
	"encoding/json"
	"log"

	"github.com/qiangxue/fasthttp-routing"
	"github.com/valyala/fasthttp"
)

func runServer() {
	router := routing.New()
	router.Get("/", func(c *routing.Context) error {
		c.SendFile("landing.html")
		return nil
	})
	router.Get("/sites", serveClient)
	router.Get("/sites/<code>", serveClient)
	router.Get("/public/<code>", serveClient)

	router.Post("/_graphql", func(c *routing.Context) error {
		var gqr GraphQLRequest
		if err = json.Unmarshal(c.Request.Body(), &gqr); err != nil {
			return HTTPError{400, "failed to read graphql request: " + err.Error()}
		}
		context := context.WithValue(context.TODO(), "loggeduser", s.LoggedAs)
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

	router.Get("/<sessionhashid:[0-9a-zA-Z-~^.]+>.xml", track)

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
