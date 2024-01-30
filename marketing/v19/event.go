package v19

import (
	"context"

	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
)

// EventService contains all methods for working on events.
type EventService struct {
	c *fb.Client
}

// SimpleList returns event names for a given pixel id.
func (es *EventService) SimpleList(ctx context.Context, pixelID string) ([]string, error) {
	type events struct {
		EventsByHour []struct {
			EventName string `json:"value"`
		} `json:"data"`
	}

	res := []events{}
	route := fb.NewRoute(Version, "/%s/stats", pixelID).
		Limit(250).
		Fields("data{value}").
		Aggregation("event")
	err := es.c.GetList(ctx, route.String(), &res)
	if err != nil {
		return nil, err
	}

	uniqueEventNamesMap := make(map[string]struct{})
	for _, event := range res {
		for _, eventByHour := range event.EventsByHour {
			if eventByHour.EventName != "" {
				uniqueEventNamesMap[eventByHour.EventName] = struct{}{}
			}
		}
	}

	uniqueEventNames := []string{}
	for uniqueEventName := range uniqueEventNamesMap {
		uniqueEventNames = append(uniqueEventNames, uniqueEventName)
	}

	return uniqueEventNames, nil
}
