package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/fjl/go-couchdb"
	"github.com/galeone/igor"
	"github.com/kelseyhightower/envconfig"
	"github.com/speps/go-hashids"
	"gopkg.in/redis.v5"
)

type Settings struct {
	Host                    string `envconfig:"HOST" required:"true"`
	Port                    string `envconfig:"PORT" required:"true"`
	CouchURL                string `envconfig:"COUCH_URL" required:"true"`
	CouchDatabaseName       string `envconfig:"COUCH_DATABASE" required:"true"`
	RedisAddr               string `envconfig:"REDIS_ADDR" required:"true"`
	RedisPassword           string `envconfig:"REDIS_PASSWORD" required:"true"`
	PostgresURL             string `envconfig:"DATABASE_URL" required:"true"`
	SessionOffsetHashidSalt string `envconfig:"SESSION_OFFSET_HASHID_SALT"`
	Auth0Secret             string `envconfig:"AUTH0_SECRET"`
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
	filename := "./track.lua"
	// try the current directory
	btracklua, err := ioutil.ReadFile(filename)
	if err != nil {
		// try some magic (based on the path of the source main.go)
		_, this, _, _ := runtime.Caller(0)
		here := path.Dir(this)
		btracklua, err = ioutil.ReadFile(filepath.Join(here, filename))
		if err != nil {
			log.Fatal("failed to read track.lua: ", err)
		}
	}
	tracklua = string(btracklua)

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
