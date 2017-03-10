package main

import (
	"context"
	"errors"
	"log"
	"sort"
	"strconv"
	"strings"

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

var compendiumType = graphql.NewObject(
	graphql.ObjectConfig{
		Name:        "Compendium",
		Description: "A day, or a month, maybe an year -- a period of time for which there are stats",
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
					for _, count := range p.Source.(Compendium).Pages {
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
					for _, scoremap := range p.Source.(Compendium).Sessions {
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
					for _, scoremap := range p.Source.(Compendium).Sessions {
						l := len(scoremap)
						nsessions := (l - 1) / 2
						totalsessions += nsessions
						for s := 0; s < nsessions; s++ {
							if l >= s*2+3 && scoremap[s*2+1:s*2+3] == "01" {
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

var siteType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Site",
		Fields: graphql.Fields{
			"code":       &graphql.Field{Type: graphql.String},
			"name":       &graphql.Field{Type: graphql.String},
			"created_at": &graphql.Field{Type: graphql.String},
			"user_id":    &graphql.Field{Type: graphql.String},
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
				Type: graphql.NewList(compendiumType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					site := p.Source.(Site)
					days := make([]Compendium, site.lastDays)

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
			"referrers": &graphql.Field{
				Type:        graphql.NewList(entryType),
				Description: "a list of entries of referrers, sorted by the number of occurrences.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					all := make(map[string]int)
					for _, compendium := range p.Source.(Site).couchDays {
						for ref, scoremap := range compendium.Sessions {
							count := (len(scoremap) - 1) / 2
							if prevcount, exists := all[ref]; exists {
								all[ref] = prevcount + count
							} else {
								all[ref] = count
							}
						}
					}

					entries := make([]Entry, len(all))
					i := 0
					for ref, count := range all {
						entries[i] = Entry{ref, count}
						i++
					}
					sort.Sort(sort.Reverse(EntrySort(entries)))

					return entries, nil
				},
			},
			"pages": &graphql.Field{
				Type:        graphql.NewList(entryType),
				Description: "a list of entries of viewed pages, sorted by the number of occurrences.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					all := make(map[string]int)
					for _, compendium := range p.Source.(Site).couchDays {
						for addr, count := range compendium.Pages {
							if prevcount, exists := all[addr]; exists {
								all[addr] = prevcount + count
							} else {
								all[addr] = count
							}
						}
					}

					entries := make([]Entry, len(all))
					i := 0
					for addr, count := range all {
						entries[i] = Entry{addr, count}
						i++
					}
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

							if _, exists := byref[ref]; !exists {
								byref[ref] = make([]int, 0)
							}

							l := len(scoremap)
							nsessions := (l - 1) / 2
							for s := 0; s < nsessions; s++ {
								if l >= s*2+3 {
									score, err := strconv.Atoi(scoremap[s*2+1 : s*2+3])
									if err != nil || score < minscore {
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
				Type: compendiumType,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					sitecode := p.Source.(Site).Code
					today := presentDay().Format(DATEFORMAT)
					compendium := compendiumFromRedis(sitecode, today)
					compendium.Day = today
					return compendium, nil
				},
			},
			"months": &graphql.Field{Type: graphql.NewList(compendiumType)},
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
			"id": &graphql.Field{Type: graphql.String},
			"plan": &graphql.Field{
				Type:        graphql.Float,
				Description: "the plan this user is currently in.",
			},
			"sites": &graphql.Field{
				Type: graphql.NewList(siteType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					var sites []Site
					err = pg.Raw(`
SELECT code, name, user_id, to_char(created_at, 'YYYYMMDD') AS created_at, shared
  FROM sites
  INNER JOIN (
    SELECT unnest(sites_order) AS c,
           generate_subscripts(sites_order, 1) as o FROM users
  )t ON c = code
WHERE user_id = ?
ORDER BY o`, p.Source.(User).Id).Scan(&sites)
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
					Select("id, array_to_string(domains, ','), colours, plan").
					Where("id = ?", userIdFromContext(p.Context)).
					Scan(&user)
				if err != nil {
					return nil, err
				}
				if user.Id == "" {
					return nil, errors.New("you must be logged in to query 'me'.")
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
				if site.UserId != userIdFromContext(p.Context) /* not owner */ &&
					!site.Shared /* not shared */ {
					return nil, errors.New("you're not authorized to view this site.")
				}

				// do we need to fetch the compendium list from couchdb?
				site.lastDays = p.Args["last"].(int)
				if site.lastDays <= 0 {
					// no.
					return site, nil
				}

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
				site.couchDays = res.toCompendiumList()

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
				userId := userIdFromContext(p.Context)
				code := makeCodeForUser(userId)
				site := Site{
					Code:   code,
					Name:   p.Args["name"].(string),
					UserId: userId,
				}

				tx := pg.Begin()
				if err = tx.Create(&site); err != nil {
					tx.Rollback()
					return nil, err
				}

				err = tx.Exec(
					`UPDATE users SET sites_order = array_append(sites_order, ?) WHERE id = ?`,
					site.Code, userId)
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
				if site.UserId != userIdFromContext(p.Context) {
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
				userId := userIdFromContext(p.Context)
				code := p.Args["code"].(string)

				tx := pg.Begin()

				err = tx.Exec(
					`DELETE FROM sites WHERE code = ? AND user_id = ?`,
					code, userId,
				)
				if err != nil {
					tx.Rollback()
					return Result{false, err.Error()}, err
				}

				err = tx.Exec(
					`UPDATE users SET sites_order = array_remove(sites_order, ?) WHERE id = ?`,
					code, userId,
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
				userId := userIdFromContext(p.Context)
				icodes := p.Args["order"].([]interface{})
				codes := make([]string, len(icodes))
				for i, icode := range icodes {
					codes[i] = icode.(string)
				}
				order := strings.Join(codes, "@^~^@")
				err = pg.Exec(
					`UPDATE users SET sites_order = string_to_array(?, '@^~^@') WHERE id = ?`,
					order, userId)
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
				userId := userIdFromContext(p.Context)
				code := p.Args["code"].(string)
				share := p.Args["share"].(bool)
				err = pg.Exec(
					`UPDATE sites SET shared = ? WHERE code = ? and user_id = ?`,
					share, code, userId)
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
				userId := userIdFromContext(p.Context)
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
					`UPDATE users SET domains = array_append(array_remove(domains, ?), ?) WHERE id = ?`,
					host, host, userId)
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
				userId := userIdFromContext(p.Context)
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
					`UPDATE users SET domains = array_remove(domains, ?) WHERE id = ?`,
					host, userId)
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
				userId := userIdFromContext(p.Context)
				colours := make(igor.JSON)

				for field, colour := range p.Args["colours"].(map[string]interface{}) {
					colours[field] = colour
				}

				err = pg.Exec(
					`UPDATE users SET colours = ? WHERE id = ?`,
					colours, userId)
				if err != nil {
					return Result{false, err.Error()}, err
				}

				return Result{true, ""}, nil
			},
		},
	},
}

func fetchSite(code string) (site Site, err error) {
	err = pg.Model(site).
		Select("code, name, user_id, to_char(created_at, 'YYYYMMDD') AS created_at, shared").
		Where("code = ?", code).
		Scan(&site)
	return site, err
}

func userIdFromContext(ctx context.Context) string {
	if userId, ok := ctx.Value("loggeduser").(string); ok {
		return userId
	}
	return ""
}
