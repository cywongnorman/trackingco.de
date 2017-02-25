package main

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/speps/go-hashids"
)

const DATEFORMAT = "20060102"

func presentDay() time.Time {
	now := time.Now().UTC()
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location())
}

func makeBaseKey(code, day string) string { return code + ":" + day }
func redisKeyFactory(code, day string) func(string) string {
	basekey := makeBaseKey(code, day)
	return func(subkey string) string {
		return basekey + ":" + subkey
	}
}

func makeCodeForUser(userId int) string {
	hd := hashids.NewData()
	hd.MinLength = 5
	hd.Alphabet = "abcdefghijklmnopqrstuvwxyz1234567890"
	h := hashids.NewWithData(hd)
	r, _ := h.Encode([]int{userId, randomNumber(9999)})
	return r
}

func randomNumber(r int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(r)
}

func compendiumFromRedis(code, day string) Compendium {
	key := redisKeyFactory(code, day)

	compendium := Compendium{
		Id:       makeBaseKey(code, day),
		Sessions: make(map[string]string),
		Pages:    make(map[string]int),
	}

	if val, err := rds.HGetAll(key("s")).Result(); err == nil {
		for k, v := range val {
			compendium.Sessions[k] = v
		}
	}
	if val, err := rds.HGetAll(key("p")).Result(); err == nil {
		for k, v := range val {
			if count, err := strconv.Atoi(v); err == nil {
				compendium.Pages[k] = count
			}
		}
	}

	return compendium
}
