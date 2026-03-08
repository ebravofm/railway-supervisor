package railway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Client struct {
	Token      string
	HTTPClient *http.Client
}

func NewClient(token string) *Client {
	return &Client{
		Token:      token,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) FetchServicesForEnvironment(envID string) ([]string, error) {
	query := `query($id: String!) { environment(id: $id) { serviceInstances { nodes { serviceId } } } }`
	vars := map[string]interface{}{"id": envID}

	respData, err := c.executeGraphQL(query, vars)
	if err != nil {
		return nil, err
	}

	var result struct {
		Data struct {
			Environment struct {
				ServiceInstances struct {
					Nodes []struct {
						ServiceID string `json:"serviceId"`
					} `json:"nodes"`
				} `json:"serviceInstances"`
			} `json:"environment"`
		} `json:"data"`
	}

	if err := json.Unmarshal(respData, &result); err != nil {
		return nil, fmt.Errorf("failed to parse environment response: %v", err)
	}

	var sIDs []string
	for _, instance := range result.Data.Environment.ServiceInstances.Nodes {
		sIDs = append(sIDs, instance.ServiceID)
	}

	return sIDs, nil
}

func (c *Client) ExecuteServiceInstanceUpdate(envID, serviceID string, sleep bool) error {
	query := `mutation($environmentId: String!, $serviceId: String!, $input: ServiceInstanceUpdateInput!) {
		serviceInstanceUpdate(environmentId: $environmentId, serviceId: $serviceId, input: $input)
	}`

	vars := map[string]interface{}{
		"environmentId": envID,
		"serviceId":     serviceID,
		"input": map[string]interface{}{
			"sleepApplication": sleep,
		},
	}

	_, err := c.executeGraphQL(query, vars)
	return err
}

func (c *Client) ExecuteServiceInstanceDeploy(envID, serviceID string) error {
	query := `mutation($environmentId: String!, $serviceId: String!) {
		serviceInstanceDeployV2(environmentId: $environmentId, serviceId: $serviceId)
	}`

	vars := map[string]interface{}{
		"environmentId": envID,
		"serviceId":     serviceID,
	}

	_, err := c.executeGraphQL(query, vars)
	return err
}

func (c *Client) executeGraphQL(query string, variables map[string]interface{}) ([]byte, error) {
	payload := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://backboard.railway.com/graphql/v2", bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var gqlError struct {
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(bodyBytes, &gqlError); err == nil && len(gqlError.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL Exception: %s", gqlError.Errors[0].Message)
	}

	return bodyBytes, nil
}
