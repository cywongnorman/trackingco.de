package main

import "github.com/jmoiron/sqlx/types"

type Session struct {
	Referrer string        `json:"referrer"`
	Events   []interface{} `json:"events"`
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

type Day struct {
	Day string `json:"day,omitempty" pg:"day"`

	// [{ referrer: 'https://xyz.com/'
	//  , events: ['/page', 5, '/otherpage', 7]
	//  }
	// , ...
	// ]
	RawSessions types.JSONText `pg:"sessions" json:"-"`
}

type Month struct {
	Month string `json:"month,omitempty" pg:"month"`

	// the average bounce rate for this month, in units of 10000
	// (for example, if the bounce rate is 43,78% it will be stored as 4378)
	BounceRate int `json:"b" pg:"bounce_rate"`
	NSessions  int `json:"s" pg:"nsessions"`  // total number of sessions
	NPageviews int `json:"v" pg:"npageviews"` // total number of pageviews
	Score      int `json:"c" pg:"score"`      // total score (sum of all session scores)

	// the top 10 referrers for this month, with their respective counts
	TopReferrers map[string]int `json:"r"`

	// the top 10 pages viewed this month, with their respective counts
	TopPages map[string]int `json:"p"`
}
