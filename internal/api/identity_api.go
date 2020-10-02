/*
Crucible
Copyright 2020 Carnegie Mellon University.
NO WARRANTY. THIS CARNEGIE MELLON UNIVERSITY AND SOFTWARE ENGINEERING INSTITUTE MATERIAL IS FURNISHED ON AN "AS-IS" BASIS. CARNEGIE MELLON UNIVERSITY MAKES NO WARRANTIES OF ANY KIND, EITHER EXPRESSED OR IMPLIED, AS TO ANY MATTER INCLUDING, BUT NOT LIMITED TO, WARRANTY OF FITNESS FOR PURPOSE OR MERCHANTABILITY, EXCLUSIVITY, OR RESULTS OBTAINED FROM USE OF THE MATERIAL. CARNEGIE MELLON UNIVERSITY DOES NOT MAKE ANY WARRANTY OF ANY KIND WITH RESPECT TO FREEDOM FROM PATENT, TRADEMARK, OR COPYRIGHT INFRINGEMENT.
Released under a MIT (SEI)-style license, please see license.txt or contact permission@sei.cmu.edu for full terms.
[DISTRIBUTION STATEMENT A] This material has been approved for public release and unlimited distribution.  Please see Copyright notice for non-US Government use and distribution.
Carnegie Mellon(R) and CERT(R) are registered in the U.S. Patent and Trademark Office by Carnegie Mellon University.
DM20-0181
*/

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"identity_provider/internal/structs"
	"identity_provider/internal/util"
	"log"
	"net/http"
	"strconv"
)

// CreateAccount creates a new identity account with the given parameters
//
// param acct: A struct containing info on the account to create
//
// param m: A A map containing configuration info for the provider
//
// Returns bool stating if this account is unique and an optional error value
func CreateAccount(acct *structs.Account, m map[string]string) (bool, error) {
	auth, err := util.GetIdenAuth(m)
	if err != nil {
		return true, err
	}

	asJSON, err := json.Marshal(acct)
	if err != nil {
		return true, err
	}

	url := m["id_api_url"] + "account"
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(asJSON))
	if err != nil {
		return true, err
	}
	request.Header.Add("Authorization", "Bearer "+auth)
	request.Header.Set("Content-Type", "application/json")
	client := &http.Client{}

	response, err := client.Do(request)
	if err != nil {
		return true, err
	}

	log.Printf("! Create account url: %v", url)
	log.Printf("! Create account response: %+v", response)

	status := response.StatusCode
	if status != http.StatusOK {
		return true, fmt.Errorf("Identity API returned with status code %d when creating account", status)
	}

	// Check for a message saying this account is not unique. If it's there, just re-enable the account.
	bodyArr := new([]interface{})
	err = json.NewDecoder(response.Body).Decode(bodyArr)
	body := (*bodyArr)[0]
	defer response.Body.Close()

	asMap := body.(map[string]interface{})
	log.Printf("! Resp body from account creation: %+v", asMap)
	if asMap["message"] != nil {
		if asMap["message"].(string) == "AccountNotUnique" {
			return true, nil
		}
	}

	return false, nil
}

// GetIDs returns the Id and globalID of an account, along with an error value
//
// param term the account to consider
//
// param m: A A map containing configuration info for the provider
func GetIDs(term string, m map[string]string) (string, string, error) {
	log.Printf("Getting IDs for account with username %s", term)
	response, err := getAccount(term, m)
	if err != nil {
		return "", "", err
	}

	log.Printf("! response: %+v", response)
	// Read data from response
	body := new([]interface{})
	err = json.NewDecoder(response.Body).Decode(body)
	defer response.Body.Close()

	if len(*body) > 1 {
		return "", "", fmt.Errorf("Error retrieving account IDs. Multiple accounts exist with the term %v", term)
	}

	if len(*body) == 0 {
		return "", "", fmt.Errorf("No accounts found with term %v", term)
	}

	asMap := (*body)[0].(map[string]interface{})

	id := strconv.FormatFloat(asMap["id"].(float64), 'f', -1, 64)
	return id, asMap["globalId"].(string), nil
}

// IsActive returns whether an account is active
//
// param term the username of the account
//
// param m: A A map containing configuration info for the provider
//
// Returns true iff the account is active and an optional error value
func IsActive(term string, m map[string]string) (bool, error) {
	response, err := getAccount(term, m)
	if err != nil {
		return false, err
	}

	// Read data from response
	body := new([]interface{})
	err = json.NewDecoder(response.Body).Decode(body)
	defer response.Body.Close()

	if len(*body) < 1 {
		return false, nil
	}

	asMap := (*body)[0].(map[string]interface{})

	return asMap["status"].(string) == "Enabled", nil
}

// ReadAccount returns a struct representation of a given account.
//
// param term the username of the account
//
// param m: A A map containing configuration info for the provider
//
// Returns an account struct and an optional error
func ReadAccount(term string, m map[string]string) (*structs.Account, error) {
	log.Printf("! Calling read API function")
	response, err := getAccount(term, m)
	if err != nil {
		return nil, err
	}

	log.Printf("! Back in ReadAccount after calling getAccount")
	// Read data from response
	body := new([]interface{})
	err = json.NewDecoder(response.Body).Decode(body)
	defer response.Body.Close()

	asMap := (*body)[0].(map[string]interface{})
	props := asMap["properties"].([]interface{})
	// Email is always the third property
	propMap := props[2].(map[string]interface{})
	user := propMap["value"].(string)

	acct := &structs.Account{
		Usernames: []string{user},
		Role:      asMap["role"].(string),
		Status:    asMap["status"].(string),
		ID:        strconv.FormatFloat(asMap["id"].(float64), 'f', -1, 64),
		GlobalID:  asMap["globalId"].(string),
	}

	log.Printf("! Returning account struct: %+v", acct)
	return acct, nil

}

