package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/dghubble/sling"
	"github.com/traPtitech/DevOpsBot/pkg/config"
)

func getConohaAPIToken() (string, error) {
	type passwordCredentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	type auth struct {
		PasswordCredentials passwordCredentials `json:"passwordCredentials"`
		TenantId            string              `json:"tenantId"`
	}
	requestJson := struct {
		Auth auth `json:"auth"`
	}{
		Auth: auth{
			PasswordCredentials: passwordCredentials{
				Username: config.C.Servers.Conoha.Username,
				Password: config.C.Servers.Conoha.Password,
			},
			TenantId: config.C.Servers.Conoha.TenantID,
		},
	}

	req, err := sling.New().
		Base(config.C.Servers.Conoha.Origin.Identity).
		Post("v2.0/tokens").
		BodyJSON(requestJson).
		Set("Accept", "application/json").
		Request()
	if err != nil {
		return "", fmt.Errorf("failed to create authentication request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to post authentication request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("invalid status code: %s (expected: 200)", resp.Status)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var responseJson struct {
		Access struct {
			Token struct {
				Id string `json:"id"`
			} `json:"token"`
		} `json:"access"`
	}
	err = json.Unmarshal(respBody, &responseJson)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return responseJson.Access.Token.Id, nil
}
