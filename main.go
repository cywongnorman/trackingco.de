package main

import (
	"log"
	"os"

	"github.com/fjl/go-couchdb"
	"github.com/galeone/igor"
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
var s Settings
var pg *igor.Database
var rds *redis.Client
var couch *couchdb.DB

func main() {
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

	// run routines or start the server
	if len(os.Args) == 1 {
		runServer()
	} else {
		switch os.Args[1] {
		case "daily":
			daily()
		case "monthly":
			monthly()
		default:
			log.Print("couldn't find what to run for ", os.Args[1])
		}
	}
}
