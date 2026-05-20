package proxmox

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

type Config struct {
	Endpoint           string
	TokenID            string
	Token              string
	Timeout            time.Duration
	InsecureSkipVerify bool
	UserAgent          string
	HTTPClient         *http.Client
}

type Client struct {
	baseURL    url.URL
	tokenID    string
	token      string
	userAgent  string
	httpClient *http.Client
}

type APIError struct {
	Method     string
	Path       string
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	body := strings.TrimSpace(e.Body)
	if len(body) > 300 {
		body = body[:300] + "..."
	}
	if body == "" {
		return fmt.Sprintf("proxmox %s %s failed with status %d", e.Method, e.Path, e.StatusCode)
	}
	return fmt.Sprintf("proxmox %s %s failed with status %d: %s", e.Method, e.Path, e.StatusCode, body)
}

func NewClient(cfg Config) (*Client, error) {
	if cfg.Endpoint == "" {
		return nil, errors.New("endpoint is required")
	}
	if cfg.TokenID == "" {
		return nil, errors.New("token ID is required")
	}
	if cfg.Token == "" {
		return nil, errors.New("token is required")
	}

	baseURL, err := parseBaseURL(cfg.Endpoint)
	if err != nil {
		return nil, err
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		timeout := cfg.Timeout
		if timeout == 0 {
			timeout = defaultTimeout
		}

		httpClient = &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.InsecureSkipVerify},
			},
		}
	}

	userAgent := cfg.UserAgent
	if userAgent == "" {
		userAgent = "traefik-pve-provider"
	}

	return &Client{
		baseURL:    *baseURL,
		tokenID:    cfg.TokenID,
		token:      cfg.Token,
		userAgent:  userAgent,
		httpClient: httpClient,
	}, nil
}

func (c *Client) Nodes(ctx context.Context) ([]Node, error) {
	var nodes []Node
	err := c.get(ctx, "/nodes", &nodes)
	return nodes, err
}

func (c *Client) ClusterResources(ctx context.Context) ([]Resource, error) {
	var resources []Resource
	err := c.get(ctx, "/cluster/resources?type=vm", &resources)
	return resources, err
}

func (c *Client) VirtualMachines(ctx context.Context, node string) ([]Resource, error) {
	var resources []Resource
	err := c.get(ctx, "/nodes/"+pathEscape(node)+"/qemu", &resources)
	return resources, err
}

func (c *Client) Containers(ctx context.Context, node string) ([]Resource, error) {
	var resources []Resource
	err := c.get(ctx, "/nodes/"+pathEscape(node)+"/lxc", &resources)
	return resources, err
}

func (c *Client) VMConfig(ctx context.Context, node string, vmid int) (GuestConfig, error) {
	var cfg GuestConfig
	data, err := c.getData(ctx, "/nodes/"+pathEscape(node)+"/qemu/"+strconv.Itoa(vmid)+"/config")
	if err != nil || len(data) == 0 {
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("decode response data: %w", err)
	}
	cfg.IPConfigs = ipConfigsFromGuestConfigData(data)
	return cfg, nil
}

func (c *Client) ContainerConfig(ctx context.Context, node string, vmid int) (GuestConfig, error) {
	var cfg GuestConfig
	err := c.get(ctx, "/nodes/"+pathEscape(node)+"/lxc/"+strconv.Itoa(vmid)+"/config", &cfg)
	return cfg, err
}

func (c *Client) VMNetworkInterfaces(ctx context.Context, node string, vmid int) (GuestAgentInterfaces, error) {
	var interfaces GuestAgentInterfaces
	err := c.get(ctx, "/nodes/"+pathEscape(node)+"/qemu/"+strconv.Itoa(vmid)+"/agent/network-get-interfaces", &interfaces)
	return interfaces, err
}

func (c *Client) ContainerInterfaces(ctx context.Context, node string, vmid int) ([]NetworkInterface, error) {
	var interfaces []NetworkInterface
	err := c.get(ctx, "/nodes/"+pathEscape(node)+"/lxc/"+strconv.Itoa(vmid)+"/interfaces", &interfaces)
	return interfaces, err
}

func (c *Client) get(ctx context.Context, apiPath string, into any) error {
	return c.do(ctx, http.MethodGet, apiPath, nil, into)
}

func (c *Client) getData(ctx context.Context, apiPath string) (json.RawMessage, error) {
	var data json.RawMessage
	if err := c.get(ctx, apiPath, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (c *Client) do(ctx context.Context, method, apiPath string, body any, into any) error {
	requestURL := c.urlFor(apiPath)

	var requestBody io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
		requestBody = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, requestURL, requestBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "PVEAPIToken="+c.tokenID+"="+c.token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &APIError{
			Method:     method,
			Path:       apiPath,
			StatusCode: resp.StatusCode,
			Body:       string(responseBody),
		}
	}

	if into == nil {
		return nil
	}

	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(responseBody, &envelope); err != nil {
		return fmt.Errorf("decode response envelope: %w", err)
	}
	if len(envelope.Data) == 0 || string(envelope.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(envelope.Data, into); err != nil {
		return fmt.Errorf("decode response data: %w", err)
	}

	return nil
}

func (c *Client) urlFor(apiPath string) string {
	u := c.baseURL
	if path, query, ok := strings.Cut(apiPath, "?"); ok {
		u.Path = joinURLPath(c.baseURL.Path, path)
		u.RawQuery = query
		return u.String()
	}

	u.Path = joinURLPath(c.baseURL.Path, apiPath)
	return u.String()
}

func parseBaseURL(endpoint string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err != nil {
		return nil, fmt.Errorf("parse endpoint: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("endpoint must include scheme and host")
	}

	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	if !strings.HasSuffix(parsed.Path, "/api2/json") {
		parsed.Path = strings.TrimRight(parsed.Path, "/") + "/api2/json"
	}
	return parsed, nil
}

func joinURLPath(basePath, apiPath string) string {
	basePath = strings.TrimRight(basePath, "/")
	apiPath = "/" + strings.TrimLeft(apiPath, "/")
	return basePath + apiPath
}

func pathEscape(segment string) string {
	return url.PathEscape(segment)
}

func ipConfigsFromGuestConfigData(data json.RawMessage) map[string]string {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	configs := make(map[string]string)
	for key, value := range raw {
		if !strings.HasPrefix(key, "ipconfig") {
			continue
		}
		var config string
		if err := json.Unmarshal(value, &config); err != nil || strings.TrimSpace(config) == "" {
			continue
		}
		configs[key] = config
	}
	if len(configs) == 0 {
		return nil
	}
	return configs
}
