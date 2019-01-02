package main

import (
	"encoding/json"

	"github.com/jmoiron/sqlx/types"
)

type Session struct {
	Referrer string        `json:"referrer"`
	Events   []interface{} `json:"events"`
}

type Day struct {
	Day string `json:"day,omitempty" db:"day"`

	// [{ referrer: 'https://xyz.com/'
	//  , events: ['/page', 5, '/otherpage', 7]
	//  }
	// , ...
	// ]
	RawSessions types.JSONText `db:"sessions" json:"-"`

	sessions []Session
}

func (day Day) stats() (stats Stats) {
	for _, s := range day.sessions {
		stats.NSessions++

		for _, event := range s.Events {
			switch v := event.(type) {
			case int:
				stats.Score += v
			case string:
				stats.NPageviews++
				stats.Score += 1
			}
		}

		if len(s.Events) == 1 {
			_, isPage := s.Events[0].(string)
			points, isPoints := s.Events[0].(int)
			if isPage || (isPoints && points == 0) {
				stats.NBounces++
			}
		}
	}
	return
}

type Month struct {
	Month string `json:"month" db:"month"`

	Stats
	Compendium
}

type Stats struct {
	NSessions  int `json:"s" db:"nsessions"`  // total number of sessions
	NBounces   int `json:"b" db:"nbounces"`   // sessions with just one pageview
	NPageviews int `json:"v" db:"npageviews"` // total number of pageviews
	Score      int `json:"c" db:"score"`      // total score (sum of all session scores)
}

type Compendium struct {
	TopReferrers       map[string]int `json:"r"`
	TopPages           map[string]int `json:"p"`
	TopReferrersScores map[string]int `json:"z"`

	RawTopReferrers       types.JSONText `json:"-" db:"top_referrers"`
	RawTopPages           types.JSONText `json:"-" db:"top_pages"`
	RawTopReferrersScores types.JSONText `json:"-" db:"top_referrers_scores"`
}

func (c *Compendium) apply(session Session) {
	count := c.TopReferrers[session.Referrer]
	c.TopReferrers[session.Referrer] = count + 1

	scores := c.TopReferrersScores[session.Referrer]
	for _, event := range session.Events {
		switch v := event.(type) {
		case int:
			scores += v
		case string:
			scores += 1

			pv := c.TopPages[v]
			c.TopPages[v] = pv + 1
		}
	}
	c.TopReferrersScores[session.Referrer] = scores
}

func (c *Compendium) join(cc Compendium) {
	for k, v := range cc.TopPages {
		prev := c.TopPages[k]
		c.TopPages[k] = prev + v
	}
	for k, v := range cc.TopReferrers {
		prev := c.TopReferrers[k]
		c.TopReferrers[k] = prev + v
	}
	for k, v := range cc.TopReferrersScores {
		prev := c.TopReferrersScores[k]
		c.TopReferrersScores[k] = prev + v
	}
}

func (c *Compendium) unmarshal() {
	json.Unmarshal(c.RawTopPages, &c.TopPages)
	json.Unmarshal(c.RawTopReferrers, &c.TopReferrers)
	json.Unmarshal(c.RawTopReferrersScores, &c.TopReferrersScores)
}
