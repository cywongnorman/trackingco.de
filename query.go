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
		make(map[string]int),
		make(map[string]int),
		make(map[string]int),
	}

	for i, day := range days {
		err = json.Unmarshal(day.RawSessions, &days[i].sessions)
		if err != nil {
			return
		}

		stats[i] = day.stats()

		for _, session := range day.sessions {
			compendium.apply(session)
		}
	}

	return struct {
		Days       []Day      `json:"days"`
		Stats      []Stats    `json:"stats"`
		Compendium Compendium `json:"compendium"`
	}{days, stats, *compendium}, nil
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

	return months, nil
}

func queryToday(params Params) (res interface{}, err error) {
	today := presentDay().Format(DATEFORMAT)
	day := dayFromRedis(params.Domain, today)
	return day.stats(), nil
}

/*
var dayType = graphql.NewObject(
	graphql.ObjectConfig{
		Name:        "Day",
		Description: "Compiled stats for a single day.",
		Fields: graphql.Fields{
			"day": &graphql.Field{
				Type:        graphql.String,
				Description: "the date in format YYYYMMDD.",
			},
			"v": &graphql.Field{
				Type:        graphql.Int,
				Description: "total number of pageviews.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					totalpages := 0
					for _, count := range p.Source.(Day).Pages {
						totalpages += count
					}
					return totalpages, nil
				},
			},
			"s": &graphql.Field{
				Type:        graphql.Int,
				Description: "total number of sessions.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					totalsessions := 0
					for _, scoremap := range p.Source.(Day).Sessions {
						totalsessions += (len(scoremap) - 1) / 2
					}
					return totalsessions, nil
				},
			},
			"b": &graphql.Field{
				Type:        graphql.Float,
				Description: "the bounce rate for this period.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					totalsessions := 0
					totalbounces := 0
					for _, scoremap := range p.Source.(Day).Sessions {
						sessions := sessionsFromScoremap(scoremap)
						for _, score := range sessions {
							totalsessions += 1
							if score == 1 {
								totalbounces += 1
							}
						}
					}
					return float64(totalbounces) / float64(totalsessions), nil
				},
			},
		},
	},
)

var monthType = graphql.NewObject(
	graphql.ObjectConfig{
		Name:        "Month",
		Description: "Compiled stats for a single month.",
		Fields: graphql.Fields{
			"month": &graphql.Field{
				Type:        graphql.String,
				Description: "the date in format YYYYMM.",
			},
			"v": &graphql.Field{
				Type:        graphql.Int,
				Description: "total number of pageviews.",
			},
			"s": &graphql.Field{
				Type:        graphql.Int,
				Description: "total number of sessions.",
			},
			"b": &graphql.Field{
				Type:        graphql.Float,
				Description: "the bounce rate for this period.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					// it is saved as an int multiplied by 100.
					return float64(p.Source.(Month).BounceRate) / 100, nil
				},
			},
			"c": &graphql.Field{
				Type:        graphql.Int,
				Description: "total score: the sum of all scores of all sessions.",
			},
		},
	},
)

					if site.lastDays > 90 {
						err = fetchMonths(&site)
						if err != nil {
							return nil, err
						}
						for _, month := range site.couchMonths {
							for ref, count := range month.TopReferrers {
								if prevcount, exists := all[ref]; exists {
									all[ref] = prevcount + count
								} else {
									all[ref] = count
								}
							}
						}
					} else {
						err = fetchDays(&site)
						if err != nil {
							return nil, err
						}
						for _, day := range site.couchDays {
							for ref, scoremap := range day.Sessions {
								count := (len(scoremap) - 1) / 2
								if prevcount, exists := all[ref]; exists {
									all[ref] = prevcount + count
								} else {
									all[ref] = count
								}
							}
						}
					}

					entries := EntriesFromMap(all)
					sort.Sort(sort.Reverse(EntrySort(entries)))

					return entries, nil
				},
			},
			"pages": &graphql.Field{
				Type:        graphql.NewList(entryType),
				Description: "a list of entries of viewed pages, sorted by the number of occurrences.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					site := p.Source.(Site)
					all := make(map[string]int)

					if site.lastDays > 90 {
						err = fetchMonths(&site)
						if err != nil {
							return nil, err
						}
						for _, month := range site.couchMonths {
							for ref, count := range month.TopPages {
								if prevcount, exists := all[ref]; exists {
									all[ref] = prevcount + count
								} else {
									all[ref] = count
								}
							}
						}
					} else {
						err = fetchDays(&site)
						if err != nil {
							return nil, err
						}
						for _, day := range site.couchDays {
							for addr, count := range day.Pages {
								if prevcount, exists := all[addr]; exists {
									all[addr] = prevcount + count
								} else {
									all[addr] = count
								}
							}
						}
					}

					entries := EntriesFromMap(all)
					sort.Sort(sort.Reverse(EntrySort(entries)))

					return entries, nil
				},
			},
			"sessionsbyreferrer": &graphql.Field{
				Type:        graphql.NewList(sessionGroupType),
				Description: "a list of tuples of type {referrer, []score}",
				Args: graphql.FieldConfigArgument{
					"limit": &graphql.ArgumentConfig{
						Type:         graphql.Int,
						DefaultValue: 400,
					},
					"minscore": &graphql.ArgumentConfig{
						Description:  "only scores equal or greater than this number.",
						Type:         graphql.Int,
						DefaultValue: 0,
					},
					"referrer": &graphql.ArgumentConfig{
						Description:  "only referrers with this host.",
						Type:         graphql.String,
						DefaultValue: "",
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					minscore, _ := p.Args["minscore"].(int)
					referrer, filterbyreferrer := p.Args["referrer"].(string)
					if filterbyreferrer {
						filterbyreferrer = referrer != ""
					}
					referrerhost := urlHost(referrer)
					limit := p.Args["limit"].(int)
					count := 0

					byref := make(map[string][]int)
					days := p.Source.(Site).couchDays
					for i := len(days) - 1; i >= 0; i-- { // from newest day to oldest
						day := days[i]
						for ref, scoremap := range day.Sessions {
							if filterbyreferrer && urlHost(ref) != referrerhost {
								continue
							}

							sessions := sessionsFromScoremap(scoremap)

							if _, exists := byref[ref]; !exists {
								byref[ref] = make([]int, 0, len(sessions))
							}

							for _, score := range sessions {
								if score < minscore {
									continue
								}
								byref[ref] = append(byref[ref], score)

								count++
								if count >= limit {
									goto finish
								}
							}
						}
					}

				finish:
					sessiongroups := make([]SessionGroup, len(byref))
					i := 0
					for ref, sessions := range byref {
						sessiongroups[i] = SessionGroup{ref, sessions}
						i++
					}

					return sessiongroups, nil
				},
			},
		},
	},
*/
