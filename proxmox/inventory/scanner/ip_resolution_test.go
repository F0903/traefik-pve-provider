package scanner

import (
	"testing"

	"github.com/F0903/traefik-pve-provider/proxmox"
)

func TestIPsFromInterfacesSortsStableRoutableIPs(t *testing.T) {
	ips := ipsFromInterfaces([]proxmox.NetworkInterface{
		{
			Name: "eth0",
			IPAddresses: []proxmox.IPAddress{
				{Address: "10.0.0.20", Type: "ipv4"},
				{Address: "127.0.0.1", Type: "ipv4"},
				{Address: "10.0.0.10", Type: "ipv4"},
			},
			Inet6: "fd00::10/64",
		},
	}, IPModeIPv4IPv6, nil)

	if len(ips) != 3 {
		t.Fatalf("ips = %#v", ips)
	}
	if ips[0].Address != "10.0.0.10" || ips[1].Address != "10.0.0.20" || ips[2].Address != "fd00::10" {
		t.Fatalf("ips = %#v", ips)
	}
}

func TestIPsFromInterfacesDefaultsToIPv4(t *testing.T) {
	ips := ipsFromInterfaces([]proxmox.NetworkInterface{
		{
			Name: "eth0",
			IPAddresses: []proxmox.IPAddress{
				{Address: "10.0.0.10", Type: "ipv4"},
				{Address: "fd00::10", Type: "ipv6"},
			},
		},
	}, "", nil)

	if len(ips) != 1 || ips[0].Address != "10.0.0.10" {
		t.Fatalf("ips = %#v", ips)
	}
}

func TestIPsFromInterfacesHonorsIPv6Mode(t *testing.T) {
	ips := ipsFromInterfaces([]proxmox.NetworkInterface{
		{
			Name: "eth0",
			IPAddresses: []proxmox.IPAddress{
				{Address: "10.0.0.10", Type: "ipv4"},
				{Address: "fd00::10", Type: "ipv6"},
			},
		},
	}, IPModeIPv6, nil)

	if len(ips) != 1 || ips[0].Address != "fd00::10" {
		t.Fatalf("ips = %#v", ips)
	}
}

func TestIPsFromInterfacesFiltersByDefaultInterfacePatterns(t *testing.T) {
	ips := ipsFromInterfaces([]proxmox.NetworkInterface{
		{
			Name: "docker0",
			IPAddresses: []proxmox.IPAddress{
				{Address: "172.17.0.1", Type: "ipv4"},
			},
		},
		{
			Name: "br-1234",
			IPAddresses: []proxmox.IPAddress{
				{Address: "172.18.0.1", Type: "ipv4"},
			},
		},
		{
			Name: "eth0",
			IPAddresses: []proxmox.IPAddress{
				{Address: "10.0.0.10", Type: "ipv4"},
			},
		},
		{
			Name: "enp6s0",
			IPAddresses: []proxmox.IPAddress{
				{Address: "10.0.0.20", Type: "ipv4"},
			},
		},
		{
			Name: "ens18",
			IPAddresses: []proxmox.IPAddress{
				{Address: "10.0.0.18", Type: "ipv4"},
			},
		},
	}, IPModeIPv4, DefaultInterfacePatterns())

	if len(ips) != 3 || ips[0].Address != "10.0.0.10" || ips[1].Address != "10.0.0.18" || ips[2].Address != "10.0.0.20" {
		t.Fatalf("ips = %#v", ips)
	}
}

func TestIPsFromInterfacesHonorsInterfacePatterns(t *testing.T) {
	ips := ipsFromInterfaces([]proxmox.NetworkInterface{
		{
			Name: "eth0",
			IPAddresses: []proxmox.IPAddress{
				{Address: "10.0.0.10", Type: "ipv4"},
			},
		},
		{
			Name: "ens18",
			IPAddresses: []proxmox.IPAddress{
				{Address: "10.0.0.18", Type: "ipv4"},
			},
		},
		{
			Name: "wg0",
			IPAddresses: []proxmox.IPAddress{
				{Address: "10.20.0.1", Type: "ipv4"},
			},
		},
	}, IPModeIPv4, []string{"eth*", "ens18"})

	if len(ips) != 2 {
		t.Fatalf("ips = %#v", ips)
	}
	if ips[0].Address != "10.0.0.10" || ips[1].Address != "10.0.0.18" {
		t.Fatalf("ips = %#v", ips)
	}
}

func TestIPsFromInterfacesAllowsAnyInterfaceWhenWildcardMatches(t *testing.T) {
	ips := ipsFromInterfaces([]proxmox.NetworkInterface{
		{
			Name: "docker0",
			IPAddresses: []proxmox.IPAddress{
				{Address: "172.17.0.1", Type: "ipv4"},
			},
		},
		{
			Name: "eth0",
			IPAddresses: []proxmox.IPAddress{
				{Address: "10.0.0.10", Type: "ipv4"},
			},
		},
	}, IPModeIPv4, []string{"*"})

	if len(ips) != 2 {
		t.Fatalf("ips = %#v", ips)
	}
}

func TestIPsFromInterfacesReturnsEmptyWhenNoInterfacePatternMatches(t *testing.T) {
	ips := ipsFromInterfaces([]proxmox.NetworkInterface{
		{
			Name: "wg0",
			IPAddresses: []proxmox.IPAddress{
				{Address: "10.20.0.1", Type: "ipv4"},
			},
		},
	}, IPModeIPv4, []string{"eth*"})

	if len(ips) != 0 {
		t.Fatalf("ips = %#v", ips)
	}
}

func TestIPsFromGuestConfigParsesStaticIPConfigs(t *testing.T) {
	ips := ipsFromGuestConfig(map[string]string{
		"ipconfig1": "ip=dhcp,ip6=fd00::50/64,gw6=fd00::1",
		"ipconfig0": "ip=10.0.0.50/24,gw=10.0.0.1",
	}, IPModeIPv4IPv6)

	if len(ips) != 2 {
		t.Fatalf("ips = %#v", ips)
	}
	if ips[0].Address != "10.0.0.50" || ips[0].Prefix != 24 || ips[0].Interface != "ipconfig0" {
		t.Fatalf("first ip = %#v", ips[0])
	}
	if ips[1].Address != "fd00::50" || ips[1].Prefix != 64 || ips[1].Interface != "ipconfig1" {
		t.Fatalf("second ip = %#v", ips[1])
	}
}

func TestIPsFromGuestConfigHonorsIPMode(t *testing.T) {
	ips := ipsFromGuestConfig(map[string]string{
		"ipconfig0": "ip=10.0.0.50/24,ip6=fd00::50/64",
	}, IPModeIPv4)

	if len(ips) != 1 || ips[0].Address != "10.0.0.50" {
		t.Fatalf("ips = %#v", ips)
	}
}
