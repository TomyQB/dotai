package selfupdate

import "testing"

// TestCanUpgrade exercises the predicate for every combination of
// Availability fields.
func TestCanUpgrade(t *testing.T) {
	cases := []struct {
		a    Availability
		want bool
	}{
		{Availability{BrewOnPath: false, DotaiManaged: false}, false},
		{Availability{BrewOnPath: true, DotaiManaged: false}, false},
		{Availability{BrewOnPath: false, DotaiManaged: true}, false},
		{Availability{BrewOnPath: true, DotaiManaged: true}, true},
	}
	for _, tc := range cases {
		if got := tc.a.CanUpgrade(); got != tc.want {
			t.Errorf("CanUpgrade(%+v) = %v, want %v", tc.a, got, tc.want)
		}
	}
}

// TestBrewReplacedDotai covers the three meaningful shapes of `brew upgrade`
// output: no-op ("already installed"), actual upgrade, and an empty/garbled
// stream that should not be treated as success.
func TestBrewReplacedDotai(t *testing.T) {
	cases := []struct {
		name string
		out  string
		want bool
	}{
		{"already installed", "Warning: dotai 0.3.2 already installed\n", false},
		{"upgrading", "==> Upgrading TomyQB/tap/dotai\n==> Pouring dotai--0.3.3.bottle.tar.gz\n", true},
		{"pouring only", "==> Pouring dotai--0.3.3.bottle.tar.gz\n", true},
		{"case insensitive", "==> UPGRADING TomyQB/tap/dotai\n", true},
		{"empty", "", false},
		{"unrelated noise", "Error: something went wrong\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := brewReplacedDotai(tc.out); got != tc.want {
				t.Errorf("brewReplacedDotai(%q) = %v, want %v", tc.out, got, tc.want)
			}
		})
	}
}
