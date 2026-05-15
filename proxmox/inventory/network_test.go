package inventory

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
	})

	if len(ips) != 3 {
		t.Fatalf("ips = %#v", ips)
	}
	if ips[0].Address != "10.0.0.10" || ips[1].Address != "10.0.0.20" || ips[2].Address != "fd00::10" {
		t.Fatalf("ips = %#v", ips)
	}
}
