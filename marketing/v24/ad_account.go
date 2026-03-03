package v24

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

// TargetingValidationResult is a single entry returned by the targetingvalidation endpoint.
type TargetingValidationResult struct {
	ID    string `json:"id"`
	Valid bool   `json:"valid"`
}

// ValidateTargeting validates a list of targeting IDs against the ad account's
// targetingvalidation endpoint. Returns the IDs that are valid and those that are invalid.
func (aas *AdAccountService) ValidateTargeting(ctx context.Context, adAccountID string, ids []string) (validIDs, invalidIDs []string, err error) {
	rb := fb.NewRoute(Version, "/act_%s/targetingvalidation", adAccountID).
		IDList(ids...).
		Limit(len(ids))

	var res []TargetingValidationResult
	err = aas.c.GetList(ctx, rb.String(), &res)
	if err != nil {
		return nil, nil, err
	}

	validSet := make(map[string]struct{}, len(res))
	for _, r := range res {
		if !r.Valid {
			continue
		}
		validSet[r.ID] = struct{}{}
		validIDs = append(validIDs, r.ID)
	}

	for _, id := range ids {
		if _, ok := validSet[id]; !ok {
			invalidIDs = append(invalidIDs, id)
		}
	}

	return validIDs, invalidIDs, nil
}
