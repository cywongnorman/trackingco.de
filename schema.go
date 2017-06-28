package main

import (
	"context"
	"errors"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/fjl/go-couchdb"
	"github.com/galeone/igor"
	"github.com/graphql-go/graphql"
)

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func query(req GraphQLRequest, context context.Context) *graphql.Result {
	r := graphql.Do(graphql.Params{
		Schema:         schema,
		RequestString:  req.Query,
		VariableValues: req.Variables,
		Context:        context,
	})
	return r
}

var schema graphql.Schema

func init() {
	schema, _ = graphql.NewSchema(graphql.SchemaConfig{
		Query:    graphql.NewObject(rootQuery),
		Mutation: graphql.NewObject(rootMutation),
	})
}

var entryType = graphql.NewObject(
	graphql.ObjectConfig{
		Name:        "Entry",
		Description: "A tuple of address / count",
		Fields: graphql.Fields{
			"a": &graphql.Field{
				Type:        graphql.String,
				Description: "the web address.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					addr := p.Source.(Entry).Address
					if addr == "" {
						return "<direct>", nil
					}
					return addr, nil
				},
			},
			"c": &graphql.Field{
				Type:        graphql.Int,
				Description: "the number of times it has appeared.",
			},
		},
	},
)

var sessionGroupType = graphql.NewObject(
	graphql.ObjectConfig{
		Name:        "SessionGroup",
		Description: "a type {referrer, []score}",
		Fields: graphql.Fields{
			"referrer": &graphql.Field{
				Type:        graphql.String,
				Description: "the referrer.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					addr := p.Source.(SessionGroup).Referrer
					if addr == "" {
						return "<direct>", nil
					}
					return addr, nil
				},
			},
			"scores": &graphql.Field{
				Type:        graphql.NewList(graphql.Int),
				Description: "the score of the session.",
			},
		},
	},
)

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

var siteType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Site",
		Fields: graphql.Fields{
			"code":       &graphql.Field{Type: graphql.String},
			"name":       &graphql.Field{Type: graphql.String},
			"created_at": &graphql.Field{Type: graphql.String},
			"user_email": &graphql.Field{Type: graphql.String},
			"shareURL": &graphql.Field{
				Type:        graphql.String,
				Description: "the URL to share this site's statistics. it is a blank string when the site is not shared.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					site := p.Source.(Site)
					if site.Shared {
						return s.Host + "/public/" + site.Code, nil
					}
					return "", nil
				},
			},
			"days": &graphql.Field{
				Type: graphql.NewList(dayType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					site := p.Source.(Site)
					days := make([]Day, site.lastDays)

					// fill in missing days with zeroes
					today := presentDay()
					yesterday := today.AddDate(0, 0, -1)
					current := today.AddDate(0, 0, -site.lastDays)
					currentpos := 0
					couchpos := 0
					for !current.After(yesterday) {
						if site.couchDays[couchpos].Day == current.Format(DATEFORMAT) {
							days[currentpos] = site.couchDays[couchpos]
							couchpos++
						} else {
							days[currentpos].Day = current.Format(DATEFORMAT)
						}
						current = current.AddDate(0, 0, 1)
						currentpos++
					}

					return days, nil
				},
			},
			"months": &graphql.Field{
				Type: graphql.NewList(monthType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					site := p.Source.(Site)
					var months []Month

					if len(site.couchMonths) == 0 {
						return months, nil
					}

					// fill in missing days with zeroes
					today := presentDay()
					lastmonth := today.AddDate(0, -1, 0)
					first, err := time.Parse(MONTHFORMAT, site.couchMonths[0].Month)
					if err != nil {
						return months, err
					}
					current := first
					couchpos := 0
					for !current.After(lastmonth) {
						if site.couchMonths[couchpos].Month == current.Format(MONTHFORMAT) {
							months = append(months, site.couchMonths[couchpos])
							couchpos++
						} else {
							months = append(months, Month{Month: current.Format(MONTHFORMAT)})
						}
						current = current.AddDate(0, 1, 0)
					}

					return months, nil
				},
			},
			"referrers": &graphql.Field{
				Type:        graphql.NewList(entryType),
				Description: "a list of entries of referrers, sorted by the number of occurrences.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					site := p.Source.(Site)
					all := make(map[string]int)

					if site.usingMonths {
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

					if site.usingMonths {
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
						for _, day := range p.Source.(Site).couchDays {
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
					referrer, filterreferrer := p.Args["referrer"].(string)
					referrerhost := urlHost(referrer)
					limit := p.Args["limit"].(int)
					count := 0

					byref := make(map[string][]int)
					days := p.Source.(Site).couchDays
					for i := len(days) - 1; i >= 0; i-- { // from newest day to oldest
						day := days[i]
						for ref, scoremap := range day.Sessions {
							if filterreferrer && urlHost(ref) != referrerhost {
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
			"today": &graphql.Field{
				Type: dayType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					sitecode := p.Source.(Site).Code
					today := presentDay().Format(DATEFORMAT)
					day := dayFromRedis(sitecode, today)
					day.Day = today
					return day, nil
				},
			},
		},
	},
)

var coloursType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Colours",
		Fields: graphql.Fields{
			"bar1": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return p.Source.(igor.JSON)["bar1"], nil
				},
			},
			"line1": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return p.Source.(igor.JSON)["line1"], nil
				},
			},
			"background": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return p.Source.(igor.JSON)["background"], nil
				},
			},
		},
	},
)

