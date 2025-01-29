package v19

import (
	"context"
	"errors"
	"fmt"

	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
)

// CustomConversionService contains all methods for working on custom conversions.
type CustomConversionService struct {
	c *fb.Client
}

// Create uploads a new custom conversion and returns the id of the custom conversion.
func (ccs *CustomConversionService) Create(ctx context.Context, businessID string, cc CustomConversion) (string, error) {
	if cc.ID != "" {
		return "", fmt.Errorf("cannot create custom conversion that already exists: %s", cc.ID)
	} else if businessID == "" {
		return "", errors.New("cannot create custom conversion without business id")
	}

	res := &fb.MinimalResponse{}
	err := ccs.c.PostJSON(ctx, fb.NewRoute(Version, "/%s/customconversions", businessID).String(), cc, res)
	if err != nil {
		return "", err
	} else if err = res.GetError(); err != nil {
		return "", err
	} else if res.ID == "" {
		return "", fmt.Errorf("creating custom conversion failed")
	}

	return res.ID, nil
}

// List returns all custom conversions for the specified account.
func (ccs *CustomConversionService) List(ctx context.Context, act string) ([]CustomConversion, error) {
	customConversions := []CustomConversion{}
	route := fb.NewRoute(Version, "/act_%s/customconversions", act).
		Limit(250).
		Fields(
			"id",
			"account_id",
			"aggregation_rule",
			"business",
			"name",
			"rule",
			"custom_event_type",
			"creation_time",
			"data_sources",
			"description",
			"default_conversion_value",
			"last_fired_time",
			"is_unavailable",
			"is_archived",
			"first_fired_time",
			//	"pixel",
			"offline_conversion_data_set",
			"event_source_type",
			"retention_days",
		)
	err := ccs.c.GetList(ctx, route.String(), &customConversions)
	if err != nil {
		return nil, err
	}

	return customConversions, nil
}

// CustomConversion https://developers.facebook.com/docs/marketing-api/reference/custom-conversion/
type CustomConversion struct {
	ID                       string       `json:"id"`
	AccountID                string       `json:"account_id"`
	AggregationRule          string       `json:"aggregation_rule"`
	Business                 Business     `json:"business"`
	Name                     string       `json:"name"`
	Rule                     string       `json:"rule"`
	CustomEventType          string       `json:"custom_event_type"`
	CreationTime             fb.Time      `json:"creation_time"`
	DataSources              []DataSource `json:"data_sources"`
	Description              string       `json:"description"`
	DefaultConversionValue   int          `json:"default_conversion_value"`
	LastFiredTime            fb.Time      `json:"last_fired_time"`
	IsUnavailable            bool         `json:"is_unavailable"`
	IsArchived               bool         `json:"is_archived"`
	FirstFiredTime           fb.Time      `json:"first_fired_time"`
	Pixel                    Pixel        `json:"pixel"`
	OfflineConversionDataSet interface{}  `json:"offline_conversion_data_set"`
	EventSourceType          string       `json:"event_source_type"`
	RetentionDays            int          `json:"retention_days"`
}

// DataSource is part of a CustomConversion.
type DataSource struct {
	ID         string `json:"id"`
	SourceType string `json:"source_type"`
}

// Pixel is part of a CustomConversion.
type Pixel struct {
	ID string `json:"id"`
}

// Business is part of a CustomConversion.
type Business struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
