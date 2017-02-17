package main

import (
	"context"
	"log"

	"github.com/kataras/iris/adaptors/gorillamux"
	"gopkg.in/kataras/iris.v6"
)

func runServer() {
	app := iris.New()
	app.Adapt(gorillamux.New())

	app.Post("/_graphql", func(c *iris.Context) {
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

	app.Get("/", func(c *iris.Context) { c.ServeFile("client/index.html", false) })
	app.Get("/client/{file:.*}", func(c *iris.Context) { c.ServeFile("client/"+c.Param("file"), false) })

	app.Get("/{x:[0-9a-zA-Z]+.(?:gif|jpg|jpeg|png)}", track)

	log.Print("listening at :" + s.Port)
	app.Listen(":" + s.Port)
}
