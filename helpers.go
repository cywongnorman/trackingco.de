package main

import "time"

const DATEFORMAT = "20060102"
const (
	SESSIONS  = "s"
	PAGEVIEWS = "v"
	REFERRERS = "r"
	PAGES     = "p"
)

func presentDay() time.Time {
	now := time.Now().UTC()
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location())
}

func makeBaseKey(code, day string) string {
	return code + ":" + day
}

func redisKeyFactory(code, day string) func(string) string {
	basekey := makeBaseKey(code, day)
	return func(subkey string) string {
		return basekey + ":" + subkey
	}
}
