package main

import (
	"context"
	"errors"
	"time"

	"github.com/fjl/go-couchdb"
	"github.com/graphql-go/graphql"
	"github.com/lucsky/cuid"
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
			"address": &graphql.Field{Type: graphql.String},
			"count":   &graphql.Field{Type: graphql.Int},
		},
	},
)

var compendiumType = graphql.NewObject(
	graphql.ObjectConfig{
		Name:        "Compendium",
		Description: "A day, or a month, maybe an year -- a period of time for which there are stats",
		Fields: graphql.Fields{
			"sessions":  &graphql.Field{Type: graphql.Int},
			"pageviews": &graphql.Field{Type: graphql.Int},
			"referrers": &graphql.Field{
				Type: graphql.NewList(entryType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					d := p.Source.(Compendium).Referrers
					entries := make([]Entry, len(d))
					i := 0
					for addr, count := range d {
						entries[i] = Entry{addr, count}
						i++
					}
					return entries, nil
				},
			},
			"pages": &graphql.Field{
				Type: graphql.NewList(entryType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					d := p.Source.(Compendium).Pages
					entries := make([]Entry, len(d))
					i := 0
					for addr, count := range d {
						entries[i] = Entry{addr, count}
						i++
					}
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
					oldestdate := time.Now().AddDate(0, 0, -p.Args["last"].(int)).Format("20060102")
					err := couch.AllDocs(&res, couchdb.Options{
						"startkey": sitecode + ":" + oldestdate,
						"endkey":   sitecode + ":",
					})
					if err != nil {
						return nil, err
					}
					return res.toCompendiumList(), nil
				},
			},
			"months": &graphql.Field{Type: graphql.NewList(compendiumType)},
		},
	},
)

var settingsType = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "Settings",
		Fields: graphql.Fields{
			"sites_order": &graphql.Field{Type: graphql.NewList(graphql.String)},
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
					err = pg.Model(Site{}).Where("user_id = ?", p.Source.(User).Id).Scan(&sites)
					if err != nil {
						return nil, err
					}
					return sites, nil
				},
			},
			"settings": &graphql.Field{Type: settingsType},
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
				if err := pg.Model(user).Where(user).Scan(&user); err != nil {
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
				site := Site{
					Code:   cuid.New(),
					Name:   p.Args["name"].(string),
					UserId: p.Context.Value("loggeduser").(int),
				}
				if err := pg.Create(&site); err != nil {
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
				err = pg.Where(
					"code = ? AND user_id = ?",
					p.Args["code"].(string), p.Context.Value("loggeduser").(int)).
					Delete(Site{})
				if err != nil {
					return nil, err
				}
				return Result{true}, nil
			},
		},
		"changeSiteOrder": &graphql.Field{
			Type: resultType,
			Args: graphql.FieldConfigArgument{
				"order": &graphql.ArgumentConfig{
					Description: "an array of all the sites codes in the desired order",
					Type:        graphql.String,
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				return nil, nil
			},
		},
	},
}
