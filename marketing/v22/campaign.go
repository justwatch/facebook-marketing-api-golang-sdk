package v22

import (
	"context"
	"errors"
	"fmt"

	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
)

// EffectiveStatus represents the possible values for a campaign's
// effective_status field as defined by the Facebook Ads API.
// Using a typed constant reduces the chance of typos compared
// to raw string values.
type EffectiveStatus string

const (
	EffectiveStatusActive             EffectiveStatus = "ACTIVE"
	EffectiveStatusPaused             EffectiveStatus = "PAUSED"
	EffectiveStatusDeleted            EffectiveStatus = "DELETED"
	EffectiveStatusPendingReview      EffectiveStatus = "PENDING_REVIEW"
	EffectiveStatusDisapproved        EffectiveStatus = "DISAPPROVED"
	EffectiveStatusPreApproved        EffectiveStatus = "PREAPPROVED"
	EffectiveStatusPendingBillingInfo EffectiveStatus = "PENDING_BILLING_INFO"
	EffectiveStatusCampaignPaused     EffectiveStatus = "CAMPAIGN_PAUSED"
	EffectiveStatusArchived           EffectiveStatus = "ARCHIVED"
	EffectiveStatusAdsetPaused        EffectiveStatus = "ADSET_PAUSED"
)

// DefaultEffectiveStatuses is the default set of effective_status values
// returned by the CampaignService.List call.
var DefaultEffectiveStatuses = []EffectiveStatus{
	EffectiveStatusActive,
	EffectiveStatusPaused,
	EffectiveStatusDeleted,
	EffectiveStatusPendingReview,
	EffectiveStatusDisapproved,
	EffectiveStatusPreApproved,
	EffectiveStatusPendingBillingInfo,
	EffectiveStatusCampaignPaused,
	EffectiveStatusArchived,
	EffectiveStatusAdsetPaused,
}

// toStrings converts a slice of EffectiveStatus values into a slice of
// raw strings. This is useful when constructing filtering parameters
// for API calls that expect string values.
func toStrings(status []EffectiveStatus) []string {
	out := make([]string, len(status))
	for i, v := range status {
		out[i] = string(v)
	}
	return out
}

// CampaignService works with campaigns.
type CampaignService struct {
	c *fb.Client
}

