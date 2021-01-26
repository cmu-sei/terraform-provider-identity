// Copyright 2021 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"identity_provider/internal/structs"
	"identity_provider/internal/util"
	"log"
	"net/http"
	"sort"
	"strconv"
)

// CreateClient creates a client with the given configuration
//
// param client the client to create
//
// param m: A map containing configuration info for the provider
//
// returns the id of the client and nil on success or some error on failure
func CreateClient(client *structs.Client, m map[string]string) (string, error) {
	auth, err := util.GetIdenAuth(m)
	if err != nil {
		return "", err
	}

	asJSON, err := json.Marshal(client)
	if err != nil {
		return "", err
	}

	url := m["id_api_url"] + "client"
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(asJSON))
	if err != nil {
		return "", err
	}
	request.Header.Add("Authorization", "Bearer "+auth)
	request.Header.Set("Content-Type", "application/json")
	APIClient := &http.Client{}

	response, err := APIClient.Do(request)
	if err != nil {
		return "", err
	}

	status := response.StatusCode
	if status != http.StatusOK {
		return "", fmt.Errorf("Identity API returned with status code %d when creating client", status)
	}

	// Get id of client
	body := make(map[string]interface{})
	err = json.NewDecoder(response.Body).Decode(&body)
	defer response.Body.Close()

	id := body["id"].(float64)
	log.Printf("! Client id: %v", id)
	client.ID = id
	return strconv.FormatFloat(id, 'f', -1, 64), nil
}

// UpdateClient updates an existing client. This function is also used to initialize a new client with scopes and
// other nested fields.
//
// param client the client to create
//
// param m: A map containing configuration info for the provider
//
// Returns the response to the api call and nil on success or some error on failure
func UpdateClient(client *structs.Client, m map[string]string) error {
	auth, err := util.GetIdenAuth(m)
	if err != nil {
		return err
	}

	log.Printf("! Client secrets: %+v", client.Secrets)

	log.Printf("! Client = %+v", client)
	asJSON, err := json.Marshal(client)
	if err != nil {
		return err
	}

	url := m["id_api_url"] + "client"
	request, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(asJSON))
	if err != nil {
		return err
	}

	request.Header.Add("Authorization", "Bearer "+auth)
	request.Header.Set("Content-Type", "application/json")
	APIClient := &http.Client{}

	response, err := APIClient.Do(request)
	if err != nil {
		return err
	}

	status := response.StatusCode
	if status != http.StatusOK {
		return fmt.Errorf("Identity API returned with status code %d when updating client", status)
	}

	// We need to grab the newly created secrets - ones without deleted set - to call addSecrets function
	secrets := new([]structs.Secret)
	for _, secret := range client.Secrets {
		if !secret.Deleted {
			*secrets = append(*secrets, secret)
		}
	}

	// Setting secrets requires a distinct API call
	if len(*secrets) > 0 {
		log.Printf("! Calling addSecrets")
		err = addSecrets(secrets, auth, strconv.Itoa(int(client.ID)), m["id_api_url"])
		if err != nil {
			return err
		}
		(*client).Secrets = *secrets
	}

	return readNestedIDs(client, response)
}

