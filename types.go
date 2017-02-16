package main

import "strings"

type Compendium struct {
	Day       string         `json:"day"`
	Sessions  int            `json:"sessions"`
	Pageviews int            `json:"pageviews"`
	Referrers map[string]int `json:"referrers"`
	Pages     map[string]int `json:"pages"`
}

type CompendiumDoc struct {
	Id        string         `json:"_id"`
	Rev       string         `json:"_rev,omitempty"`
	Sessions  int            `json:"s"`
	Pageviews int            `json:"v"`
	Referrers map[string]int `json:"r"`
	Pages     map[string]int `json:"p"`
}

type Entry struct {
	Address string `json:"addr"`
	Count   int    `json:"count"`
}

type EntrySort []Entry

func (a EntrySort) Len() int           { return len(a) }
func (a EntrySort) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a EntrySort) Less(i, j int) bool { return a[i].Count < a[j].Count }

type User struct {
	Id       int          `json:"id,omitempty" igor:"primary_key"`
	Name     string       `json:"name,omitempty"`
	Email    string       `json:"email,omitempty"`
	Password string       `json:"password,omitempty" sql:"-"`
	Sites    []Site       `json:"sites,omitempty" sql:"-"`
	Settings UserSettings `json:"settings,omitempty" sql:"-"`
}

type Site struct {
	Code      string       `json:"code,omitempty" igor:"primary_key"`
	Name      string       `json:"name,omitempty"`
	UserId    int          `json:"user_id,omitempty"`
	CreatedAt string       `json:"created_at,omitempty"`
	Days      []Compendium `json:"days,omitempty" sql:"-"`
	Months    []Compendium `json:"months,omitempty" sql:"-"`
}

type UserSettings struct {
	SitesOrder []string `json:"sites_order,omitempty" igor:"primary_key"`
}

func (_ UserSettings) TableName() string { return "settings" }
func (_ User) TableName() string         { return "users" }
func (_ Site) TableName() string         { return "sites" }

type CouchDBResults struct {
	Rows []struct {
		Rev string        `json:"rev"`
		Id  string        `json:"id"`
		Doc CompendiumDoc `json:"doc"`
	} `json:"rows"`
}

func (res CouchDBResults) toCompendiumList() []Compendium {
	var c = make([]Compendium, len(res.Rows))
	for i, row := range res.Rows {
		c[i] = Compendium{
			Day:       strings.Split(row.Id, ":")[1],
			Sessions:  row.Doc.Sessions,
			Pageviews: row.Doc.Pageviews,
			Referrers: row.Doc.Referrers,
			Pages:     row.Doc.Pages,
		}
	}
	return c
}

type Result struct {
	Ok bool `json:"ok"`
}
