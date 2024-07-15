package v19

import (
	"context"

	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
)

// AdAccountService works with ad accounts.
type AdAccountService struct {
	c *fb.Client
}

// List lists all ad accounts that belong to this business.
func (aas *AdAccountService) List(ctx context.Context, businessID string) ([]AdAccount, error) {
	res := []AdAccount{}
	rb := fb.NewRoute(Version, "/%s/owned_ad_accounts", businessID).Limit(1000).Fields("name", "currency", "account_id", "timezone_name")
	err := aas.c.GetList(ctx, rb.String(), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// AdAccount represents an ad account.
type AdAccount struct {
	Name         string `json:"name"`
	AccountID    string `json:"account_id"`
	Currency     string `json:"currency"`
	TimeZoneName string `json:"timezone_name"`
}
