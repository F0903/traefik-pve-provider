package proxmox

import (
	"encoding/json"
	"strings"
)

type Node struct {
	Name   string `json:"node"`
	Status string `json:"status,omitempty"`
}

type Resource struct {
	VMID   int    `json:"vmid"`
	Node   string `json:"node,omitempty"`
	Type   string `json:"type,omitempty"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status,omitempty"`
	Tags   string `json:"tags,omitempty"`
}

type GuestConfig struct {
	Name        string `json:"name,omitempty"`
	Hostname    string `json:"hostname,omitempty"`
	Description string `json:"description,omitempty"`
	Tags        string `json:"tags,omitempty"`
	IPConfigs   map[string]string
}

func (c *GuestConfig) UnmarshalJSON(data []byte) error {
	type guestConfig GuestConfig

	var decoded guestConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*c = GuestConfig(decoded)

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for key, value := range raw {
		if !strings.HasPrefix(key, "ipconfig") {
			continue
		}
		var config string
		if err := json.Unmarshal(value, &config); err != nil || strings.TrimSpace(config) == "" {
			continue
		}
		if c.IPConfigs == nil {
			c.IPConfigs = make(map[string]string)
		}
		c.IPConfigs[key] = config
	}
	return nil
}

type GuestAgentInterfaces struct {
	Result []NetworkInterface `json:"result"`
}

type NetworkInterface struct {
	Name            string      `json:"name,omitempty"`
	HardwareAddress string      `json:"hardware-address,omitempty"`
	HWAddr          string      `json:"hwaddr,omitempty"`
	Inet            string      `json:"inet,omitempty"`
	Inet6           string      `json:"inet6,omitempty"`
	IPAddresses     []IPAddress `json:"ip-addresses,omitempty"`
}

type IPAddress struct {
	Address string      `json:"ip-address,omitempty"`
	Type    string      `json:"ip-address-type,omitempty"`
	Prefix  json.Number `json:"prefix,omitempty"`
}
