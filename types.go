package main

type Track struct {
	TrackingCode string `json:"t,omitempty"`
	Session      string `json:"s"`
	Page         string `json:"p"`
	Referrer     string `json:"r"`
}

type Compendium struct {
	Sessions  int     `json:"sessions,omitempty"`
	Pageviews int     `json:"pageviews,omitempty"`
	Referrers []Entry `json:"referrers,omitempty"`
	Pages     []Entry `json:"pages,omitempty"`
}

type Entry struct {
	Address string `json:"address,omitempty"`
	Count   int    `json:"count,omitempty"`
}

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
