package main

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/fjl/go-couchdb"
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
			},
			"c": &graphql.Field{
				Type:        graphql.Int,
				Description: "the number of times it appeared.",
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
			SESSIONS: &graphql.Field{
				Type:        graphql.Int,
				Description: "total number of sessions.",
			},
			PAGEVIEWS: &graphql.Field{
				Type:        graphql.Int,
				Description: "total number of pageviews.",
			},
			REFERRERS: &graphql.Field{
				Type:        graphql.NewList(entryType),
				Description: "a list of entries of referrers, sorted by the number of occurrences.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					d := p.Source.(Compendium).Referrers
					entries := make([]Entry, len(d))
					i := 0
					for addr, count := range d {
						entries[i] = Entry{addr, count}
						i++
					}
					sort.Sort(EntrySort(entries))
					return entries, nil
				},
			},
			PAGES: &graphql.Field{
				Type:        graphql.NewList(entryType),
				Description: "a list of entries of viewed pages, sorted by the number of occurrences.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					d := p.Source.(Compendium).Pages
					entries := make([]Entry, len(d))
					i := 0
					for addr, count := range d {
						entries[i] = Entry{addr, count}
						i++
					}
					sort.Sort(EntrySort(entries))
					return entries, nil
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
			"days": &graphql.Field{
				Type: graphql.NewList(compendiumType),
				Args: graphql.FieldConfigArgument{
					"last": &graphql.ArgumentConfig{
						Description:  "number of last days to fetch",
						Type:         graphql.Int,
						DefaultValue: 30,
					},
				},
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					res := CouchDBResults{}
					sitecode := p.Source.(Site).Code
					last := p.Args["last"].(int)
					yesterday := presentDay().AddDate(0, 0, -1)
					startday := yesterday.AddDate(0, 0, -last)
					err := couch.AllDocs(&res, couchdb.Options{
						"startkey":     makeBaseKey(sitecode, startday.Format(DATEFORMAT)),
						"endkey":       makeBaseKey(sitecode, yesterday.Format(DATEFORMAT)),
						"include_docs": true,
					})
					if err != nil {
						return nil, err
					}
					fetcheddays := res.toCompendiumList()
					days := make([]Compendium, last+1)

					// fill in missing days with zeroes
					current := startday
					currentpos := 0
					fetchedpos := 0
					for !current.After(yesterday) {
						if fetcheddays[fetchedpos].Day == current.Format(DATEFORMAT) {
							days[currentpos] = fetcheddays[fetchedpos]
							fetchedpos++
						} else {
							days[currentpos].Day = current.Format(DATEFORMAT)
						}
						current = current.AddDate(0, 0, 1)
						currentpos++
					}

					return days, nil
				},
			},
			"months": &graphql.Field{Type: graphql.NewList(compendiumType)},
		},
	},
)

var userType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "User",
		Fields: graphql.Fields{
			"id":   &graphql.Field{Type: graphql.Int},
			"name": &graphql.Field{Type: graphql.String},
			"sites": &graphql.Field{
				Type: graphql.NewList(siteType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					var sites []Site
					err = pg.Raw(`
SELECT code, name, user_id, created_at FROM sites
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
		},
	},
)

var rootQuery = graphql.ObjectConfig{
	Name: "RootQuery",
	Fields: graphql.Fields{
		"me": &graphql.Field{
			Type: userType,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				user := User{Id: p.Context.Value("loggeduser").(int)}
				err = pg.Model(user).
					Select("id, name, email").
					Where(user).
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
					Type:        graphql.String,
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				site := Site{Code: p.Args["code"].(string)}
				if err := pg.Model(site).Where(site).Scan(&site); err != nil {
					return nil, err
				}
				if site.UserId != p.Context.Value("loggeduser").(int) {
					return nil, errors.New("you're not authorized to view this site.")
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
			"ok": &graphql.Field{Type: graphql.Boolean},
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
					Type:        graphql.String,
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				userId := p.Context.Value("loggeduser").(int)
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
					Type:        graphql.String,
				},
				"name": &graphql.ArgumentConfig{
					Description: "a new name for the site",
					Type:        graphql.String,
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				site := Site{Code: p.Args["code"].(string)}
				if err := pg.Model(site).Where(site).Scan(&site); err != nil {
					return nil, err
				}
				if site.UserId != p.Context.Value("loggeduser").(int) {
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
					Type:        graphql.String,
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				user := p.Context.Value("loggeduser").(int)
				code := p.Args["code"].(string)

				tx := pg.Begin()

				err = tx.Exec(
					`DELETE FROM sites WHERE code = ? AND user_id = ?`,
					code, user,
				)
				if err != nil {
					tx.Rollback()
					return Result{false}, err
				}

				err = tx.Exec(
					`UPDATE users SET sites_order = array_remove(sites_order, ?) WHERE id = ?`,
					code, user,
				)
				if err != nil {
					tx.Rollback()
					return Result{false}, err
				}

				if err = tx.Commit(); err != nil {
					tx.Rollback()
					return Result{false}, err
				}

				return Result{true}, nil
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
				user := p.Context.Value("loggeduser").(int)
				icodes := p.Args["order"].([]interface{})
				codes := make([]string, len(icodes))
				for i, icode := range icodes {
					codes[i] = icode.(string)
				}
				order := strings.Join(codes, "@^~^@")
				err = pg.Exec(
					`UPDATE users SET sites_order = string_to_array(?, '@^~^@') WHERE id = ?`,
					order, user)
				if err != nil {
					return Result{false}, err
				}
				return Result{true}, nil
			},
		},
	},
}
