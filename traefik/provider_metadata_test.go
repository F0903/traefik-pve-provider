package traefik

import "testing"

func TestProviderInterfacePatterns(t *testing.T) {
	patterns := providerInterfacePatterns(map[string]string{
		"pve.interfaces": " eth*, , ens18, eth*, enp* ",
	})

	if len(patterns) != 3 || patterns[0] != "eth*" || patterns[1] != "ens18" || patterns[2] != "enp*" {
		t.Fatalf("interface patterns = %#v", patterns)
	}
}

func TestProviderInterfacePatternsMissing(t *testing.T) {
	patterns := providerInterfacePatterns(map[string]string{
		"traefik.enable": "true",
	})

	if patterns != nil {
		t.Fatalf("interface patterns = %#v, want nil", patterns)
	}
}
