package v19

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
)

const (
	adsetListLimit = 50
)

// AdsetService is used for working with adsets.
type AdsetService struct {
	c *fb.Client
}

// Get returns a single Adset.
func (as *AdsetService) Get(ctx context.Context, id string, fields ...string) (*Adset, error) {
	if len(fields) == 0 {
		fields = AdsetFields
	}
	res := &Adset{}
	err := as.c.GetJSON(ctx, fb.NewRoute(Version, "/%s", id).Fields(fields...).String(), res)
	if err != nil {
		if fb.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return res, nil
}

// GetDeliveryEstimate returns the delivery_estimate mau for a given adset.
func (as *AdsetService) GetDeliveryEstimate(ctx context.Context, id string, t *Targeting) (uint64, error) {
	r := fb.NewRoute(Version, "/%s/delivery_estimate", id).Limit(10).Fields("estimate_mau_upper_bound")

	if t != nil {
		r.TargetingSpec(t)
	}

	var de []*Data
	err := as.c.GetList(ctx, r.String(), &de)
	if err != nil {
		return 0, err
	}

	var size int64
	for _, e := range de {
		if e.EstimateMauUpperBound > size {
			size = e.EstimateMauUpperBound
		}
	}

	return uint64(size), nil
}

// Create uploads a new adset, returns the fields and returns the created adset.
func (as *AdsetService) Create(ctx context.Context, a Adset) (string, fb.Time, error) {
	if a.ID != "" {
		return "", fb.Time{}, fmt.Errorf("cannot create adset that already exists: %s", a.ID)
	} else if a.AccountID == "" {
		return "", fb.Time{}, errors.New("cannot create adset without account id")
	}

	res := &fb.MinimalResponse{}
	err := as.c.PostJSON(ctx, fb.NewRoute(Version, "/act_%s/adsets", a.AccountID).Fields("updated_time", "id").String(), a, res)
	if err != nil {
		return "", fb.Time{}, err
	} else if err = res.GetError(); err != nil {
		return "", fb.Time{}, err
	} else if res.ID == "" {
		return "", fb.Time{}, fmt.Errorf("creating adset failed")
	}

	return res.ID, res.UpdatedTime, nil
}

// Update updates an adset.
func (as *AdsetService) Update(ctx context.Context, a Adset) (fb.Time, error) {
	if a.ID == "" {
		return fb.Time{}, errors.New("cannot update adset without id")
	}

	res := &fb.MinimalResponse{}
	err := as.c.PostJSON(ctx, fb.NewRoute(Version, "/%s", a.ID).Fields("updated_time", "id").String(), a, res)
	if err != nil {
		return fb.Time{}, err
	} else if err = res.GetError(); err != nil {
		return fb.Time{}, err
	} else if !res.Success && res.ID == "" {
		return fb.Time{}, fmt.Errorf("updating the adset failed")
	}

	return res.UpdatedTime, nil
}

// List returns a list of adsets for an account.
func (as *AdsetService) List(account string, fields []string) *AdsetListCall {
	if len(fields) == 0 {
		fields = AdsetFields
	}

	return &AdsetListCall{
		RouteBuilder: fb.NewRoute(Version, "/act_%s/adsets", account).Limit(adsetListLimit).Fields(fields...),
		c:            as.c,
	}
}

// ListOfCampaign returns an adsetlistcall for listing the adsets of a campaign id.
func (as *AdsetService) ListOfCampaign(campaignID string, fields []string) *AdsetListCall {
	if len(fields) == 0 {
		fields = AdsetFields
	}

	return &AdsetListCall{
		RouteBuilder: fb.NewRoute(Version, "/%s/adsets", campaignID).Limit(adsetListLimit).Fields(fields...),
		c:            as.c,
	}
}

// CountAdSets returns the total amount of active adsets.
func (as *AdsetService) CountAdSets(ctx context.Context, accountID string) (uint64, error) {
	sc := &fb.SummaryContainer{}
	err := as.c.GetJSON(ctx, fb.NewRoute(Version, "/act_%s/adsets", accountID).Limit(0).Summary("1").String(), sc)

	return sc.Summary.TotalCount, err
}

// AdsetListCall is used for Listing adsets.
type AdsetListCall struct {
	*fb.RouteBuilder
	c *fb.Client
}

// Do calls the graph API.
func (as *AdsetListCall) Do(ctx context.Context) ([]Adset, error) {
	res := []Adset{}
	err := as.c.GetList(ctx, as.RouteBuilder.String(), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// AdsetFields is a selection of fields to be returned.
var AdsetFields = []string{
	"bid_amount", "attribution_spec", "bid_info",
	"billing_event", "campaign_id", "created_time",
	"daily_budget", "destination_type", "effective_status",
	"daily_spend_cap", "daily_min_spend_target", "end_time",
	"creative_sequence", "frequency_control_specs", "id",
	"configured_status", "instagram_actor_id", "lifetime_budget",
	"lifetime_imps", "lifetime_min_spend_target", "lifetime_spend_cap",
	"name", "budget_remaining", "optimization_goal", "adset_schedule",
	"adlabels", "recurring_budget_semantics",
	"rf_prediction_id", "source_adset_id", "start_time", "targeting",
	"time_based_ad_rotation_id_blocks", "time_based_ad_rotation_intervals",
	"pacing_type", "promoted_object", "recommendations",
	"source_adset", "status", "updated_time", "use_new_app_click",
	"campaign{name,objective,effective_status}", "dsa_beneficiary", "dsa_payor",
}

// Adset from https://developers.facebook.com/docs/marketing-api/reference/ad-campaign
type Adset struct {
	AccountID                  string                 `json:"account_id,omitempty"`
	AttributionSpec            json.RawMessage        `json:"attribution_spec,omitempty"`
	BidAmount                  uint64                 `json:"bid_amount,omitempty"`
	BidStrategy                string                 `json:"bid_strategy,omitempty"`
	BillingEvent               string                 `json:"billing_event,omitempty"`
	BudgetRemaining            float64                `json:"budget_remaining,omitempty,string"`
	Campaign                   *Campaign              `json:"campaign,omitempty"`
	CampaignID                 string                 `json:"campaign_id,omitempty"`
	ConfiguredStatus           string                 `json:"configured_status,omitempty"`
	CreatedTime                fb.Time                `json:"created_time,omitempty"`
	DailyBudget                float64                `json:"daily_budget,omitempty,string"`
	DailyMinSpendTarget        uint64                 `json:"daily_min_spend_target,omitempty,string"`
	DailySpendCap              uint64                 `json:"daily_spend_cap,omitempty,string"`
	DestinationType            string                 `json:"destination_type,omitempty"`
	DeliveryEstimate           *DeliveryEstimate      `json:"delivery_estimate,omitempty"`
	EffectiveStatus            string                 `json:"effective_status,omitempty"`
	EndTime                    fb.Time                `json:"end_time,omitempty"`
	FrequencyControlSpecs      []FrequencyControlSpec `json:"frequency_control_specs,omitempty"`
	ID                         string                 `json:"id,omitempty"`
	LifetimeBudget             float64                `json:"lifetime_budget,omitempty,string"`
	LifetimeMinSpendTarget     uint64                 `json:"lifetime_min_spend_target,omitempty,string"`
	LifeTimeSpendCap           uint64                 `json:"lifetime_spend_cap,omitempty,string"`
	LifetimeImps               uint64                 `json:"lifetime_imps,omitempty"`
	Name                       string                 `json:"name,omitempty"`
	OptimizationGoal           string                 `json:"optimization_goal,omitempty"`
	PacingType                 []string               `json:"pacing_type,omitempty"`
	PromotedObject             *PromotedObject        `json:"promoted_object,omitempty"`
	RecurringBudgetSemantics   bool                   `json:"recurring_budget_semantics,omitempty"`
	StartTime                  fb.Time                `json:"start_time,omitempty"`
	Status                     string                 `json:"status,omitempty"`
	Targeting                  *Targeting             `json:"targeting,omitempty"`
	UpdatedTime                fb.Time                `json:"updated_time,omitempty"`
	TargetingOptimizationTypes map[string]int32       `json:"targeting_optimization_types,omitempty"`
	DSABeneficiary             string                 `json:"dsa_beneficiary,omitempty"`
	DSAPayor                   string                 `json:"dsa_payor,omitempty"`
}

// FrequencyControlSpec controls the frequency of an adset.
type FrequencyControlSpec struct {
	Event        string `json:"event"`
	IntervalDays uint64 `json:"interval_days"`
	MaxFrequency uint64 `json:"max_frequency"`
}

// PromotedObject contains the id of a promoted page.
type PromotedObject struct {
	PageID             string `json:"page_id,omitempty"`
	PixelID            string `json:"pixel_id,omitempty"`
	PixelRule          string `json:"pixel_rule,omitempty"`
	CustomEventType    string `json:"custom_event_type,omitempty"`
	CustomConversionID string `json:"custom_conversion_id,omitempty"`
}

// Targeting contains all the targeting information of an adset.
type Targeting struct {
	// inventories
	PublisherPlatforms []string `json:"publisher_platforms,omitempty"`
	// sub inventories
	FacebookPositions        []string `json:"facebook_positions,omitempty"`
	InstagramPositions       []string `json:"instagram_positions,omitempty"`
	AudienceNetworkPositions []string `json:"audience_network_positions,omitempty"`
	MessengerPositions       []string `json:"messenger_positions,omitempty"`

	AgeMin  uint64 `json:"age_min,omitempty"`
	AgeMax  uint64 `json:"age_max,omitempty"`
	Genders []int  `json:"genders,omitempty"`

	AppInstallState string `json:"app_install_state,omitempty"`

	CustomAudiences         []IDContainer  `json:"custom_audiences,omitempty"`
	ExcludedCustomAudiences []IDContainer  `json:"excluded_custom_audiences,omitempty"`
	GeoLocations            *GeoLocations  `json:"geo_locations,omitempty"`
	ExcludedGeoLocations    *GeoLocations  `json:"excluded_geo_locations,omitempty"`
	FlexibleSpec            []FlexibleSpec `json:"flexible_spec,omitempty"`
	Exclusions              *FlexibleSpec  `json:"exclusions,omitempty"`

	DevicePlatforms             []string                 `json:"device_platforms,omitempty"`
	ExcludedPublisherCategories []string                 `json:"excluded_publisher_categories,omitempty"`
	Locales                     []int                    `json:"locales,omitempty"`
	TargetingOptimization       string                   `json:"targeting_optimization,omitempty"`
	UserDevice                  []string                 `json:"user_device,omitempty"`
	UserOs                      []string                 `json:"user_os,omitempty"`
	WirelessCarrier             []string                 `json:"wireless_carrier,omitempty"`
	TargetingRelaxationTypes    TargetingRelaxationTypes `json:"targeting_relaxation_types,omitempty"`
}

// Advantage custom audience and Advantage lookalike can be enabled or disabled.
// if a value of 0 is passed, it will be disabled. If a value of 1 is passed, it will be enabled.
// If no key/value pair is passed, it will be considered as enabled.
// https://developers.facebook.com/docs/graph-api/changelog/version15.0/
type TargetingRelaxationTypes struct {
	CustomAudience int8 `json:"custom_audience"`
	Lookalike      int8 `json:"lookalike"`
}

// FlexibleSpec is used for targeting
type FlexibleSpec struct {
	Interests            []IDContainer `json:"interests,omitempty"`
	Behaviors            []IDContainer `json:"behaviors,omitempty"`
	LifeEvents           []IDContainer `json:"life_events,omitempty"`
	WorkEmployers        []IDContainer `json:"work_employers,omitempty"`
	FamilyStatuses       []IDContainer `json:"family_statuses,omitempty"`
	WorkPositions        []IDContainer `json:"work_positions,omitempty"`
	Politics             []IDContainer `json:"politics,omitempty"`
	EducationMajors      []IDContainer `json:"education_majors,omitempty"`
	EducationStatuses    []int         `json:"education_statuses,omitempty"`
	RelationshipStatuses []int         `json:"relationship_statuses,omitempty"`
}

// IDContainer contains an ID and a name.
type IDContainer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GeoLocations is a set of countries, cities, and regions that can be targeted.
type GeoLocations struct {
	Countries     []string `json:"countries,omitempty"`
	LocationTypes []string `json:"location_types,omitempty"`
	Cities        []City   `json:"cities,omitempty"`
	Regions       []Region `json:"regions,omitempty"`
	Zips          []Zip    `json:"zips,omitempty"`
}

// City can be targeted.
type City struct {
	Country      string `json:"country"`
	DistanceUnit string `json:"distance_unit"`
	Key          string `json:"key"`
	Name         string `json:"name"`
	Radius       int    `json:"radius"`
	Region       string `json:"region"`
	RegionID     string `json:"region_id"`
}

// Region can be targeted.
type Region struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Country string `json:"country"`
}

// Zip can be targeted.
type Zip struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	PrimaryCityID int    `json:"primary_city_id"`
	RegionID      int    `json:"region_id"`
	Country       string `json:"country"`
}

// DailyOutcomesCurve talk to phillip.
type DailyOutcomesCurve struct {
	Spend       float64 `json:"spend"`
	Reach       float64 `json:"reach"`
	Impressions float64 `json:"impressions"`
	Actions     float64 `json:"actions"`
}

// Data is a single delivery estimate.
type Data struct {
	DailyOutcomesCurve    []DailyOutcomesCurve `json:"daily_outcomes_curve"`
	EstimateDau           int64                `json:"estimate_dau"`
	EstimateMau           int64                `json:"estimate_mau"`
	EstimateMauLowerBound int64                `json:"estimate_mau_lower_bound"`
	EstimateMauUpperBound int64                `json:"estimate_mau_upper_bound"`
	EstimateReady         bool                 `json:"estimate_ready"`
}

// DeliveryEstimate is a collection of Data structs.
type DeliveryEstimate struct {
	Data []Data `json:"data"`
}
