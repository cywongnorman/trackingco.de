package main

const DATEFORMAT = "20060102"

func makeBaseKey(code string, day string) string {
	return day + ":" + code
}

func redisKeyFactory(code string, day string) func(string) string {
	basekey := makeBaseKey(code, day)
	return func(subkey string) string {
		return basekey + ":" + subkey
	}
}
