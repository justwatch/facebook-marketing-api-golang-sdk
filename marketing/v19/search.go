package v19

import (
	"context"

	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
)

// SearchService performs searches on the graph API.
type SearchService struct {
	c *fb.Client
}

const (
	getGeoLocationLimit    = 5000
	getDevicesResultsLimit = 5000
)

// GetAdGeoLocations returns all AdGeoLocations.
func (s *SearchService) GetAdGeoLocations(ctx context.Context) ([]AdGeoLocation, error) {
	rb := fb.NewRoute(Version, "/search").
		Type("adgeolocation").
		LocationTypes("country", "city", "region").
		Limit(getGeoLocationLimit)
	res := []AdGeoLocation{}
	err := s.c.GetList(ctx, rb.String(), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (s *SearchService) GetRegions(ctx context.Context, country string) ([]AdGeoLocation, error) {
	rb := fb.NewRoute(Version, "/search").
		Type("adgeolocation").
		LocationTypes("region").
		Limit(getGeoLocationLimit)
	if country != "" {
		rb.Q(country)
	}
	res := []AdGeoLocation{}
	err := s.c.GetList(ctx, rb.String(), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// AdGeoLocation contains different geolocation types to be used for adset targeting.
type AdGeoLocation struct {
	Key            string `json:"key"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	CountryCode    string `json:"country_code"`
	SupportsRegion bool   `json:"supports_region"`
	SupportsCity   bool   `json:"supports_city"`
}

// GetDevices returns all devices.
func (s *SearchService) GetDevices(ctx context.Context) ([]Device, error) {
	rb := fb.NewRoute(Version, "/search").
		Type("adTargetingCategory").
		Class("user_device").
		Limit(getDevicesResultsLimit)
	res := []Device{}
	err := s.c.GetList(ctx, rb.String(), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Device contains device information to be used for adset targeting.
type Device struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Platform     string `json:"platform"`
	AudienceSize int    `json:"audience_size"`
	Type         string `json:"type"`
}

// GetOperatingSystems returns all operating systems.
func (s *SearchService) GetOperatingSystems(ctx context.Context) ([]OperatingSystem, error) {
	rb := fb.NewRoute(Version, "/search").
		Type("adTargetingCategory").
		Class("user_os").
		Limit(getDevicesResultsLimit)
	res := []OperatingSystem{}
	err := s.c.GetList(ctx, rb.String(), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// OperatingSystem contains information about an user OS to be used for adset targeting.
type OperatingSystem struct {
	Type        string `json:"type"`
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	Description string `json:"description"`
}

// GetAdLocales returns all ad locales.
func (s *SearchService) GetAdLocales(ctx context.Context) ([]AdLocale, error) {
	rb := fb.NewRoute(Version, "/search").
		Type("adlocale").
		Limit(1000)
	res := []AdLocale{}
	err := s.c.GetList(ctx, rb.String(), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// AdLocale can be used for locale adset targeting.
type AdLocale struct {
	Key  int    `json:"key"`
	Name string `json:"name"`
}

// TargetingOptionStatus is the json response of the ValidateInterest request.
type TargetingOptionStatus struct {
	ID            string `json:"id"`
	CurrentStatus string `json:"current_status"`
	FuturePlan    []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"future_plan"`
}

// ValidateInterests validates a list of interests and returns a list of valid and a list of invalid IDs.
func (s *SearchService) ValidateInterests(ctx context.Context, externalIDs []string) (validIDs []string, invalidIDs []string, err error) {
	rb := fb.NewRoute(Version, "/search").
		Type("targetingoptionstatus").
		TargetingOptionList(externalIDs...).
		Limit(1000)
	var res []TargetingOptionStatus
	err = s.c.GetList(ctx, rb.String(), &res)
	if err != nil {
		return nil, nil, err
	}

	for _, targetingOption := range res {
		if targetingOption.CurrentStatus == "NORMAL" {
			validIDs = append(validIDs, targetingOption.ID)
		} else {
			invalidIDs = append(invalidIDs, targetingOption.ID)
		}
	}

	return validIDs, invalidIDs, nil
}
