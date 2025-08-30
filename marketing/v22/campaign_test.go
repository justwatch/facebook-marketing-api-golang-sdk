package v22_test

import (
	"net/url"
	"strings"
	"testing"

	"github.com/justwatch/facebook-marketing-api-golang-sdk/marketing/v22"
)

func toStrings(status []v22.EffectiveStatus) []string {
	out := make([]string, len(status))
	for i, v := range status {
		out[i] = string(v)
	}
	return out
}

func newTestService() *v22.CampaignService {
	return &v22.CampaignService{}
}

func TestListByEffectiveStatus_buildsEffectiveStatusParam(t *testing.T) {
	cs := newTestService()
	call := cs.ListByEffectiveStatus("123",
		v22.EffectiveStatusActive, v22.EffectiveStatusPaused)

	raw := call.RouteBuilder.String()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}

	got := u.Query().Get("effective_status")
	want := `["ACTIVE","PAUSED"]`
	if got != want {
		t.Fatalf("effective_status mismatch\n got: %s\nwant: %s", got, want)
	}
}

func TestListByEffectiveStatus_emptyOmitsParam(t *testing.T) {
	cs := newTestService()
	call := cs.ListByEffectiveStatus("123") // no statuses

	u, _ := url.Parse(call.RouteBuilder.String())
	if v := u.Query().Get("effective_status"); v != "" {
		t.Fatalf("expected no effective_status param, got %q", v)
	}
}

func TestList_usesDefaultStatuses(t *testing.T) {
	cs := newTestService()
	call := cs.List("123")

	u, _ := url.Parse(call.RouteBuilder.String())
	got := u.Query().Get("effective_status")

	exp := `["` + strings.Join(toStrings(v22.DefaultEffectiveStatuses), `","`) + `"]` // see helper below
	if got != exp {
		t.Fatalf("defaults mismatch\n got: %s\nwant: %s", got, exp)
	}
}
