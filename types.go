package main

import (
	"strings"

	"github.com/jmoiron/sqlx/types"
)

type Day struct {
	Id  string `json:"_id,omitempty"`
	Rev string `json:"_rev,omitempty"`
	Day string `json:"day,omitempty"`

	// a map of referrers to strings like "~1201020302050422"
	// representing the score for each visitor: [12, 1, 2, 3, 2, 5, 4, 22]
	// each pageview is 1 point. users can track points using the JS API.
	// the raw number of sessions is given by the sum of len(split(eachvalue, ',')).
	Sessions map[string]string `json:"s"`

	// a map of all pages viewed with the number of views in each one.
	// the raw number of pageviews is given by the sum of eachvalue.
	Pages map[string]int `json:"p"`
}

type Month struct {
	Id    string `json:"_id,omitempty"`
	Rev   string `json:"_rev,omitempty"`
	Month string `json:"month,omitempty"`

	// the average bounce rate for this month, in units of 10000
	// (for example, if the bounce rate is 43,78% it will be stored as 4378)
	BounceRate int `json:"b"`
	Sessions   int `json:"s"` // total number of sessions in this month
	Pageviews  int `json:"v"` // total number of pageviews in this month
	Score      int `json:"c"` // the total score (sum of all session scores)

	// the top 10 referrers for this month, with their respective counts
	TopReferrers map[string]int `json:"r"`

	// the top 10 pages viewed this month, with their respective counts
	TopPages map[string]int `json:"p"`
}

type Entry struct {
	Address string `json:"a"`
	Count   int    `json:"c"`
}

func EntriesFromMap(dict map[string]int) []Entry {
	entries := make([]Entry, len(dict))
	i := 0
	for ref, count := range dict {
		entries[i] = Entry{ref, count}
		i++
	}
	return entries
}
func MapFromEntries(entries []Entry) map[string]int {
	dict := make(map[string]int, len(entries))
	for _, entry := range entries {
		dict[entry.Address] = entry.Count
	}
	return dict
}

type EntrySort []Entry

func (a EntrySort) Len() int           { return len(a) }
func (a EntrySort) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a EntrySort) Less(i, j int) bool { return a[i].Count < a[j].Count }

type SessionGroup struct {
	Referrer string `json:"referrer"`
	Scores   []int  `json:"scores"`
}

type User struct {
	Id      string         `json:"id" db:"id"`
	Domains string         `json:"domains" db:"domains"` // array, comma-separated.
	Colours types.JSONText `json:"colours" db:"colours"`
	NMonths int            `json:"nmonths" db:"nmonths"`
}

type Payment struct {
	Id        string `json:"id" db:"id"`
	UserId    string `json:"user_id" db:"user_id"`
	Amount    int    `json:"amount" db:"amount"`
	CreatedAt string `json:"created_at" db:"created_at"`
	HasPaid   bool   `json:"has_paid" db:"has_paid"`
	PaidAt    string `json:"paid_at" db:"paid_at"`
}

type Site struct {
	Code      string `json:"code,omitempty" db:"code"`
	Name      string `json:"name,omitempty" db:"name"`
	Owner     string `json:"owner,omitempty" db:"owner"`
	CreatedAt string `json:"created_at,omitempty" db:"created_at"`
	Shared    bool   `json:"shared,omitempty" db:"shared"`

	lastDays    int
	couchDays   []Day
	couchMonths []Month

	ShareURL string  `json:"shareURL,omitempty"`
	Days     []Day   `json:"days,omitempty"`
	Months   []Month `json:"months,omitempty"`
	Today    Day     `json:"today,omitempty"`
}

type CouchDBDayResults struct {
	Rows []struct {
		Rev string `json:"rev"`
		Id  string `json:"id"`
		Doc Day    `json:"doc"`
	} `json:"rows"`
}

func (res CouchDBDayResults) toDayList() []Day {
	var c = make([]Day, len(res.Rows)+1) // +1 will be used in a later loop, and it does no harm
	for i, row := range res.Rows {
		c[i] = row.Doc
		c[i].Day = strings.Split(row.Id, ":")[1]
		c[i].Id = ""
		c[i].Rev = ""
	}
	return c
}

type CouchDBMonthResults struct {
	Rows []struct {
		Rev string `json:"rev"`
		Id  string `json:"id"`
		Doc Month  `json:"doc"`
	} `json:"rows"`
}

func (res CouchDBMonthResults) toMonthList() []Month {
	var c = make([]Month, len(res.Rows))
	for i, row := range res.Rows {
		c[i] = row.Doc
		c[i].Month = strings.Split(row.Id, ".")[1]
		c[i].Id = ""
		c[i].Rev = ""
	}
	return c
}

type Result struct {
	Ok    bool   `json:"ok"`
	Error string `json:"error"`
}
