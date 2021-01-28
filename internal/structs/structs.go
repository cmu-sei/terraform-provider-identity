// Copyright 2021 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.
package structs

import (
	"fmt"
	"sort"
)

// Account holds the info on an identity account
type Account struct {
	Usernames  []string
	Password   string
	Role       string
	Status     string
	ID         string
	GlobalID   string
	Properties []Property
}

// Property holds the info on a property within an account
type Property struct {
	AccountID int    `json:"accountId"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

// AsMap returns the map representation of a Property struct
func (prop *Property) AsMap() map[string]interface{} {
	return map[string]interface{}{
		"account_id": prop.AccountID,
		"key":        prop.Key,
		"value":      prop.Value,
	}
}

// PropertyFromMap returns a Property struct given the map representation of that struct
func PropertyFromMap(asMap map[string]interface{}) *Property {
	return &Property{
		Key:   asMap["key"].(string),
		Value: asMap["value"].(string),
	}
}

// Client holds information on an indentity client
type Client struct {
	ID          float64
	Name        string
	DisplayName string `json:"displayName"`
	Scopes      string
	Grants      string
	Enabled     bool
	// These lifetimes are not set (for now) but are required for the API to work
	// These fields need to be exported for json.marshal to work

	ConsentLifetime              string
	IdentityTokenLifetime        string
	AccessTokenLifetime          string
	AuthorizationCodeLifetime    string
	SlidingRefreshTokenLifetime  string
	AbsoluteRefreshTokenLifetime string

	RedirectURLs   []URL
	PostLogoutURLs []URL
	CorsURLs       []URL
	Claims         []Claim
	Secrets        []Secret
	Managers       []interface{} // This field only exists b/c we get a 400 when calling the API without it
}

// NewClient returns a new instance of a client with default values set
func NewClient(name, displayName, scopes, grants string, enabled bool) Client {
	ret := Client{}
	ret.Name = name
	ret.DisplayName = displayName
	ret.Enabled = enabled
	ret.Scopes = scopes
	ret.Grants = grants
	// Set up default values for lifetimes
	ret.ConsentLifetime = "30d"
	ret.IdentityTokenLifetime = "5m"
	ret.AccessTokenLifetime = "1h"
	ret.AuthorizationCodeLifetime = "5m"
	ret.SlidingRefreshTokenLifetime = "15d"
	ret.AbsoluteRefreshTokenLifetime = "30d"
	// Init array fields with empty arrays
	ret.RedirectURLs = []URL{}
	ret.PostLogoutURLs = []URL{}
	ret.CorsURLs = []URL{}
	ret.Claims = []Claim{}
	ret.Secrets = []Secret{}
	ret.Managers = []interface{}{}

	return ret
}

// SortFields sorts the slice fields within a client object
func (client *Client) SortFields() {
	sort.Slice((*client).RedirectURLs, func(i, j int) bool {
		return (*client).RedirectURLs[i].Value < (*client).RedirectURLs[j].Value
	})
	sort.Slice((*client).PostLogoutURLs, func(i, j int) bool {
		return (*client).PostLogoutURLs[i].Value < (*client).PostLogoutURLs[j].Value
	})
	sort.Slice((*client).CorsURLs, func(i, j int) bool {
		return (*client).CorsURLs[i].Value < (*client).CorsURLs[j].Value
	})

	sort.Slice((*client).Claims, func(i, j int) bool {
		return (*client).Claims[i].Value < (*client).Claims[j].Value
	})

	sort.Slice((*client).Secrets, func(i, j int) bool {
		return (*client).Secrets[i].Value < (*client).Secrets[j].Value
	})
}

// URL is an identity client url
type URL struct {
	ID       int
	Type     string
	Value    string
	ClientID int
	Deleted  bool
}

// URLFromMap returns a URL struct given an equivalent map
func URLFromMap(m map[string]interface{}) URL {
	// For some reason, go will sometimes think this field is a float.
	var id int
	t := fmt.Sprintf("%T", m["id"])
	if t == "int" {
		id = m["id"].(int)
	} else {
		id = int(m["id"].(float64))
	}

	// API and tf have different name for this field
	// Go also thinks this is a float when it comes from the API
	var clientID int
	if m["client_id"] != nil {
		clientID = m["client_id"].(int)
	} else {
		clientID = int(m["clientId"].(float64))
	}

	return URL{
		ID:       id,
		Type:     m["type"].(string),
		Value:    m["value"].(string),
		ClientID: clientID,
		Deleted:  m["deleted"].(bool),
	}
}

// AsMap returns the map representation of a URL struct
func (url URL) AsMap() map[string]interface{} {
	return map[string]interface{}{
		"id":        url.ID,
		"type":      url.Type,
		"value":     url.Value,
		"client_id": url.ClientID,
		"deleted":   url.Deleted,
	}

}

// Claim is an identity client claim
type Claim struct {
	ID       int
	Value    string
	ClientID int
	Deleted  bool
}

// ClaimFromMap returns a Claim struct given an equivalent map
func ClaimFromMap(m map[string]interface{}) Claim {
	var id int
	t := fmt.Sprintf("%T", m["id"])
	if t == "int" {
		id = m["id"].(int)
	} else {
		id = int(m["id"].(float64))
	}

	var clientID int
	if m["client_id"] != nil {
		clientID = m["client_id"].(int)
	} else {
		clientID = int(m["clientId"].(float64))
	}

	return Claim{
		ID:       id,
		Value:    m["value"].(string),
		ClientID: clientID,
		Deleted:  m["deleted"].(bool),
	}
}

// AsMap returns the map representation of a Claim struct
func (claim Claim) AsMap() map[string]interface{} {
	return map[string]interface{}{
		"id":        claim.ID,
		"value":     claim.Value,
		"client_id": claim.ClientID,
		"deleted":   claim.Deleted,
	}
}

// Secret is an identity client secret
type Secret struct {
	ID      int
	Value   string
	Deleted bool
}

// SecretFromMap returns a secret struct given an equivalent map
func SecretFromMap(m map[string]interface{}) Secret {
	return Secret{
		ID:      m["id"].(int),
		Value:   m["value"].(string),
		Deleted: m["deleted"].(bool),
	}
}

// AsMap returns the map representation of a Secret struct
func (secret Secret) AsMap() map[string]interface{} {
	return map[string]interface{}{
		"id":      secret.ID,
		"value":   secret.Value,
		"deleted": secret.Deleted,
	}
}
