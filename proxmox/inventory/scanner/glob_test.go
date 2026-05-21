package scanner

import "testing"

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern string
		value   string
		want    bool
	}{
		{pattern: "eth0", value: "eth0", want: true},
		{pattern: "eth0", value: "eth1", want: false},
		{pattern: "eth*", value: "eth0", want: true},
		{pattern: "eth*", value: "eth10", want: true},
		{pattern: "eth?", value: "eth0", want: true},
		{pattern: "eth?", value: "eth10", want: false},
		{pattern: "*0", value: "pterodactyl0", want: true},
		{pattern: "e*h?", value: "eth0", want: true},
		{pattern: "", value: "eth0", want: false},
	}

	for _, test := range tests {
		got := matchGlob(test.pattern, test.value)
		if got != test.want {
			t.Fatalf("matchGlob(%q, %q) = %t, want %t", test.pattern, test.value, got, test.want)
		}
	}
}