// ReadClient reads the state of a given client
//
// param id the of the client to consider
//
// param m: A map containing configuration info for the provider
//
// returns a client struct and an optional error value
func ReadClient(id string, m map[string]string) (*structs.Client, error) {
	auth, err := util.GetIdenAuth(m)
	if err != nil {
		return nil, err
	}

	// Call API and get response with client state
	url := m["id_api_url"] + "client/" + id
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Add("Authorization", "Bearer "+auth)
	APIClient := &http.Client{}

	response, err := APIClient.Do(request)
	if err != nil {
		return nil, err
	}

	status := response.StatusCode
	if status != http.StatusOK {
		return nil, fmt.Errorf("Identity API returned with status code %d when reading client", status)
	}

	// Read response
	body := make(map[string]interface{})
	err = json.NewDecoder(response.Body).Decode(&body)
	defer response.Body.Close()

	// Get top level fields
	client := &structs.Client{
		ID:          body["id"].(float64),
		Name:        body["name"].(string),
		DisplayName: body["displayName"].(string),
		Scopes:      body["scopes"].(string),
		Grants:      body["grants"].(string),
	}

	// TODO this can probably be refactored and combined with the function to read nested IDs

	// Grab redirect URLs
	redirectsGeneric := body["redirectUrls"].([]interface{})
	redirects := new([]map[string]interface{})
	for _, url := range redirectsGeneric {
		*redirects = append(*redirects, url.(map[string]interface{}))
	}
	sort.Slice(*redirects, func(i, j int) bool {
		return (*redirects)[i]["value"].(string) < (*redirects)[j]["value"].(string)
	})
	redirectStructs := new([]structs.URL)
	for _, url := range *redirects {
		*redirectStructs = append(*redirectStructs, structs.URLFromMap(url))
	}
	client.RedirectURLs = *redirectStructs

	// Grab cors URLs
	corsGeneric := body["corsUrls"].([]interface{})
	cors := new([]map[string]interface{})
	for _, url := range corsGeneric {
		*cors = append(*cors, url.(map[string]interface{}))
	}
	sort.Slice(*cors, func(i, j int) bool {
		return (*cors)[i]["value"].(string) < (*cors)[j]["value"].(string)
	})
	corsStructs := new([]structs.URL)
	for _, url := range *cors {
		*corsStructs = append(*corsStructs, structs.URLFromMap(url))
	}
	client.CorsURLs = *corsStructs

	// Grab postlogout URLs
	logoutsGeneric := body["postLogoutUrls"].([]interface{})
	logouts := new([]map[string]interface{})
	for _, url := range logoutsGeneric {
		*logouts = append(*logouts, url.(map[string]interface{}))
	}
	sort.Slice(*logouts, func(i, j int) bool {
		return (*logouts)[i]["value"].(string) < (*logouts)[j]["value"].(string)
	})
	logoutStructs := new([]structs.URL)
	for _, url := range *logouts {
		*logoutStructs = append(*logoutStructs, structs.URLFromMap(url))
	}
	client.PostLogoutURLs = *logoutStructs

	// Grab claims
	claimsGeneric := body["claims"].([]interface{})
	claims := new([]map[string]interface{})
	for _, url := range claimsGeneric {
		*claims = append(*claims, url.(map[string]interface{}))
	}
	sort.Slice(*claims, func(i, j int) bool {
		return (*claims)[i]["value"].(string) < (*claims)[j]["value"].(string)
	})
	claimStructs := new([]structs.Claim)
	for _, claim := range *claims {
		*claimStructs = append(*claimStructs, structs.ClaimFromMap(claim))
	}
	client.Claims = *claimStructs

	return client, nil
}

// ClientExists returns whether a client with a given id exists
//
// param id the of the client to consider
//
// param m: A map containing configuration info for the provider
//
// Returns whether the client exists and an optional error value
func ClientExists(id string, m map[string]string) (bool, error) {
	auth, err := util.GetIdenAuth(m)
	if err != nil {
		return false, err
	}

	url := m["id_api_url"] + "client/" + id
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	request.Header.Add("Authorization", "Bearer "+auth)
	APIClient := &http.Client{}

	response, err := APIClient.Do(request)
	if err != nil {
		return false, err
	}

	status := response.StatusCode
	if status != http.StatusOK {
		return false, fmt.Errorf("Identity API returned with status code %d when checking if client exists", status)
	}
	// API returns 400 bad request if client does not exist
	return status != http.StatusBadRequest, nil
}

// DeleteClient deletes the specified client.
//
// param id the of the client to consider
//
// param m: A map containing configuration info for the provider
//
// Returns nil on success or some error on failure
func DeleteClient(id string, m map[string]string) error {
	auth, err := util.GetIdenAuth(m)
	if err != nil {
		return err
	}

	url := m["id_api_url"] + "client/" + id
	request, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	request.Header.Add("Authorization", "Bearer "+auth)
	APIClient := &http.Client{}

	log.Printf("! Client delete request: %+v", request)
	response, err := APIClient.Do(request)
	if err != nil {
		return err
	}

	status := response.StatusCode
	if status != http.StatusOK {
		return fmt.Errorf("Identity API returned with status code %d when deleting client", status)
	}
	return nil
}

