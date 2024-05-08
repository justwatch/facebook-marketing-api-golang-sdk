package types

import (
	"fmt"
	"testing"
)

func TestCustomerInfoEmailHashing(t *testing.T) {
	for _, td := range []struct {
		in  string
		out string
	}{
		{"", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"test something", "1f1f0339c99d3760514132f7a62b16906d8b431beeb40ba54841a447aa7be180"},
	} {
		t.Run(fmt.Sprintf("check hash for [%s]", td.in), func(t *testing.T) {
			userData := NewCustomerInformation().WithEmail(td.in)
			if userData.Email[0] != td.out {
				fmt.Printf("expected [%v] got [%v]", td.out, userData.Email[0])
				t.Fail()
			}
		})
	}
}
