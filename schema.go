package main

import (
	"context"

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
			"referrers": &graphql.Field{Type: graphql.NewList(entryType)},
			"pages":     &graphql.Field{Type: graphql.NewList(entryType)},
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
			"days":       &graphql.Field{Type: graphql.NewList(compendiumType)},
			"months":     &graphql.Field{Type: graphql.NewList(compendiumType)},
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
			"id":       &graphql.Field{Type: graphql.Int},
			"name":     &graphql.Field{Type: graphql.String},
			"sites":    &graphql.Field{Type: graphql.NewList(siteType)},
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
				user := User{
					Id: p.Context.Value("me").(int),
				}
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
				return nil, nil
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
				return nil, nil
			},
		},
		"deleteSite": &graphql.Field{
			Type: resultType,
			Args: graphql.FieldConfigArgument{
				"code": &graphql.ArgumentConfig{
					Description: "the unique code for the site",
					Type:        graphql.String,
				},
			},
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				return nil, nil
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
