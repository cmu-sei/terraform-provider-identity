/*
Crucible
Copyright 2020 Carnegie Mellon University.
NO WARRANTY. THIS CARNEGIE MELLON UNIVERSITY AND SOFTWARE ENGINEERING INSTITUTE MATERIAL IS FURNISHED ON AN "AS-IS" BASIS. CARNEGIE MELLON UNIVERSITY MAKES NO WARRANTIES OF ANY KIND, EITHER EXPRESSED OR IMPLIED, AS TO ANY MATTER INCLUDING, BUT NOT LIMITED TO, WARRANTY OF FITNESS FOR PURPOSE OR MERCHANTABILITY, EXCLUSIVITY, OR RESULTS OBTAINED FROM USE OF THE MATERIAL. CARNEGIE MELLON UNIVERSITY DOES NOT MAKE ANY WARRANTY OF ANY KIND WITH RESPECT TO FREEDOM FROM PATENT, TRADEMARK, OR COPYRIGHT INFRINGEMENT.
Released under a MIT (SEI)-style license, please see license.txt or contact permission@sei.cmu.edu for full terms.
[DISTRIBUTION STATEMENT A] This material has been approved for public release and unlimited distribution.  Please see Copyright notice for non-US Government use and distribution.
Carnegie Mellon(R) and CERT(R) are registered in the U.S. Patent and Trademark Office by Carnegie Mellon University.
DM20-0181
*/

package provider

import (
	"fmt"
	"identity_provider/internal/api"
	"identity_provider/internal/structs"
	"log"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func identityClient() *schema.Resource {
	return &schema.Resource{
		Create: identityClientCreate,
		Read:   identityClientRead,
		Update: identityClientUpdate,
		Delete: identityClientDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"scopes": {
				Type:     schema.TypeString,
				Required: true,
			},
			"grants": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "client_credentials",
			},
			"url": {
				Type:     schema.TypeList,
				Required: true, // We actually need 3 of these, but tf isn't that smart
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: func(value interface{}, something string) ([]string, []error) {
								str := value.(string)
								if str != "redirectUri" && str != "corsUri" && str != "postLogoutRedirectUri" {
									return nil, []error{fmt.Errorf("Invalid url type")}
								}
								return nil, nil
							},
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
						"client_id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"deleted": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
			// Default values for these blocks may need to be revisited
			"claim": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
						"client_id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"deleted": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
			"secret": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"value": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"deleted": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  false,
						},
					},
				},
			},
		},
	}
}

func identityClientCreate(d *schema.ResourceData, m interface{}) error {
	client := structs.NewClient(d.Get("name").(string), d.Get("scopes").(string), d.Get("grants").(string))

	casted := m.(map[string]string)
	id, err := api.CreateClient(&client, casted)
	if err != nil {
		return err
	}

	// handle URLs
	urlList := d.Get("url").([]interface{})
	redirect := new([]structs.URL)
	cors := new([]structs.URL)
	postlogout := new([]structs.URL)

	for _, url := range urlList {
		asMap := url.(map[string]interface{})
		t := asMap["type"].(string)
		urlObj := structs.URLFromMap(asMap)
		urlObj.ClientID, _ = strconv.Atoi(id) // ID is always an int, don't need to worry about error value
		if t == "redirectUri" {
			*redirect = append(*redirect, urlObj)
		} else if t == "corsUri" {
			*cors = append(*cors, urlObj)
		} else {
			*postlogout = append(*postlogout, urlObj)
		}
	}
	client.RedirectURLs = *redirect
	client.CorsURLs = *cors
	client.PostLogoutURLs = *postlogout

	// handle claims
	claims := new([]structs.Claim)
	claimList := d.Get("claim").([]interface{})

	for _, claim := range claimList {
		asMap := claim.(map[string]interface{})
		claimObj := structs.ClaimFromMap(asMap)
		claimObj.ClientID, _ = strconv.Atoi(id)
		*claims = append(*claims, claimObj)
	}
	client.Claims = *claims

	// handle secrets
	secrets := new([]structs.Secret)
	secretList := d.Get("secret").([]interface{})

	for _, sec := range secretList {
		asMap := sec.(map[string]interface{})
		*secrets = append(*secrets, structs.SecretFromMap(asMap))
	}
	client.Secrets = *secrets

	client.Managers = []interface{}{}

	err = api.UpdateClient(&client, casted)
	// Would partial state be a better solution here?
	if err != nil {
		// Destroy resource if initialization fails
		d.SetId("")
		return err
	}

	client.SortFields()

	d.SetId(id)

	err = d.Set("name", client.Name)
	if err != nil {
		return err
	}
	err = d.Set("scopes", client.Scopes)
	if err != nil {
		return err
	}
	err = d.Set("grants", client.Grants)
	if err != nil {
		return err
	}

	// Set state of nested resources
	urls := append(client.RedirectURLs, client.CorsURLs...)
	urls = append(urls, client.PostLogoutURLs...)
	urlMaps := new([]map[string]interface{})
	for _, url := range urls {
		*urlMaps = append(*urlMaps, url.AsMap())
	}
	err = d.Set("url", urlMaps)
	if err != nil {
		return err
	}

	claimMaps := new([]map[string]interface{})
	for _, claim := range client.Claims {
		*claimMaps = append(*claimMaps, claim.AsMap())
	}
	err = d.Set("claim", claimMaps)
	if err != nil {
		return err
	}

	secretMaps := new([]map[string]interface{})
	for _, sec := range client.Secrets {
		*secretMaps = append(*secretMaps, sec.AsMap())
	}
	err = d.Set("secret", secretMaps)
	if err != nil {
		return err
	}

	return identityClientRead(d, m)
}

