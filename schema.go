package main

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/fiatjaf/accountd"
	"github.com/graphql-go/graphql"
	"github.com/jmoiron/sqlx/types"
	"github.com/timjacobi/go-couchdb"
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
			"owner":      &graphql.Field{Type: graphql.String},
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

					// do we need to fetch the day list from couchdb?
					if site.lastDays > 0 && site.lastDays <= 90 {
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
					}

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

					// do we need to fetch the month list from couchdb?
					if site.lastDays > 90 {
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

					months := make([]Month, 0, len(site.couchMonths))
					if len(site.couchMonths) == 0 {
						return months, nil
					}

					// fill in missing months with zeroes
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
			"bar1":       &graphql.Field{Type: graphql.String},
			"line1":      &graphql.Field{Type: graphql.String},
			"background": &graphql.Field{Type: graphql.String},
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
			"id": &graphql.Field{
				Type: graphql.String,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return userFromContext(p.Context), nil
				},
			},
			"sites": &graphql.Field{
				Type: graphql.NewList(siteType),
				Args: graphql.FieldConfigArgument{
					"last": &graphql.ArgumentConfig{
						Description:  "number of last days to use (don't set if not requesting days, referrers or pages).",
						Type:         graphql.Int,
						DefaultValue: -1,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					var sites []Site
					err = pg.Select(&sites, `
SELECT code, name, owner, to_char(created_at, 'YYYYMMDD') AS created_at, shared
  FROM sites
  INNER JOIN (
    SELECT unnest(sites_order) AS c,
           generate_subscripts(sites_order, 1) as o FROM users
  )t ON c = code
WHERE owner = $1
ORDER BY o
                    `, p.Source.(User).Id)
					if err != nil {
						return nil, err
					}

					for i := range sites {
						sites[i].lastDays = p.Args["last"].(int)
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
			"nmonths": &graphql.Field{Type: graphql.Int},
			"colours": &graphql.Field{
				Type: coloursType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					u, ok := p.Source.(User)
					if !ok {
						return nil, errors.New("failed to fetch colours")
					}

					t := make(map[string]interface{})
					err := (&u.Colours).Unmarshal(&t)
					return t, err
				},
			},
			"payments": &graphql.Field{
				Type: graphql.NewList(paymentType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					var payments []Payment
					err = pg.Select(&payments, `
SELECT
  id, user_id, created_at, amount,
  coalesce(paid_at, to_timestamp(0)) AS paid_at,
  (paid_at IS NOT NULL) AS has_paid
FROM payments
WHERE user_id = $1
ORDER BY created_at DESC
                    `, p.Source.(User).Id)
					if err != nil {
						return nil, err
					}
					return payments, nil
				},
			},
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
				err = pg.Get(&user, `
SELECT
  id,
  array_to_string(domains, ',') AS domains,
  colours, 
  cardinality(months_using) AS nmonths
FROM users WHERE id = $1
                `, userFromContext(p.Context))
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
				if site.Owner != userFromContext(p.Context) /* not owner */ &&
					!site.Shared /* not shared */ {
					return nil, errors.New("you're not authorized to view this site.")
				}

				site.lastDays = p.Args["last"].(int)

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

var strikeChargeType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "StrikeChargeType",
		Fields: graphql.Fields{
			"id":              &graphql.Field{Type: graphql.String},
			"amount":          &graphql.Field{Type: graphql.Int},
			"amount_satoshi":  &graphql.Field{Type: graphql.Int},
			"payment_hash":    &graphql.Field{Type: graphql.String},
			"payment_request": &graphql.Field{Type: graphql.String},
			"created":         &graphql.Field{Type: graphql.Int},
			"description":     &graphql.Field{Type: graphql.String},
			"paid":            &graphql.Field{Type: graphql.Boolean},
		},
	},
)

var paymentType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "PaymentType",
		Fields: graphql.Fields{
			"id":         &graphql.Field{Type: graphql.String},
			"amount":     &graphql.Field{Type: graphql.Int},
			"created_at": &graphql.Field{Type: graphql.String},
			"has_paid":   &graphql.Field{Type: graphql.Boolean},
			"paid_at":    &graphql.Field{Type: graphql.String},
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
				user := userFromContext(p.Context)
				code := makeCodeForUser(user)

				// ensure user account exists
				if err = ensureUser(user); err != nil {
					return nil, err
				}

				var site Site
				err = pg.Get(&site, `
WITH
ins AS (
  INSERT INTO sites (code, owner, name)
  VALUES ($1, $2, $3)
  RETURNING *
),
upd AS (
  UPDATE users
  SET sites_order = array_append(sites_order, $1)
  WHERE id = $2
)
SELECT * FROM ins
                `, code, user, p.Args["name"].(string))

				return site, err
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
				user := userFromContext(p.Context)
				code := p.Args["code"].(string)
				name := p.Args["name"].(string)

				var site Site
				err := pg.Get(&site, `
WITH upd AS (
  UPDATE sites SET name = $2
  WHERE code = $1 AND owner = $3
)
SELECT * FROM sites WHERE code = $1
                `, code, name, user)

				if err != nil {
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
				user := userFromContext(p.Context)
				code := p.Args["code"].(string)

				_, err := pg.Exec(`
WITH
del AS (
  DELETE FROM sites
  WHERE code = $1 AND owner = $2
),
UPDATE users
SET sites_order = array_remove(sites_order, $1)
WHERE id = $2
                `, code, user)
				if err != nil {
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
				user := userFromContext(p.Context)
				icodes := p.Args["order"].([]interface{})
				codes := make([]string, len(icodes))
				for i, icode := range icodes {
					codes[i] = icode.(string)
				}
				order := strings.Join(codes, "@^~^@")
				_, err = pg.Exec(`
UPDATE users
SET sites_order = string_to_array($1, '@^~^@')
WHERE id = $2
                `, order, user)
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
				user := userFromContext(p.Context)
				code := p.Args["code"].(string)
				share := p.Args["share"].(bool)

				_, err = pg.Exec(`
UPDATE sites
SET shared = $3
WHERE code = $1 and owner = $2
                `, code, user, share)
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
				user := userFromContext(p.Context)
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

				_, err = pg.Exec(`
UPDATE users
SET domains = array_append(array_remove(domains, $2), $2)
WHERE id = $1
                `, user, host)
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
				user := userFromContext(p.Context)
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

				_, err = pg.Exec(`
UPDATE users
SET domains = array_remove(domains, $2)
WHERE id = $1
                `, user, host)
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
				user := userFromContext(p.Context)
				colours := p.Args["colours"].(map[string]interface{})

				x, err := json.Marshal(colours)
				if err != nil {
					return Result{false, err.Error()}, err
				}

				var coloursjson types.JSONText
				err = coloursjson.UnmarshalJSON(x)
				if err != nil {
					return Result{false, err.Error()}, err
				}

				_, err = pg.Exec(`
UPDATE users
SET colours = $1
WHERE id = $2
                `, coloursjson, user)
				if err != nil {
					return Result{false, err.Error()}, err
				}

				return Result{true, ""}, nil
			},
		},
		"createCharge": &graphql.Field{
			Type: strikeChargeType,
			Args: graphql.FieldConfigArgument{
				"amount": &graphql.ArgumentConfig{
					Type:        graphql.Int,
					Description: "Amount in satoshis",
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				user := userFromContext(p.Context)
				amount := p.Args["amount"].(int)
				return createCharge(user, amount)
			},
		},
	},
}

func fetchSite(code string) (site Site, err error) {
	err = pg.Get(&site, `
SELECT code, name, owner, to_char(created_at, 'YYYYMMDD') AS created_at, shared
FROM SITES WHERE code = $1
    `, code)
	return site, err
}

func userFromContext(ctx context.Context) string {
	if user, ok := ctx.Value("loggeduser").(string); ok {
		return user
	}
	return ""
}

func ensureUser(user string) (err error) {
	if _, err = pg.Exec(`
INSERT INTO users
(id, sites_order, colours)
VALUES ($1, '{}'::text[], '{}')
ON CONFLICT (id) DO NOTHING
  `, user); err != nil {
		return err
	}

	_, err = rewriteAccounts(user)
	return err
}

func rewriteAccounts(user string) (n int, err error) {
	look, err := accountd.Lookup(user)
	if err != nil {
		return
	}
	if len(look.Accounts) == 0 {
		return
	}

	for _, acc := range look.Accounts {
		_, err = pg.Exec(`
WITH su AS (
  UPDATE sites SET owner = $1 WHERE owner = $2
)
UPDATE users SET id = $1 WHERE id = $2
        `, look.Id, acc.Account)
		if err != nil {
			return
		}
		n++
	}

	return
}
