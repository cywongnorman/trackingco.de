package main

import "strings"

type Compendium struct {
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

type Entry struct {
	Address string `json:"a"`
	Count   int    `json:"c"`
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
	Id       int    `json:"id,omitempty" igor:"primary_key"`
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty" sql:"-"`

	SitesOrder []string `json:"-" sql:"-"`

	Sites []Site `json:"sites,omitempty" sql:"-"`
}

func (_ User) TableName() string { return "users" }

type Site struct {
	Code      string `json:"code,omitempty" igor:"primary_key"`
	Name      string `json:"name,omitempty"`
	UserId    int    `json:"user_id,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	Shared    bool   `json:"shared,omitempty"`

	lastDays  int
	couchDays []Compendium

	ShareURL string       `json:"shareURL,omitempty" sql:"-"`
	Days     []Compendium `json:"days,omitempty" sql:"-"`
	Months   []Compendium `json:"months,omitempty" sql:"-"`
	Today    Compendium   `json:"today,omitempty" sql:"-"`
}

func (_ Site) TableName() string { return "sites" }

type CouchDBDayResults struct {
	Rows []struct {
		Rev string     `json:"rev"`
		Id  string     `json:"id"`
		Doc Compendium `json:"doc"`
	} `json:"rows"`
}

func (res CouchDBDayResults) toCompendiumList() []Compendium {
	var c = make([]Compendium, len(res.Rows)+1) // +1 will be used in a later loop, and it does no harm
	for i, row := range res.Rows {
		c[i] = row.Doc
		c[i].Day = strings.Split(row.Id, ":")[1]
		c[i].Id = ""
		c[i].Rev = ""
	}
	return c
}

type Result struct {
	Ok bool `json:"ok"`
}