var coloursInput = graphql.NewInputObject(
	graphql.InputObjectConfig{
		Name: "ColoursInput",
		Fields: graphql.InputObjectConfigFieldMap{
			"bar1":       &graphql.InputObjectFieldConfig{Type: graphql.String},
			"line1":      &graphql.InputObjectFieldConfig{Type: graphql.String},
			"background": &graphql.InputObjectFieldConfig{Type: graphql.String},
		},
	},
)

var userType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "User",
		Fields: graphql.Fields{
			"email": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return emailFromContext(p.Context), nil
				},
			},
			"plan": &graphql.Field{
				Type:        graphql.Float,
				Description: "the billing plan this user is currently in.",
			},
			"sites": &graphql.Field{
				Type: graphql.NewList(siteType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					var sites []Site
					err = pg.Raw(`
SELECT code, name, user_email, to_char(created_at, 'YYYYMMDD') AS created_at, shared
  FROM sites
  INNER JOIN (
    SELECT unnest(sites_order) AS c,
           generate_subscripts(sites_order, 1) as o FROM users
  )t ON c = code
WHERE user_email = ?
ORDER BY o`, p.Source.(User).Email).Scan(&sites)
					if err != nil {
						return nil, err
					}
					return sites, nil
				},
			},
			"domains": &graphql.Field{
				Type: graphql.NewList(graphql.String),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					list := strings.Split(p.Source.(User).Domains, ",")
					if len(list) == 1 && list[0] == "" {
						return []string{}, nil
					} else {
						return list, nil
					}
				},
			},
			"colours": &graphql.Field{Type: coloursType},
			"billingHistory": &graphql.Field{
				Type: graphql.NewList(billingEntryType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					var entries []BillingEntry
					err = pg.Raw(`
SELECT
  id,
  to_char(time, 'YYYY-MM-dd'),
  delta,
  CASE WHEN due IS NULL THEN '' ELSE to_char(time + due, 'YYYY-MM-dd') END AS due
FROM balances
WHERE user_email = ?
ORDER BY time DESC, delta
                    `, p.Source.(User).Email).Scan(&entries)
					if err != nil {
						return nil, err
					}
					return entries, nil
				},
			},
		},
	},
)

var billingEntryType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "BillingEntry",
		Fields: graphql.Fields{
			"id":    &graphql.Field{Type: graphql.Float},
			"time":  &graphql.Field{Type: graphql.String},
			"delta": &graphql.Field{Type: graphql.Float},
			"due":   &graphql.Field{Type: graphql.String},
		},
	},
)

var rootQuery = graphql.ObjectConfig{
	Name: "RootQuery",
	Fields: graphql.Fields{
		"me": &graphql.Field{
			Type: userType,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				user := User{}
				err = pg.Model(user).
					Select("email, array_to_string(domains, ','), colours, plan").
					Where("email = ?", emailFromContext(p.Context)).
					Scan(&user)
				if err != nil {
					return nil, err
				}
				return user, nil
			},
		},
		"site": &graphql.Field{
			Type: siteType,
			Args: graphql.FieldConfigArgument{
				"code": &graphql.ArgumentConfig{
					Description: "a site's unique tracking code.",
					Type:        graphql.NewNonNull(graphql.String),
				},
				"last": &graphql.ArgumentConfig{
					Description:  "number of last days to use (don't set if not requesting days, referrers or pages).",
					Type:         graphql.Int,
					DefaultValue: -1,
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				site, err := fetchSite(p.Args["code"].(string))
				if err != nil {
					return nil, err
				}
				if site.UserEmail != emailFromContext(p.Context) /* not owner */ &&
					!site.Shared /* not shared */ {
					return nil, errors.New("you're not authorized to view this site.")
				}

				// do we need to fetch the day list from couchdb?
				site.lastDays = p.Args["last"].(int)
				if site.lastDays <= 0 {
					// no.
					return site, nil
				} else if site.lastDays <= 90 {
					// yes.
					res := CouchDBDayResults{}
					today := presentDay()
					startday := today.AddDate(0, 0, -site.lastDays)
					err = couch.AllDocs(&res, couchdb.Options{
						"startkey":     makeBaseKey(site.Code, startday.Format(DATEFORMAT)),
						"endkey":       makeBaseKey(site.Code, today.Format(DATEFORMAT)),
						"include_docs": true,
					})
					if err != nil {
						return nil, err
					}
					site.couchDays = res.toDayList()
				} else {
					// no, we must fetch months instead.
					res := CouchDBMonthResults{}
					today := presentDay()
					startday := today.AddDate(0, 0, -site.lastDays)
					err = couch.AllDocs(&res, couchdb.Options{
						"startkey":     makeMonthKey(site.Code, startday.Format(MONTHFORMAT)),
						"endkey":       makeMonthKey(site.Code, today.Format(MONTHFORMAT)),
						"include_docs": true,
					})
					if err != nil {
						return nil, err
					}
					site.couchMonths = res.toMonthList()
					site.usingMonths = true
				}

				return site, nil
			},
		},
	},
}

