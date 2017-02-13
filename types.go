package main

type Track struct {
	TrackingCode string `json:"t,omitempty"`
	Session      string `json:"s"`
	Page         string `json:"p"`
	Referrer     string `json:"r"`
}

type Compendium struct {
	Sessions  int            `json:"sessions,omitempty"`
	Pageviews int            `json:"pageviews,omitempty"`
	Referrers map[string]int `json:"referrers,omitempty"`
	Pages     map[string]int `json:"pages,omitempty"`
}

type Entry struct {
	Address string `json:"addr,omitempty"`
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
		Rev string `json:"rev"`
		Id  string `json:"id"`
		Doc struct {
			Id        string         `json:"_id"`
			Rev       string         `json:"_rev"`
			Sessions  int            `json:"s"`
			Pageviews int            `json:"v"`
			Referrers map[string]int `json:"r"`
			Pages     map[string]int `json:"p"`
		}
	} `json:"rows"`
}

func (res CouchDBResults) toCompendiumList() []Compendium {
	var c = make([]Compendium, len(res.Rows))
	for i, row := range res.Rows {
		c[i] = Compendium{
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
