package v24

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestFlexibleSpecCustomAudienceRoundTrip(t *testing.T) {
	input := `[{"custom_audiences":[{"id":"123","name":"First party"}],"interests":[{"id":"456","name":"Movies"}]},{"interests":[{"id":"789","name":"Comedy"}]}]`
	var groups []FlexibleSpec
	if err := json.Unmarshal([]byte(input), &groups); err != nil {
		t.Fatal(err)
	}
	output, err := json.Marshal(groups)
	if err != nil {
		t.Fatal(err)
	}
	var want, got any
	if err := json.Unmarshal([]byte(input), &want); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("mixed groups changed on round trip:\nwant %s\ngot  %s", input, output)
	}
}

func TestFlexibleSpecLegacyJSONUnchanged(t *testing.T) {
	output, err := json.Marshal(FlexibleSpec{Interests: []IDContainer{{ID: "456", Name: "Movies"}}})
	if err != nil {
		t.Fatal(err)
	}
	if string(output) != `{"interests":[{"id":"456","name":"Movies"}]}` {
		t.Fatalf("legacy targeting changed: %s", output)
	}
}
