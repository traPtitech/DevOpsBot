package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"sync"

	"github.com/dghubble/sling"
	"github.com/traPtitech/DevOpsBot/pkg/config"
	"golang.org/x/sync/errgroup"
)

type hostsCommand struct {
}

type serversResponse struct {
	Servers []struct {
		ID    string `json:"id"`
		Links []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"links"`
		Name string `json:"name"`
	} `json:"servers"`
}

type serverResponse struct {
	Server struct {
		ID               string `json:"id"`
		Status           string `json:"status"`
		OSEXTSRVATTRHost string `json:"OS-EXT-SRV-ATTR:host"`
		Metadata         struct {
			InstanceNameTag string `json:"instance_name_tag"`
		} `json:"metadata"`
	} `json:"server"`
}

type resultData struct {
	ID   string
	Host string
	Name string
}

func (sc *hostsCommand) Execute(args []string) error {
	token, err := getConohaAPIToken()
	if err != nil {
		return fmt.Errorf("failed to get conoha api token: %w", err)
	}

	req, err := sling.New().
		Base(config.C.Servers.Conoha.Origin.Compute).
		Get(fmt.Sprintf("v2/%s/servers", config.C.Servers.Conoha.TenantID)).
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid status code: %s (expected: 200)", resp.Status)
	}

	var response serversResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	servers := []resultData{}

	eg := errgroup.Group{}
	mu := sync.Mutex{}
	for _, server := range response.Servers {
		eg.Go(func() error {
			req, err := sling.New().
				Base(config.C.Servers.Conoha.Origin.Compute).
				Get(fmt.Sprintf("v2/%s/servers/%s", config.C.Servers.Conoha.TenantID, server.ID)).
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

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("invalid status code: %s (expected: 200)", resp.Status)
			}

			var response serverResponse
			if err := json.Unmarshal(respBody, &response); err != nil {
				return fmt.Errorf("failed to unmarshal response body: %w", err)
			}

			mu.Lock()
			servers = append(servers, resultData{
				ID:   response.Server.ID,
				Host: response.Server.OSEXTSRVATTRHost,
				Name: response.Server.Metadata.InstanceNameTag,
			})
			mu.Unlock()

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return fmt.Errorf("error in goroutines: %w", err)
	}

	sort.Slice(servers, func(i, j int) bool {
		return servers[i].Name < servers[j].Name
	})

	for _, server := range servers {
		log.Printf("%s: %s\n", server.Name, server.Host)
	}

	return nil
}
