package handlers

import "testing"

func TestNormalizeModName(t *testing.T) {
	cases := map[string]string{
		"@Dabs Framework":  "dabsframework",
		"@CF":              "cf",
		"WOC_RP (Realism)": "wocrprealism",
	}
	for in, want := range cases {
		if got := normalizeModName(in); got != want {
			t.Fatalf("normalizeModName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMatchModTitle(t *testing.T) {
	serverMods := []serverModInfo{
		{ID: "1001", Title: "Dabs Framework"},
		{ID: "2002", Title: "@Community-Online-Tool"},
		{ID: "3003", Title: "BiggerMapMod"},
	}
	cases := []struct {
		mod  string
		want string
		ok   bool
	}{
		{"@Dabs Framework", "1001", true},
		{"Community-Online-Tool", "2002", true},
		{"@Bigger Map Mod", "3003", true},
		{"@Unknown Mod", "", false},
	}
	for _, c := range cases {
		id, ok := matchModTitle(c.mod, serverMods)
		if ok != c.ok || (ok && id != c.want) {
			t.Fatalf("matchModTitle(%q) = (%q, %v), want (%q, %v)", c.mod, id, ok, c.want, c.ok)
		}
	}
}
