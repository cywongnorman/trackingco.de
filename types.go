package main

import "github.com/jmoiron/sqlx/types"

type Session struct {
	Referrer string        `json:"referrer"`
	Events   []interface{} `json:"events"`
}

type Day struct {
	Day string `json:"day,omitempty" pg:"day"`

	// [{ referrer: 'https://xyz.com/'
	//  , events: ['/page', 5, '/otherpage', 7]
	//  }
	// , ...
	// ]
	RawSessions types.JSONText `pg:"sessions" json:"-"`

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
			if _, isPage := s.Events[0].(string); isPage {
				stats.NBounces++
			}
		}
	}
	return
}

type Month struct {
	Month string `json:"month" pg:"month"`

	Stats
	Compendium
}

type Stats struct {
	NSessions  int `json:"s" pg:"nsessions"`  // total number of sessions
	NBounces   int `json:"b" pg:"nbounces"`   // sessions with just one pageview
	NPageviews int `json:"v" pg:"npageviews"` // total number of pageviews
	Score      int `json:"c" pg:"score"`      // total score (sum of all session scores)
}

type Compendium struct {
	TopReferrers       map[string]int `json:"r" pg:"top_referrers"`
	TopPages           map[string]int `json:"p" pg:"top_pages"`
	TopReferrersScores map[string]int `json:"z" pg:"top_referrers_scores"`
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
