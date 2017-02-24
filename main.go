package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/fjl/go-couchdb"
	"github.com/galeone/igor"
	"github.com/kelseyhightower/envconfig"
	"github.com/speps/go-hashids"
	"gopkg.in/redis.v5"
)

type Settings struct {
	Port                    string `envconfig:"PORT"`
	CouchURL                string `envconfig:"COUCH_URL"`
	CouchDatabaseName       string `envconfig:"COUCH_DATABASE"`
	RedisAddr               string `envconfig:"REDIS_ADDR"`
	RedisPassword           string `envconfig:"REDIS_PASSWORD"`
	PostgresURL             string `envconfig:"DATABASE_URL"`
	SessionOffsetHashidSalt string `envconfig:"SESSION_OFFSET_HASHID_SALT"`
	LoggedAs                int    `envconfig:"LOGGED_AS"`
}

var err error
var s Settings
var pg *igor.Database
var hso *hashids.HashID
var rds *redis.Client
var couch *couchdb.DB
var tracklua string

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
		log.Fatal("failed to created couchdb client: ", err)
	}
	couch = couchS.DB(s.CouchDatabaseName)

	// hashids for session offset
	hd := hashids.NewData()
	hd.Salt = s.SessionOffsetHashidSalt
	hso = hashids.NewWithData(hd)

	// track.lua
	if btracklua, err := ioutil.ReadFile("./track.lua"); err == nil {
		tracklua = string(btracklua)
	} else {
		log.Fatal("failed to read track.lua: ", err)
	}

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