// Add the specified secrets to the client
func addSecrets(secrets *[]structs.Secret, auth, clientID, baseURL string) error {
	log.Printf("! Adding secrets to client with id %v", clientID)
	log.Printf("! Secrets array: %+v", secrets)

	for i := range *secrets {
		log.Printf("! Adding a secret")

		url := baseURL + "client/" + clientID + "/secret"
		request, err := http.NewRequest(http.MethodPut, url, nil)
		if err != nil {
			return err
		}
		request.Header.Add("Authorization", "Bearer "+auth)
		request.Header.Set("Content-Type", "application/json")
		APIClient := &http.Client{}

		response, err := APIClient.Do(request)
		if err != nil {
			return err
		}

		status := response.StatusCode
		if status != http.StatusOK {
			return fmt.Errorf("Identity API returned with status code %d when adding secret", status)
		}

		// Read secret properties from resp body
		body := make(map[string]interface{})
		err = json.NewDecoder(response.Body).Decode(&body)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		(*secrets)[i].ID = int(body["id"].(float64))
		(*secrets)[i].Value = body["value"].(string)
		(*secrets)[i].Deleted = body["deleted"].(bool)
		log.Printf("! Added secret: %+v", (*secrets)[i])
	}

	return nil
}

// Get the IDs of URLs, secrets, and managers. Set them in the passed client pointer
func readNestedIDs(client *structs.Client, resp *http.Response) error {
	body := make(map[string]interface{})
	err := json.NewDecoder(resp.Body).Decode(&body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Sort struct fields so they can be corresponded to remote state
	client.SortFields()

	// Grab redirect URLs
	redirectsGeneric := body["redirectUrls"].([]interface{})
	redirects := new([]map[string]interface{})
	for _, url := range redirectsGeneric {
		*redirects = append(*redirects, url.(map[string]interface{}))
	}
	sort.Slice(*redirects, func(i, j int) bool {
		return (*redirects)[i]["value"].(string) < (*redirects)[j]["value"].(string)
	})
	for i, url := range *redirects {
		log.Printf("! Redirect URL id: %+v", url["id"])
		(*client).RedirectURLs[i].ID = int(url["id"].(float64))
	}

	// Grab cors URLs
	corsGeneric := body["corsUrls"].([]interface{})
	cors := new([]map[string]interface{})
	for _, url := range corsGeneric {
		*cors = append(*cors, url.(map[string]interface{}))
	}
	sort.Slice(*cors, func(i, j int) bool {
		return (*cors)[i]["value"].(string) < (*cors)[j]["value"].(string)
	})
	for i, url := range *cors {
		(*client).CorsURLs[i].ID = int(url["id"].(float64))
	}

	// Grab postlogout URLs
	logoutsGeneric := body["postLogoutUrls"].([]interface{})
	logouts := new([]map[string]interface{})
	for _, url := range logoutsGeneric {
		*logouts = append(*logouts, url.(map[string]interface{}))
	}
	sort.Slice(*logouts, func(i, j int) bool {
		return (*logouts)[i]["value"].(string) < (*logouts)[j]["value"].(string)
	})
	for i, url := range *logouts {
		(*client).PostLogoutURLs[i].ID = int(url["id"].(float64))
	}

	// Grab claims
	claimsGeneric := body["claims"].([]interface{})
	claims := new([]map[string]interface{})
	for _, url := range claimsGeneric {
		*claims = append(*claims, url.(map[string]interface{}))
	}
	sort.Slice(*claims, func(i, j int) bool {
		return (*claims)[i]["value"].(string) < (*claims)[j]["value"].(string)
	})
	for i, claim := range *claims {
		(*client).Claims[i].ID = int(claim["id"].(float64))
	}

	return nil
}
