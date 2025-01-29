package v19

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
	"golang.org/x/sync/errgroup"
)

// AdService works with Ads.
type AdService struct {
	c *fb.Client
}

// Get returns a single ad.
func (as *AdService) Get(ctx context.Context, id string) (*Ad, error) {
	res := &Ad{}
	err := as.c.GetJSON(ctx, fb.NewRoute(Version, "/%s", id).Fields("id", "creative", "name", "account_id", "adset_id",
		"adset{id,daily_budget,name,start_time,end_time,status,bid_strategy,targeting{age_min,age_max,publisher_platforms,geo_locations,genders,custom_audiences,excluded_custom_audiences,flexible_spec,exclusions}}",
		"adcreatives{id,title,object_story_spec}").Limit(1000).String(), res)
	if err != nil {
		if fb.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return res, nil
}

// Create uploads a new ad, returns the fields and returns the created ad.
func (as *AdService) Create(ctx context.Context, a Ad) (string, error) {
	if a.ID != "" {
		return "", fmt.Errorf("cannot create ad that already exists: %s", a.ID)
	} else if a.AccountID == "" {
		return "", errors.New("cannot create ad without account id")
	}

	res := &fb.MinimalResponse{}
	err := as.c.PostJSON(ctx, fb.NewRoute(Version, "/act_%s/ads", a.AccountID).String(), a, res)
	if err != nil {
		return "", err
	} else if err = res.GetError(); err != nil {
		return "", err
	} else if res.ID == "" {
		return "", fmt.Errorf("creating ad failed")
	}

	return res.ID, nil
}

// Update updates an ad.
func (as *AdService) Update(ctx context.Context, a Ad) error {
	if a.ID == "" {
		return errors.New("cannot update ad without id")
	}

	res := &fb.MinimalResponse{}
	err := as.c.PostJSON(ctx, fb.NewRoute(Version, "/%s", a.ID).String(), a, res)
	if err != nil {
		return err
	} else if err = res.GetError(); err != nil {
		return err
	} else if !res.Success && res.ID == "" {
		return fmt.Errorf("updating the ad failed")
	}

	return nil
}

// List returns all ads of an account.
func (as *AdService) List(act string) *AdListCall {
	return &AdListCall{
		RouteBuilder: fb.NewRoute(Version, "/act_%s/ads", act).Fields("adset_id", "creative", "id", "name", "account_id", "adset{id}", "adcreatives{id}").Limit(1000),
		c:            as.c,
	}
}

// ListOfAdset returns all ads of an adset.
func (as *AdService) ListOfAdset(adsetID string) *AdListCall {
	return &AdListCall{
		RouteBuilder: fb.NewRoute(Version, "/%s/ads", adsetID).Fields("id", "adset{id}", "adcreatives{id}").Limit(1000),
		c:            as.c,
	}
}

// AdListCall is used for Listing ads.
type AdListCall struct {
	*fb.RouteBuilder
	c *fb.Client
}

// Do calls the graph API.
func (as *AdListCall) Do(ctx context.Context) ([]Ad, error) {
	res := []Ad{}
	err := as.c.GetList(ctx, as.RouteBuilder.String(), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// Do calls the graph API.
func (as *AdListCall) Read(ctx context.Context, c chan<- Ad) error {
	jres := make(chan json.RawMessage)
	wg := errgroup.Group{}
	wg.Go(func() error {
		defer close(jres)

		return as.c.ReadList(ctx, as.RouteBuilder.String(), jres)
	})
	wg.Go(func() error {
		for e := range jres {
			v := Ad{}
			err := json.Unmarshal(e, &v)
			if err != nil {
				return err
			}
			c <- v
		}

		return nil
	})

	return wg.Wait()
}

// Ad represents a Facebook Ad.
type Ad struct {
	AccountID     string                  `json:"account_id,omitempty"`
	ID            string                  `json:"id,omitempty"`
	Name          string                  `json:"name,omitempty"`
	Status        string                  `json:"status,omitempty"`
	AdsetID       string                  `json:"adset_id,omitempty"`
	Creative      *AdCreative             `json:"creative,omitempty"`
	Adset         *Adset                  `json:"adset,omitempty"`
	TrackingSpecs []ConversionActionQuery `json:"tracking_specs,omitempty"`
	Adcreatives   *struct {
		Data []AdCreative `json:"data,omitempty"`
	} `json:"adcreatives,omitempty"`
}

// ConversionActionQuery contains tracking specs.
type ConversionActionQuery struct {
	ActionType []string `json:"action.type,omitempty"`
	FbPixel    []string `json:"fb_pixel,omitempty"`
	Page       []string `json:"page,omitempty"`
	Post       []string `json:"post,omitempty"`
}