func identityClientRead(d *schema.ResourceData, m interface{}) error {
	if m == nil {
		return fmt.Errorf("Error configuring provider")
	}

	client, err := api.ReadClient(d.Id(), m.(map[string]string))
	if err != nil {
		return err
	}

	client.SortFields()
	log.Printf("! Client returned by API read func: %+v", client)

	// Set top level fields
	err = d.Set("name", client.Name)
	if err != nil {
		return err
	}
	err = d.Set("scopes", client.Scopes)
	if err != nil {
		return err
	}
	err = d.Set("grants", client.Grants)
	if err != nil {
		return err
	}

	// Set nested resource values
	urls := append(client.RedirectURLs, client.CorsURLs...)
	urls = append(urls, client.PostLogoutURLs...)
	urlMaps := new([]map[string]interface{})
	for _, url := range urls {
		*urlMaps = append(*urlMaps, url.AsMap())
	}
	err = d.Set("url", urlMaps)
	if err != nil {
		return err
	}

	claimMaps := new([]map[string]interface{})
	for _, claim := range client.Claims {
		*claimMaps = append(*claimMaps, claim.AsMap())
	}
	err = d.Set("claim", claimMaps)
	if err != nil {
		return err
	}

	// API will not show us actual value for secret, so don't bother reading/setting them

	return nil
}