// DisableAccount sets the status of a given account to inactive.
//
// param id the ID of the account to disable
//
// param m: A A map containing configuration info for the provider
//
// Returns some error on failure or nil on success
func DisableAccount(id string, m map[string]string) error {
	auth, err := util.GetIdenAuth(m)
	if err != nil {
		return err
	}

	url := m["id_api_url"] + "account/" + id + "/state/disabled"

	request, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		return err
	}
	request.Header.Add("Authorization", "Bearer "+auth)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("error disabling account %v", id)
	}
	return nil
}

// EnableAccount sets the status of a given account to active.
//
// param id the ID of the account to enable
//
// param m: A A map containing configuration info for the provider
//
// Returns some error on failure or nil on success
func EnableAccount(id string, m map[string]string) error {
	auth, err := util.GetIdenAuth(m)
	if err != nil {
		return err
	}

	url := m["id_api_url"] + "account/" + id + "/state/enabled"

	request, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		return err
	}
	request.Header.Add("Authorization", "Bearer "+auth)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("error enabling account %v", id)
	}
	return nil
}

// SetRole sets the role of a given account
//
// param id the ID of the account
//
// param role the role to set
//
// param m: A A map containing configuration info for the provider
//
// Returns some error on failure or nil on success
func SetRole(id, role string, m map[string]string) error {
	log.Printf("! At top of SetTole")

	auth, err := util.GetIdenAuth(m)
	if err != nil {
		return err
	}

	url := m["id_api_url"] + "account/" + id + "/role/" + role
	log.Printf("! URL adding role: %v", url)

	request, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		return err
	}
	request.Header.Add("Authorization", "Bearer "+auth)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err
	}

	log.Printf("! Response to setting role: %+v", response)
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("error setting role on account %v", id)
	}
	return nil
}

// AddProperties adds a list of properties to an account
//
// param acctID the ID of the account
//
// param prop the property to add
//
// param m: A A map containing configuration info for the provider
//
// Returns nil on success or some error on failure
func AddProperties(props *[]*structs.Property, m map[string]string) error {
	auth, err := util.GetIdenAuth(m)
	if err != nil {
		return err
	}

	for i, prop := range *props {
		log.Printf("! Adding property: %+v", *prop)
		payload, err := json.Marshal(prop)
		if err != nil {
			return err
		}

		test := new(map[string]interface{})
		json.Unmarshal(payload, test)
		log.Printf("! unmarshaled: %+v", test)

		url := m["id_api_url"] + "account/property"
		request, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(payload))
		if err != nil {
			return err
		}
		request.Header.Add("Authorization", "Bearer "+auth)
		request.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		log.Printf("! Request: %+v", request)

		response, err := client.Do(request)
		if err != nil {
			return err
		}

		status := response.StatusCode
		if status != http.StatusOK {
			return fmt.Errorf("Identity API returned with status code %d when creating property %d", status, i)
		}
	}

	return nil
}

// ReadProperties reads the proprties associated with a given account.
//
// param acct: the id of the account to consider
//
// param m: A A map containing configuration info for the provider
//
// Returns an array of maps representing proprties and nil on success or some error on failure
func ReadProperties(acct string, m map[string]string) (*[]map[string]interface{}, error) {
	log.Printf("! Calling read properties API function")
	response, err := getAccount(acct, m)
	if err != nil {
		return nil, err
	}

	// Read data from response
	body := new([]interface{})
	err = json.NewDecoder(response.Body).Decode(body)
	defer response.Body.Close()

	asMap := (*body)[0].(map[string]interface{})
	props := asMap["properties"].([]interface{})
	// The first 3 properties are automatically set and not managed by terraform
	props = props[3:]

	// Read each relevant property
	ret := new([]map[string]interface{})
	for _, prop := range props {
		asMap := prop.(map[string]interface{})
		*ret = append(*ret, map[string]interface{}{
			"account_id": int(asMap["accountId"].(float64)),
			"key":        asMap["key"].(string),
			"value":      asMap["value"].(string),
		})
	}

	return ret, nil

}

// Call API to get account with the given search term.
func getAccount(term string, m map[string]string) (*http.Response, error) {
	log.Printf("! At top of getAccount")

	auth, err := util.GetIdenAuth(m)
	if err != nil {
		return nil, err
	}

	apiURL := m["id_api_url"] + "accounts" + "?Term=" + term

	request, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		log.Printf("! Error creating request")
		return nil, err
	}

	request.Header.Add("Authorization", "Bearer "+auth)
	request.Header.Add("Content-Type", "text/plain")
	log.Printf("! In get account, request is %+v", request)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		log.Printf("! Error making request")
		return nil, err
	}

	status := response.StatusCode
	if status != http.StatusOK {
		return nil, fmt.Errorf("Error retrieving account. Status code was %d", status)
	}

	log.Printf("! returning from getAccount with response = %+v", response)
	return response, nil
}

