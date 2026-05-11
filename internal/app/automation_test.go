package app

import "testing"

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"v0.1.0", "v0.1.0", 0},
		{"v0.1.0", "v0.1.1", -1},
		{"v0.1.1", "v0.1.0", 1},
		{"v0.1.0-beta.1", "v0.1.0-beta.2", -1},
		{"v0.1.0-beta.2", "v0.1.0-beta.10", -1},
		{"v0.1.0-beta.1", "v0.1.0", -1},
		{"v0.1.0", "v0.1.0-beta.1", 1},
		{"dev", "v0.1.0-beta.6", -1},
		{"v0.1.0-beta.6", "dev", 1},
		{"v0.2.0-beta.1", "v0.1.0", 1},
	}
	for _, c := range cases {
		got := compareVersions(c.a, c.b)
		if got != c.want {
			t.Errorf("compareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}
