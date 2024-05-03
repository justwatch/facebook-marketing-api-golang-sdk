package v16

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

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

func (css *CustomConversionService) PushServerEvents(ctx context.Context, pixel Pixel, serverEvents ServerEvents, testEventCode, metaConversionAPIAccessToken string) error {
	// Prepare request body (form encoded)
	bodyForm := url.Values{}
	bodyForm.Add("access_token", metaConversionAPIAccessToken)

	// you can send an array of data
	jsonData, err := json.Marshal(serverEvents)
	if err != nil {
		return fmt.Errorf("could not json marshal conversion-event: %w", err)
	}
	bodyForm.Add("data", string(jsonData))

	if testEventCode != "" {
		bodyForm.Add("test_event_code", testEventCode)
	}

	// Do request
	conversionAPIRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("https://graph.facebook.com/%s/%s/events", Version, pixel.ID), strings.NewReader(bodyForm.Encode()))
	if err != nil {
		return fmt.Errorf("could not create request: %w", err)
	}

	resp, err := css.c.Do(conversionAPIRequest)
	if err != nil {
		return fmt.Errorf("could not do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("status code not 200: %d: %w", resp.StatusCode, err)
		}

		return fmt.Errorf("status code not 200: %d. %q", resp.StatusCode, string(body))
	}

	return nil
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

type ServerEvents []ServerEvent

// ServerEvent entity https://developers.facebook.com/docs/marketing-api/conversions-api/parameters/server-event
type ServerEvent struct {
	EventName      string              `json:"event_name"`
	EventID        string              `json:"event_id"`
	EventTime      int64               `json:"event_time"`
	EventSourceURL string              `json:"event_source_url"`
	ActionSource   string              `json:"action_source"`
	UserData       CustomerInformation `json:"user_data,omitempty"`
	Contents       []Contents          `json:"contents,omitempty"`

	// CustomData might represent your cutom struct with any fields.
	CustomData interface{} `json:"custom_data,omitempty"`
}

// Contents entity is a part of standart parameters. A list of JSON objects that contain the product IDs associated with the event plus information about the products
type Contents struct {
	ID               string `json:"id,omitempty"`
	Quantity         int    `json:"quantity,omitempty"`
	DeliveryCategory string `json:"delivery_category,omitempty"`
}

// CutomerInfromation entity https://developers.facebook.com/docs/marketing-api/conversions-api/parameters/customer-information-parameters
type CustomerInformation struct {
	Email           []string `json:"em,omitempty"`
	PhoneNumber     []string `json:"ph,omitempty"`
	ClientIPAddress string   `json:"client_ip_address,omitempty"`
	ClientUserAgent string   `json:"client_user_agent,omitempty"`
	Fbc             string   `json:"fbc,omitempty"`
	Fbp             string   `json:"fbp,omitempty"`
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
