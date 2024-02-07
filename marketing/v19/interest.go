package v19

import (
	"context"
	"strings"

	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
)

// InterestService works with ads interests.
type InterestService struct {
	c *fb.Client
}

// Search returns a list of InterestTargetings.
func (is *InterestService) Search(ctx context.Context, query string, limit int) ([]InterestTargeting, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []InterestTargeting{}, nil
	}

	res := []InterestTargeting{}
	err := is.c.GetList(ctx, fb.NewRoute(Version, "/search").Type("adinterest").Q(query).Limit(limit).Fields("id", "name", "audience_size_upper_bound", "path", "description", "topic").String(), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// TargetingSearch searches for a targeting.
func (is *InterestService) TargetingSearch(ctx context.Context, act string, query string) ([]InterestTargeting, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []InterestTargeting{}, nil
	}

	res := []InterestTargeting{}
	err := is.c.GetList(ctx, fb.NewRoute(Version, "/act_%s/targetingsearch", act).Q(query).String(), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// InterestTargeting represents an ad interest to be used in an adset targeting.
type InterestTargeting struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	AudienceSize uint64   `json:"audience_size_upper_bound"`
	Type         string   `json:"type"`
	Path         []string `json:"path"`
}
