// Copyright 2021 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.
package provider

import (
	"fmt"
	"identity_provider/internal/api"
	"identity_provider/internal/structs"
	"log"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func identityAccount() *schema.Resource {
	return &schema.Resource{
		Create: identityAccountCreate,
		Read:   identityAccountRead,
		Update: identityAccountUpdate,
		Delete: identityAccountDelete,

		Schema: map[string]*schema.Schema{
			"username": {
				Type:     schema.TypeString,
				Required: true,
			},
			"password": {
				Type:     schema.TypeString,
				Required: true,
			},
			"role": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"global_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			// Compute b/c it doesn't make sense for an account to be initialzed as inactive.
			// Will change if someone wants to do that.
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			// Will run into similar issues as the admin team with properties generated implicitly.
			// For now just skip the first three properties
			"property": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						// Storing ID is not necessary. We can change a value given a key,
						// and changing a key creates a new property
						"account_id": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						// Keys cannot change. This "destroys" and recreates the whole account, but since destroying
						// is just deactivating, it achieves the same result as simply appending another property
						"key": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"value": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func identityAccountCreate(d *schema.ResourceData, m interface{}) error {
	log.Printf("! At top of identityAccountCreate")
	log.Printf("! properties: %+v", d.Get("property"))

	if m == nil {
		return fmt.Errorf("Error configuring provider")
	}

	// API wants username to be an array, but that doesn't make sense in tf, so just make it an array of size 1
	acct := &structs.Account{
		Usernames: []string{d.Get("username").(string)},
		Password:  d.Get("password").(string),
		Role:      d.Get("role").(string),
		Status:    d.Get("status").(string),
	}

	casted := m.(map[string]string)
	exists, err := api.CreateAccount(acct, casted)
	if err != nil {
		return err
	}

	email := acct.Usernames[0]

	id, glob, err := api.GetIDs(email, casted)
	if err != nil {
		return err
	}
	if exists {
		err = api.EnableAccount(id, casted)
		if err != nil {
			return err
		}
	}

	// Set role if it is set in config
	err = api.SetRole(id, acct.Role, casted)
	if err != nil {
		return err
	}

	d.SetId(id)

	err = d.Set("global_id", glob)
	if err != nil {
		log.Printf("! Error setting global id in create")
		return err
	}

	err = d.Set("username", acct.Usernames[0])
	if err != nil {
		log.Printf("! Error setting username in create")
		return err
	}

	err = d.Set("password", acct.Password)
	if err != nil {
		log.Printf("! Error setting password in create")
		return err
	}

	err = d.Set("role", acct.Role)
	if err != nil {
		log.Printf("! Error setting role in create")
		return err
	}

	err = d.Set("status", "Enabled")
	if err != nil {
		log.Printf("! Error setting status in create")
		return err
	}

	props := d.Get("property").([]interface{})
	if len(props) > 0 {
		err = createProperties(&props, d, casted)
		if err != nil {
			return err
		}
	}

	return identityAccountRead(d, m)
}

func identityAccountRead(d *schema.ResourceData, m interface{}) error {
	log.Printf("! At top of identityAccountRead")

	if m == nil {
		return fmt.Errorf("Error configuring provider")
	}

	// Ensure account is active
	casted := m.(map[string]string)
	user := d.Get("username").(string)
	exists, err := api.IsActive(user, casted)
	if err != nil {
		return err
	}
	if !exists {
		d.SetId("")
		return nil
	}

	// Read account data (ignoring properties for now)
	acct, err := api.ReadAccount(user, casted)
	if err != nil {
		return err
	}

	d.SetId(acct.ID)

	err = d.Set("global_id", acct.GlobalID)
	if err != nil {
		log.Printf("! Error setting global id in read")
	}

	err = d.Set("username", acct.Usernames[0])
	if err != nil {
		log.Printf("! Error setting username in read")
		return err
	}

	err = d.Set("role", acct.Role)
	if err != nil {
		log.Printf("! Error setting role in read")
		return err
	}

	err = d.Set("status", acct.Status)
	if err != nil {
		log.Printf("! Error setting status in read")
		return err
	}

	// Password cannot be read from remote state, so ignore.
	// Maybe be possible to update pw in tf, but it is immutable for now

	// Read state of properties
	props, err := api.ReadProperties(d.Id(), casted)
	if err != nil {
		return err
	}

	return d.Set("property", props)
}

func identityAccountUpdate(d *schema.ResourceData, m interface{}) error {
	if m == nil {
		return fmt.Errorf("Error configuring provider")
	}
	casted := m.(map[string]string)

	// The only things that can be updated are roles and the value field of properties.
	if d.HasChange("role") {
		role := d.Get("role").(string)
		err := api.SetRole(d.Id(), role, casted)
		if err != nil {
			return err
		}
	}

	if d.HasChange("property") {
		oldGeneric, currGeneric := d.GetChange("property")
		oldList := oldGeneric.([]interface{})
		currList := currGeneric.([]interface{})
		// Properties cannot be deleted, so having less properties than before is an error state
		if len(currList) < len(oldList) {
			return fmt.Errorf("properties cannot be deleted")
		}

		toUpdate := new([]*structs.Property)

		// Check for changes in each value field
		for i, prop := range oldList {
			asMapOld := prop.(map[string]interface{})
			asMapCurr := (currList[i]).(map[string]interface{})
			if asMapOld["value"] != asMapCurr["value"] {
				asStruct := structs.PropertyFromMap(asMapCurr)
				acctID, err := strconv.Atoi(d.Id())
				if err != nil {
					return err
				}
				asStruct.AccountID = acctID
				*toUpdate = append(*toUpdate, asStruct)
			}
		}
		// We call the same endpoint as for creation. Since keys are unchanged, the API will update existing properties
		err := api.AddProperties(toUpdate, casted)
		if err != nil {
			return err
		}
	}

	return identityAccountRead(d, m)
}

// Note that this does not destroy an account, it simply disables it.
// If someone tries to create an account with the same name, the API will not
// error (still returns 200), but will not create the account and return a message saying the
// account is not unique.
func identityAccountDelete(d *schema.ResourceData, m interface{}) error {
	if m == nil {
		return fmt.Errorf("Error configuring provider")
	}

	id := d.Id()
	casted := m.(map[string]string)
	exists, err := api.IsActive(id, casted)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	return api.DisableAccount(id, casted)
}

// Create properties specified in config
func createProperties(props *[]interface{}, d *schema.ResourceData, m map[string]string) error {
	// Get structs for the properties
	propStructs := new([]*structs.Property)
	for _, prop := range *props {
		asMap := prop.(map[string]interface{})
		curr := structs.PropertyFromMap(asMap)
		accID, err := strconv.Atoi(d.Id())
		if err != nil {
			return err
		}
		curr.AccountID = accID
		*propStructs = append(*propStructs, curr)
	}

	// Call API
	err := api.AddProperties(propStructs, m)
	if err != nil {
		return err
	}

	// Sort properties by key - same reasoning as in view
	sort.Slice((*propStructs), func(i, j int) bool {
		return (*propStructs)[i].Key < (*propStructs)[j].Key
	})

	// Set local state
	localMaps := new([]map[string]interface{})
	for _, prop := range *propStructs {
		*localMaps = append(*localMaps, prop.AsMap())
	}

	return d.Set("property", localMaps)
}