var resultType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Result",
		Fields: graphql.Fields{
			"ok":    &graphql.Field{Type: graphql.Boolean},
			"error": &graphql.Field{Type: graphql.String},
		},
	},
)

var rootMutation = graphql.ObjectConfig{
	Name: "Mutation",
	Fields: graphql.Fields{
		"createSite": &graphql.Field{
			Type: siteType,
			Args: graphql.FieldConfigArgument{
				"name": &graphql.ArgumentConfig{
					Description: "a name to identify the site",
					Type:        graphql.NewNonNull(graphql.String),
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				email := emailFromContext(p.Context)
				code := makeCodeForUser(email)
				site := Site{
					Code:      code,
					Name:      p.Args["name"].(string),
					UserEmail: email,
				}

				tx := pg.Begin()
				if err = tx.Create(&site); err != nil {
					tx.Rollback()
					return nil, err
				}

				err = tx.Exec(
					`UPDATE users SET sites_order = array_append(sites_order, ?) WHERE email = ?`,
					site.Code, email)
				if err != nil {
					tx.Rollback()
					return nil, err
				}

				if err = tx.Commit(); err != nil {
					return nil, err
				}
				return site, nil
			},
		},
		"renameSite": &graphql.Field{
			Type: siteType,
			Args: graphql.FieldConfigArgument{
				"code": &graphql.ArgumentConfig{
					Description: "the code of the site to rename",
					Type:        graphql.NewNonNull(graphql.String),
				},
				"name": &graphql.ArgumentConfig{
					Description: "a new name for the site",
					Type:        graphql.NewNonNull(graphql.String),
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				site, err := fetchSite(p.Args["code"].(string))
				if err != nil {
					return nil, err
				}
				if site.UserEmail != emailFromContext(p.Context) {
					return nil, errors.New("you're not authorized to update this site.")
				}
				site.Name = p.Args["name"].(string)
				if err = pg.Updates(&site); err != nil {
					return nil, err
				}
				return site, nil
			},
		},
		"deleteSite": &graphql.Field{
			Type: resultType,
			Args: graphql.FieldConfigArgument{
				"code": &graphql.ArgumentConfig{
					Description: "the code of the site to rename",
					Type:        graphql.NewNonNull(graphql.String),
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				email := emailFromContext(p.Context)
				code := p.Args["code"].(string)

				tx := pg.Begin()

				err = tx.Exec(
					`DELETE FROM sites WHERE code = ? AND user_email = ?`,
					code, email,
				)
				if err != nil {
					tx.Rollback()
					return Result{false, err.Error()}, err
				}

				err = tx.Exec(
					`UPDATE users SET sites_order = array_remove(sites_order, ?) WHERE email = ?`,
					code, email,
				)
				if err != nil {
					tx.Rollback()
					return Result{false, err.Error()}, err
				}

				if err = tx.Commit(); err != nil {
					tx.Rollback()
					return Result{false, err.Error()}, err
				}

				return Result{true, ""}, nil
			},
		},
		"changeSiteOrder": &graphql.Field{
			Type: resultType,
			Args: graphql.FieldConfigArgument{
				"order": &graphql.ArgumentConfig{
					Description: "an array of all the sites codes in the desired order",
					Type:        graphql.NewList(graphql.String),
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				email := emailFromContext(p.Context)
				icodes := p.Args["order"].([]interface{})
				codes := make([]string, len(icodes))
				for i, icode := range icodes {
					codes[i] = icode.(string)
				}
				order := strings.Join(codes, "@^~^@")
				err = pg.Exec(
					`UPDATE users SET sites_order = string_to_array(?, '@^~^@') WHERE email = ?`,
					order, email)
				if err != nil {
					return Result{false, err.Error()}, err
				}
				return Result{true, ""}, nil
			},
		},
		"shareSite": &graphql.Field{
			Type: resultType,
			Args: graphql.FieldConfigArgument{
				"code": &graphql.ArgumentConfig{
					Description: "the code of the site to set sharing",
					Type:        graphql.NewNonNull(graphql.String),
				},
				"share": &graphql.ArgumentConfig{
					Description: "to share or to stop sharing.",
					Type:        graphql.NewNonNull(graphql.Boolean),
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				email := emailFromContext(p.Context)
				code := p.Args["code"].(string)
				share := p.Args["share"].(bool)
				err = pg.Exec(
					`UPDATE sites SET shared = ? WHERE code = ? and user_email = ?`,
					share, code, email)
				if err != nil {
					return Result{false, err.Error()}, err
				}
				return Result{true, ""}, nil
			},
		},
		"addDomain": &graphql.Field{
			Type: resultType,
			Args: graphql.FieldConfigArgument{
				"hostname": &graphql.ArgumentConfig{
					Type: graphql.NewNonNull(graphql.String),
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				email := emailFromContext(p.Context)
				host := p.Args["hostname"].(string)

				data := map[string]string{"hostname": host}
				resp, err := herokuDomains("POST", "", data)
				if err != nil {
					return Result{false, err.Error()}, err
				}

				if resp.Hostname == "" {
					if resp.Id == "invalid_params" && strings.Index(resp.Message, "already") != -1 {
						// ok, that's good.
					} else {
						log.Print("failed to add domain ", host, " ", resp)
						return Result{false, resp.Message}, nil
					}
				}

				err = pg.Exec(
					`UPDATE users SET domains = array_append(array_remove(domains, ?), ?) WHERE email = ?`,
					host, host, email)
				if err != nil {
					return Result{false, err.Error()}, err
				}
				return Result{true, ""}, nil
			},
		},
		"removeDomain": &graphql.Field{
			Type: resultType,
			Args: graphql.FieldConfigArgument{
				"hostname": &graphql.ArgumentConfig{
					Type: graphql.NewNonNull(graphql.String),
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				email := emailFromContext(p.Context)
				host := p.Args["hostname"].(string)

				resp, err := herokuDomains("DELETE", "/"+host, nil)
				if err != nil {
					return Result{false, err.Error()}, err
				}

				if resp.Hostname == "" {
					if resp.Id == "not_found" {
						// ok, that's good.
					} else {
						log.Print("failed to remove domain ", host, " ", resp)
						return Result{false, resp.Message}, nil
					}
				}

				err = pg.Exec(
					`UPDATE users SET domains = array_remove(domains, ?) WHERE email = ?`,
					host, email)
				if err != nil {
					return Result{false, err.Error()}, err
				}
				return Result{true, ""}, nil
			},
		},
		"setColours": &graphql.Field{
			Type: resultType,
			Args: graphql.FieldConfigArgument{
				"colours": &graphql.ArgumentConfig{
					Type:        coloursInput,
					Description: "an object with each colour definition.",
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				email := emailFromContext(p.Context)
				colours := make(igor.JSON)

				for field, colour := range p.Args["colours"].(map[string]interface{}) {
					colours[field] = colour
				}

				err = pg.Exec(
					`UPDATE users SET colours = ? WHERE email = ?`,
					colours, email)
				if err != nil {
					return Result{false, err.Error()}, err
				}

				return Result{true, ""}, nil
			},
		},
		"setPlan": &graphql.Field{
			Type: resultType,
			Args: graphql.FieldConfigArgument{
				"plan": &graphql.ArgumentConfig{
					Type:        graphql.Float,
					Description: "a plan number.",
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				email := emailFromContext(p.Context)
				plan := p.Args["plan"].(float64)

				tx := pg.Begin()

				var funds float64
				tx.Raw(`
                    SELECT sum(delta) FROM balances WHERE user_email = ?`,
					email).Scan(&funds)

				if funds < float64(planValues[plan]) {
					tx.Rollback()
					err := errors.New("Please fund your account before upgrading.")
					return Result{false, err.Error()}, err
				}

				tx.Exec(
					`UPDATE users SET plan = ? WHERE email = ?`,
					plan, email)

				if err = tx.Commit(); err != nil {
					return Result{false, err.Error()}, err
				}

				return Result{true, ""}, nil
			},
		},
	},
}

func fetchSite(code string) (site Site, err error) {
	err = pg.Model(site).
		Select("code, name, user_email, to_char(created_at, 'YYYYMMDD') AS created_at, shared").
		Where("code = ?", code).
		Scan(&site)
	return site, err
}

func emailFromContext(ctx context.Context) string {
	if email, ok := ctx.Value("loggeduser").(string); ok {
		return email
	}
	return ""
}
