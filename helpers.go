package main

const DATEFORMAT = "20060102"

func makeBaseKey(code, day string) string {
	return day + ":" + code
}

func redisKeyFactory(code, day string) func(string) string {
	basekey := makeBaseKey(code, day)
	return func(subkey string) string {
		return basekey + ":" + subkey
	}
}
