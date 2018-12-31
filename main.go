package main

import (
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"gopkg.in/redis.v5"
)

type Settings struct {
	Host          string `envconfig:"HOST" required:"true"`
	Port          string `envconfig:"PORT" required:"true"`
	RedisAddr     string `envconfig:"REDIS_ADDR" required:"true"`
	RedisPassword string `envconfig:"REDIS_PASSWORD" required:"true"`
	PostgresURL   string `envconfig:"DATABASE_URL" required:"true"`
}

var err error
var s Settings
var pg *sqlx.DB
var rds *redis.Client
var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})
var blacklist map[string]bool

func main() {
	err = envconfig.Process("", &s)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't process envconfig")
	}

	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log = log.With().Timestamp().Logger()

	// redis
	rds = redis.NewClient(&redis.Options{
		Addr:     s.RedisAddr,
		Password: s.RedisPassword,
	})

	// postgres connection
	pg, err = sqlx.Connect("postgres", s.PostgresURL)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't connect to postgres")
	}

	// referrer blacklist
	blacklist = buildReferrerBlacklist()
	log.Print("using referrer blacklist with ", len(blacklist), " entries.")

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