// Get returns a single campaign.
func (cs *CampaignService) Get(ctx context.Context, id string, fields ...string) (*Campaign, error) {
	if len(fields) == 0 {
		fields = campaignFields
	}
	res := &Campaign{}
	err := cs.c.GetJSON(ctx, fb.NewRoute(Version, "/%s", id).Fields(fields...).String(), res)
	if err != nil {
		if fb.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return res, nil
}

// Create uploads a new campaign, returns the fields and returns the created campaign.
func (cs *CampaignService) Create(ctx context.Context, c Campaign) (string, error) {
	if c.ID != "" {
		return "", fmt.Errorf("cannot create campaign that already exists: %s", c.ID)
	} else if c.AccountID == "" {
		return "", errors.New("cannot create campaign without account id")
	}

	res := &fb.MinimalResponse{}
	url := fb.NewRoute(Version, "/act_%s/campaigns", c.AccountID).String()
	err := cs.c.PostJSON(ctx, url, c, res)
	if err != nil {
		return "", fmt.Errorf("could not POST to %q: %w", url, err)
	} else if err = res.GetError(); err != nil {
		return "", fmt.Errorf("got error response from POST to %q: %w", url, err)
	} else if res.ID == "" {
		return "", fmt.Errorf("creating campaign failed")
	}

	return res.ID, nil
}

// Update updates an campaign.
func (cs *CampaignService) Update(ctx context.Context, c Campaign) error {
	if c.ID == "" {
		return errors.New("cannot update a campaign without id")
	}

	res := &fb.MinimalResponse{}
	err := cs.c.PostJSON(ctx, fb.NewRoute(Version, "/%s", c.ID).String(), c, res)
	if err != nil {
		return err
	} else if err = res.GetError(); err != nil {
		return err
	} else if !res.Success && res.ID == "" {
		return fmt.Errorf("updating the campaign failed")
	}

	return nil
}

// List creates a new CampaignListCall.
func (cs *CampaignService) List(act string) *CampaignListCall {
	return cs.ListByEffectiveStatus(act, DefaultEffectiveStatuses...)
}

// ListByEffectiveStatus returns a CampaignListCall filtered by the given
// effective_status values. If no statuses are provided, the filter is omitted.
// Behavior otherwise matches List.
func (cs *CampaignService) ListByEffectiveStatus(act string, statuses ...EffectiveStatus) *CampaignListCall {
	rb := fb.NewRoute(Version, "/act_%s/campaigns", act).
		Fields(campaignFieldsShort...).
		Limit(1000)

	if len(statuses) > 0 {
		rb = rb.Filtering(
			fb.Filter{
				Field:    "effective_status",
				Operator: "IN",
				Value:    toStrings(statuses),
			},
		)
	}

	return &CampaignListCall{
		RouteBuilder: rb, c: cs.c,
	}
}

// CampaignListCall is used for listing campaigns.
type CampaignListCall struct {
	*fb.RouteBuilder
	c *fb.Client
}

// Do function performs the CampaignListCall and returns all campaigns as slice.
func (csc *CampaignListCall) Do(ctx context.Context) ([]Campaign, error) {
	res := []Campaign{}
	err := csc.c.GetList(ctx, csc.RouteBuilder.String(), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

var campaignFields = []string{
	"id",
	"account_id",
	"adlabels",
	"bid_strategy",
	"boosted_object_id",
	"brand_lift_studies",
	"budget_rebalance_flag",
	"budget_remaining",
	"buying_type",
	"can_create_brand_lift_study",
	"can_use_spend_cap",
	"configured_status",
	"created_time",
	"daily_budget",
	"effective_status",
	"lifetime_budget",
	"name",
	"objective",
	"recommendations",
	"source_campaign",
	"source_campaign_id",
	"spend_cap",
	"start_time",
	"status",
	"stop_time",
	"updated_time",
}

// campaignFieldsShort are the fields required for the Sub Campaign Group sync.
var campaignFieldsShort = []string{
	"id",
	"name",
	"objective",
	"status",
	"spend_cap",
	"start_time",
	"stop_time",
	"buying_type",
	"can_use_spend_cap",
	"updated_time",
	"daily_budget",
	"lifetime_budget",
	"bid_strategy",
}

// Campaign from https://developers.facebook.com/docs/marketing-api/reference/ad-campaign-group
type Campaign struct {
	AccountID           string   `json:"account_id,omitempty"`
	BuyingType          string   `json:"buying_type,omitempty"`
	CampaignGroupID     string   `json:"campaign_group_id,omitempty"`
	BidStrategy         string   `json:"bid_strategy,omitempty"`
	BidAmount           uint64   `json:"bid_amount,omitempty"`
	CanUseSpendCap      bool     `json:"can_use_spend_cap,omitempty"`
	ConfiguredStatus    string   `json:"configured_status,omitempty"`
	CreatedTime         string   `json:"created_time,omitempty"`
	DailyBudget         uint64   `json:"daily_budget,omitempty,string"`
	EffectiveStatus     string   `json:"effective_status,omitempty"`
	ID                  string   `json:"id,omitempty"`
	LifeTimeBudget      uint64   `json:"lifetime_budget,omitempty,string"`
	Name                string   `json:"name,omitempty"`
	Objective           string   `json:"objective,omitempty"`
	SpendCap            uint64   `json:"spend_cap,omitempty,string"`
	StartTime           fb.Time  `json:"start_time,omitempty"`
	Status              string   `json:"status,omitempty"`
	StopTime            fb.Time  `json:"stop_time,omitempty"`
	UpdatedTime         fb.Time  `json:"updated_time,omitempty"`
	SpecialAdCategories []string `json:"special_ad_categories,omitempty"`
}
