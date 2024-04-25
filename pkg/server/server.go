package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dghubble/sling"
	"github.com/samber/lo"
	"io"
	"log"
	"net/http"

	"github.com/traPtitech/DevOpsBot/pkg/config"
)

type ServersCommand struct {
	sub *subCommand
}

func Compile() (*ServersCommand, error) {
	cmd := &ServersCommand{}

	s := &subCommand{
		Commands: make(map[string]command),
	}
	s.Commands["restart"] = &restartCommand{s}
	cmd.sub = s

	return cmd, nil
}

func (sc *ServersCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("invalid arguments, expected server id")
	}

	// args == [server id] restart [SOFT|HARD]
	serverID := args[0]
	return sc.sub.Execute(serverID, args[1:])
}

type subCommand struct {
	Commands map[string]command
}

func (i *subCommand) Execute(serverID string, args []string) error {
	if len(args) < 1 {
		return errors.New("invalid arguments, expected server action verb (supported: restart)")
	}

	// args == restart [SOFT|HARD]
	verb := args[0]
	c, ok := i.Commands[verb]
	if !ok {
		return fmt.Errorf("unknown command: `%s`", verb)
	}
	return c.Execute(serverID, args[1:])
}

type command interface {
	Execute(serverID string, args []string) error
}

type restartCommand struct {
	server *subCommand
}

type m map[string]any

func (sc *restartCommand) Execute(serverID string, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("invalid arguments, expected restart type (SOFT or HARD)")
	}

	// args == [SOFT|HARD]
	restartType := args[0]

	if !lo.Contains([]string{"SOFT", "HARD"}, restartType) {
		return fmt.Errorf("unknown restart type: %s", restartType)
	}

	token, err := getConohaAPIToken()
	if err != nil {
		return fmt.Errorf("failed to get conoha api token: %w", err)
	}

	req, err := sling.New().
		Base(config.C.Servers.Conoha.Origin.Compute).
		Post(fmt.Sprintf("v2/%s/servers/%s/action", config.C.Servers.Conoha.TenantID, serverID)).
		BodyJSON(m{"reboot": m{"type": args[0]}}).
		Set("Accept", "application/json").
		Set("X-Auth-Token", token).
		Request()
	if err != nil {
		return fmt.Errorf("failed to create restart request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)

	if err != nil {
		return fmt.Errorf("failed to post restart request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	logStr := fmt.Sprintf(`Request
- URL: %s
- RestartType: %s

Response
- Header: %+v
- Body: %s
- Status: %s (Expected: 202)
`, req.URL.String(), restartType, resp.Header, string(respBody), resp.Status)
	log.Println(logStr)

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("incorrect status code: %s", resp.Status)
	}

	return nil
}

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