func identityClientUpdate(d *schema.ResourceData, m interface{}) error {
	// Fields that can be updated:
	// top level properties
	// value fields in urls/claims
	// can also set deleted fields to true to remove a nested resource
	// Can set deleted even for a secret but value of secret is immutable

	// Build client object to pass to API
	client := structs.NewClient(d.Get("name").(string), d.Get("scopes").(string), d.Get("grants").(string))
	client.ID, _ = strconv.ParseFloat(d.Id(), 64)

	urlOld, urlNew := d.GetChange("url")

	oldTemp := urlOld.([]interface{})
	newTemp := urlNew.([]interface{})

	oldMaps := new([]map[string]interface{})
	newMaps := new([]map[string]interface{})

	for _, old := range oldTemp {
		*oldMaps = append(*oldMaps, old.(map[string]interface{}))
	}
	for _, item := range newTemp {
		*newMaps = append(*newMaps, item.(map[string]interface{}))
	}

	oldURLStructs := new([]structs.URL)
	newURLStructs := new([]structs.URL)

	for _, url := range *oldMaps {
		*oldURLStructs = append(*oldURLStructs, structs.URLFromMap(url))
	}
	for _, url := range *newMaps {
		*newURLStructs = append(*newURLStructs, structs.URLFromMap(url))
	}

	// Note: This *does* support creating new URLs and claims because any new blocks will get put into the
	// update payload and be created by the API

	// Search for any URLs removed from config
	toDelete := new([]structs.URL)
	toUpdate := new([]structs.URL)
	for _, old := range *oldURLStructs {
		found := false
		for _, item := range *newURLStructs {
			if item.ID == old.ID {
				found = true
			}
		}
		if !found {
			old.Deleted = true
			*toDelete = append(*toDelete, old)
		}
	}
	*toUpdate = *newURLStructs
	urls := append(*toDelete, *toUpdate...)

	// Assign each URL to the proper slot in the client object
	for _, url := range urls {
		if url.Type == "redirectUri" {
			client.RedirectURLs = append(client.RedirectURLs, url)
		} else if url.Type == "corsUri" {
			client.CorsURLs = append(client.CorsURLs, url)
		} else {
			client.PostLogoutURLs = append(client.PostLogoutURLs, url)
		}
	}

	// Do the same as above for claims
	claimOld, claimNew := d.GetChange("claim")

	oldTempClaim := claimOld.([]interface{})
	newTempClaim := claimNew.([]interface{})

	oldMapsClaim := new([]map[string]interface{})
	newMapsClaim := new([]map[string]interface{})

	for _, old := range oldTempClaim {
		*oldMapsClaim = append(*oldMapsClaim, old.(map[string]interface{}))
	}
	for _, item := range newTempClaim {
		*newMapsClaim = append(*newMapsClaim, item.(map[string]interface{}))
	}

	log.Printf("! Old claims: %+v", oldMapsClaim)
	log.Printf("! New claims: %+v", newMapsClaim)

	oldClaimStructs := new([]structs.Claim)
	newClaimStructs := new([]structs.Claim)

	for _, claim := range *oldMapsClaim {
		*oldClaimStructs = append(*oldClaimStructs, structs.ClaimFromMap(claim))
	}
	for _, claim := range *newMapsClaim {
		*newClaimStructs = append(*newClaimStructs, structs.ClaimFromMap(claim))
	}

	log.Printf("! Old claims as structs: %+v", oldClaimStructs)
	log.Printf("! New claims as structs: %+v", newClaimStructs)

	// Find removed claims
	toDeleteClaim := new([]structs.Claim)
	toUpdateClaim := new([]structs.Claim)
	for _, old := range *oldClaimStructs {
		found := false
		for _, item := range *newClaimStructs {
			if item.ID == old.ID {
				found = true
			}
		}
		if !found {
			old.Deleted = true
			*toDeleteClaim = append(*toDeleteClaim, old)
		}
	}
	*toUpdateClaim = *newClaimStructs

	log.Printf("! Claims to delete: %+v", toDeleteClaim)
	log.Printf("! Claims to update: %+v", toUpdateClaim)

	claims := append(*toDeleteClaim, *toUpdateClaim...)
	client.Claims = claims

	// Handle secrets
	secOld, secNew := d.GetChange("secret")
	oldSecTemp := secOld.([]interface{})
	newSecTemp := secNew.([]interface{})

	oldSec := new([]structs.Secret)
	newSec := new([]structs.Secret)

	for _, old := range oldSecTemp {
		*oldSec = append(*oldSec, structs.SecretFromMap(old.(map[string]interface{})))
	}
	for _, curr := range newSecTemp {
		*newSec = append(*newSec, structs.SecretFromMap(curr.(map[string]interface{})))
	}

	log.Printf("! Old secrets: %+v", oldSec)
	log.Printf("! New secrets: %+v", newSec)

	toDeleteSecret := new([]structs.Secret)
	toCreateSecret := new([]structs.Secret)
	// Find secrets missing from config or secrets with deleted set
	// Also look for any secret with ID 0 - those are new ones to create
	for _, old := range *oldSec {
		found := false
		log.Printf("! Old secret: %+v", old)
		for _, curr := range *newSec {
			log.Printf("! Current secret: %+v", curr)
			if old.ID == curr.ID {
				found = true
			}
			if curr.Deleted {
				*toDeleteSecret = append(*toDeleteSecret, curr)
			} else if curr.ID == 0 {
				*toCreateSecret = append(*toCreateSecret, curr)
			}
		}
		if !found {
			old.Deleted = true
			*toDeleteSecret = append(*toDeleteSecret, old)
		}
	}
	// Edge case when there are no old secrets
	if len(*oldSec) == 0 {
		toCreateSecret = newSec
	}

	log.Printf("! Secrets to delete: %+v", toDeleteSecret)
	log.Printf("! Secrets to create: %v", toCreateSecret)

	secrets := append(*toDeleteSecret, *toCreateSecret...)
	client.Secrets = secrets
	log.Printf("! secrets appended together: %+v", secrets)

	// Ensure no URL types or claims are gone completely
	if len(client.RedirectURLs) == 0 || len(client.CorsURLs) == 0 || len(client.PostLogoutURLs) == 0 || len(client.Claims) == 0 {
		return fmt.Errorf("there must be at least one of each URL type and a claim")
	}

	log.Printf("! Calling update with payload %+v", client)
	err := api.UpdateClient(&client, m.(map[string]string))
	if err != nil {
		return err
	}

	return identityClientRead(d, m)
}

func identityClientDelete(d *schema.ResourceData, m interface{}) error {
	if m == nil {
		return fmt.Errorf("error configuring provider")
	}
	casted := m.(map[string]string)

	exists, err := api.ClientExists(d.Id(), casted)
	if err != nil {
		return err
	}
	if !exists {
		log.Printf("! Client has already been deleted")
		return nil
	}

	return api.DeleteClient(d.Id(), casted)
}

