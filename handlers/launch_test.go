package handlers

import (
	"strings"
	"testing"
)

func TestBuildLaunchURL(t *testing.T) {
	u := buildLaunchURL(221100, "146.66.12.4:2402", "", "@CF;@WOC_RP")
	if !strings.HasPrefix(u, "steam://rungameid/221100//-connect=146.66.12.4:2402") {
		t.Fatalf("bad url: %s", u)
	}
	if !strings.HasSuffix(u, "-mod=@CF;@WOC_RP") {
		t.Fatalf("missing mods: %s", u)
	}

	up := buildLaunchURL(221100, "1.2.3.4:2302", "secret", "@CF")
	if !strings.Contains(up, "-password=secret") {
		t.Fatalf("missing password: %s", up)
	}
}