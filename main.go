package main

import (
	"context"
	"log"
	"time"

	"github.com/fjl/go-couchdb"
	"github.com/galeone/igor"
	"github.com/iris-contrib/graceful"
	"github.com/kataras/iris"
	"github.com/kelseyhightower/envconfig"
	"gopkg.in/redis.v5"
)

type Settings struct {
	Port              string `envconfig:"PORT"`
	CouchURL          string `envconfig:"COUCH_URL"`
	CouchDatabaseName string `envconfig:"COUCH_DATABASE"`
	RedisAddr         string `envconfig:"REDIS_ADDR"`
	RedisPassword     string `envconfig:"REDIS_PASSWORD"`
	PostgresURL       string `envconfig:"DATABASE_URL"`
	LoggedAs          int    `envconfig:"LOGGED_AS"`
}

var err error
var pg *igor.Database
var rds *redis.Client
var couch *couchdb.DB

func main() {
	var s Settings
	err = envconfig.Process("", &s)
	if err != nil {
		log.Fatal("couldn't process envconfig: ", err)
	}

	// redis
	rds = redis.NewClient(&redis.Options{
		Addr:     s.RedisAddr,
		Password: s.RedisPassword,
	})

	// postgres
	pg, err = igor.Connect(s.PostgresURL)
	if err != nil {
		log.Fatal("couldn't connect to postgres at "+s.PostgresURL+": ", err)
	}

	// couchdb
	couchS, err := couchdb.NewClient(s.CouchURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	couch = couchS.DB(s.CouchDatabaseName)

	api := iris.New()
	api.Post("/_graphql", func(c *iris.Context) {
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

	api.Get("/t.gif", func(c *iris.Context) {
		code := c.FormValue("t")
		page := c.FormValue("p")
		referrer := c.FormValue("r")

		if code == "" || page == "" {
			c.SetStatusCode(400)
			return
		}

		// store data to redis
		twodays := time.Hour * 48
		basekey := time.Now().Format("20060102") + ":" + code

		if c.GetCookie("_tcs") == "" {
			// new session
			c.SetCookieKV("_tcs", "1")

			rds.Incr(basekey + ":s")
			rds.Expire(basekey+":s", twodays)
		}

		rds.Incr(basekey + ":pv")
		rds.Expire(basekey+":pv", twodays)

		rds.HIncrBy(basekey+":p", page, 1)
		rds.Expire(basekey+":p", twodays)

		rds.HIncrBy(basekey+":r", referrer, 1)
		rds.Expire(basekey+":r", twodays)

		log.Print("tracked " + code)
		c.SetStatusCode(200)
	})

	graceful.Run(":"+s.Port, time.Duration(10)*time.Second, api)
}
