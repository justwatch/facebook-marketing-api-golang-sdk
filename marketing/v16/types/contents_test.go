package types

import (
	"testing"
)

func TestContents(t *testing.T) {
	contents := NewContents().AddContent(Content{ID: "some id", Quantity: 15, DeliveryCategory: "category"})
	if len(contents) != 1 {
		t.Fail()
	}
}
