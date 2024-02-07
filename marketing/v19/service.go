package v19

import (
	"context"
	"errors"

	"github.com/go-kit/kit/log"
	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
)

// Version of the graph API being used.
const Version = "v19.0"

// Service interacts with the Facebook Marketing API.
type Service struct {
	*fb.Client
	AdAccounts        *AdAccountService
	AdCreatives       *AdCreativeService
	Adsets            *AdsetService
	Ads               *AdService
	Audiences         *AudienceService
	Campaigns         *CampaignService
	CustomConversions *CustomConversionService
	Events            *EventService
	Insights          *InsightsService
	Interests         *InterestService
	Images            *ImageService
	Pages             *PageService
	Posts             *PostService
	Search            *SearchService
	Videos            *VideoService
}

// New initializes a new Service and all the Services contained.
func New(l log.Logger, accessToken, appSecret string) (*Service, error) {
	c := fb.NewClient(l, accessToken, appSecret)
	err := c.GetJSON(context.Background(), fb.NewRoute(Version, "/me").String(), &struct{}{})
	if err != nil {
		return nil, err
	}

	return &Service{
		Client:            c,
		AdAccounts:        &AdAccountService{c},
		AdCreatives:       &AdCreativeService{c, fb.NewStatsContainer()},
		Adsets:            &AdsetService{c},
		Ads:               &AdService{c},
		Audiences:         &AudienceService{c},
		Campaigns:         &CampaignService{c},
		CustomConversions: &CustomConversionService{c},
		Events:            &EventService{c},
		Insights:          newInsightsService(l, c),
		Interests:         &InterestService{c},
		Images:            &ImageService{c},
		Pages:             &PageService{c},
		Posts:             &PostService{c, fb.NewStatsContainer()},
		Search:            &SearchService{c},
		Videos:            &VideoService{c},
	}, nil
}

// GetMetadata returns the metadata of a graph API object.
func (s *Service) GetMetadata(ctx context.Context, id string) (*fb.Metadata, error) {
	res := &fb.MetadataContainer{}
	err := s.Client.GetJSON(ctx, fb.NewRoute(Version, "/%s", id).Metadata(true).String(), res)
	if err != nil {
		return nil, err
	} else if res.Metadata == nil {
		return nil, errors.New("could not get metadata")
	}

	return res.Metadata, nil
}
