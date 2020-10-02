/*
Crucible
Copyright 2020 Carnegie Mellon University.
NO WARRANTY. THIS CARNEGIE MELLON UNIVERSITY AND SOFTWARE ENGINEERING INSTITUTE MATERIAL IS FURNISHED ON AN "AS-IS" BASIS. CARNEGIE MELLON UNIVERSITY MAKES NO WARRANTIES OF ANY KIND, EITHER EXPRESSED OR IMPLIED, AS TO ANY MATTER INCLUDING, BUT NOT LIMITED TO, WARRANTY OF FITNESS FOR PURPOSE OR MERCHANTABILITY, EXCLUSIVITY, OR RESULTS OBTAINED FROM USE OF THE MATERIAL. CARNEGIE MELLON UNIVERSITY DOES NOT MAKE ANY WARRANTY OF ANY KIND WITH RESPECT TO FREEDOM FROM PATENT, TRADEMARK, OR COPYRIGHT INFRINGEMENT.
Released under a MIT (SEI)-style license, please see license.txt or contact permission@sei.cmu.edu for full terms.
[DISTRIBUTION STATEMENT A] This material has been approved for public release and unlimited distribution.  Please see Copyright notice for non-US Government use and distribution.
Carnegie Mellon(R) and CERT(R) are registered in the U.S. Patent and Trademark Office by Carnegie Mellon University.
DM20-0181
*/

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

