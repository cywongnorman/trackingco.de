package main

import "encoding/json"

type Params struct {
	Domain         string `json:"domain"`
	Last           int    `json:"last"`
	Limit          int    `json:"limit"`
	MinScore       int    `json:"min_score"`
	ReferrerFilter string `json:"referrer_filter"`
}

func queryDays(params Params) (res interface{}, err error) {
	var days []Day
	err = pg.Select(&days, `
SELECT day, sessions FROM days
WHERE domain = $1
ORDER BY day DESC
LIMIT $2
    `, params.Domain, params.Last)
	if err != nil {
		return
	}

	stats := make([]Stats, len(days))
	compendium := &Compendium{
		TopPages:           make(map[string]int),
		TopReferrers:       make(map[string]int),
		TopReferrersScores: make(map[string]int),
	}
	daynames := make([]string, len(days))

	for i := range days {
		err = json.Unmarshal(days[i].RawSessions, &days[i].sessions)
		if err != nil {
			return
		}

		daynames[i] = days[i].Day
		stats[i] = days[i].stats()

		for _, session := range days[i].sessions {
			compendium.apply(session)
		}
	}

	return struct {
		Days       []string   `json:"days"`
		Stats      []Stats    `json:"stats"`
		Compendium Compendium `json:"compendium"`
	}{daynames, stats, *compendium}, nil
}

func queryMonths(params Params) (res interface{}, err error) {
	var months []Month
	err = pg.Select(&months, `
SELECT month,
  nbounces, nsessions, npageviews, score,
  top_pages,
  top_referrers --, top_referrers_scores
FROM months
WHERE domain = $1
ORDER BY month DESC
LIMIT $2
    `, params.Domain, params.Last)
	if err != nil {
		return
	}

	for i := range months {
		months[i].unmarshal()
	}

	return months, nil
}

func queryToday(params Params) (res interface{}, err error) {
	today := presentDay().Format(DATEFORMAT)
	day := dayFromRedis(params.Domain, today)
	return day.stats(), nil
}
