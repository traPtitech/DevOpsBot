package server

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/dghubble/sling"
	"github.com/samber/lo"
	"github.com/traPtitech/DevOpsBot/pkg/config"
)

type restartCommand struct {
}

type m map[string]any

func (sc *restartCommand) Execute(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("invalid arguments, expected server id")
	}

	serverID := args[0]
	args = args[1:]

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
