// Copyright 2021 Carnegie Mellon University. All Rights Reserved.
// Released under a MIT (SEI)-style license. See LICENSE.md in the project root for license information.
package util

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// GetIdenAuth authenticates with the Identity API. Will probably want to modify this to handle the
// first part in seed-data but this is fine for now
func GetIdenAuth(m map[string]string) (string, error) {
	// Call token endpoint
	apiURL := m["id_token_url"]
	resource := "connect/token"
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", m["client_id"])
	data.Set("client_secret", m["client_secret"])
	data.Set("scope", "identity-api identity-api-privileged")

	u, err := url.ParseRequestURI(apiURL)
	if err != nil {
		return "", err
	}
	u.Path = resource
	urlStr := u.String()

	request, err := http.NewRequest(http.MethodPost, urlStr, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}

	status := response.StatusCode
	if status != http.StatusOK {
		return "", fmt.Errorf("Identity API returned with status %d when getting bearer token", status)
	}

	// Read body of response to find the token
	body := make(map[string]interface{})
	err = json.NewDecoder(response.Body).Decode(&body)
	defer response.Body.Close()

	if err != nil {
		return "", err
	}

	return body["access_token"].(string), nil
}

