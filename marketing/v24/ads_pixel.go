package v24

import (
	"context"
	"time"

	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
)

// AdsPixelService works on ad account pixels.
type AdsPixelService struct {
	c *fb.Client
}

// AdsPixel is a Meta pixel visible to an ad account.
type AdsPixel struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	CreationTime  fb.Time `json:"creation_time"`
	LastFiredTime fb.Time `json:"last_fired_time"`
}

// AdsPixelStats is one aggregation bucket of pixel stats.
type AdsPixelStats struct {
	StartTime fb.Time              `json:"start_time"`
	Data      []AdsPixelStatsEntry `json:"data"`
}

// AdsPixelStatsEntry holds the count of one aggregation value, e.g. an event name.
type AdsPixelStatsEntry struct {
	Value string `json:"value"`
	Count int64  `json:"count"`
}

// List returns the pixels of an ad account.
func (aps *AdsPixelService) List(ctx context.Context, act string) ([]AdsPixel, error) {
	pixels := []AdsPixel{}
	route := fb.NewRoute(Version, "/act_%s/adspixels", act).
		Limit(250).
		Fields("id", "name", "creation_time", "last_fired_time")
	err := aps.c.GetList(ctx, route.String(), &pixels)
	if err != nil {
		return nil, err
	}

	return pixels, nil
}

// Stats returns the pixel's fired events from the given start time on, aggregated by the given dimension (e.g. "event").
func (aps *AdsPixelService) Stats(ctx context.Context, pixelID, aggregation string, startTime time.Time) ([]AdsPixelStats, error) {
	stats := []AdsPixelStats{}
	route := fb.NewRoute(Version, "/%s/stats", pixelID).
		Aggregation(aggregation).
		StartTime(startTime)
	err := aps.c.GetList(ctx, route.String(), &stats)
	if err != nil {
		return nil, err
	}

	return stats, nil
}
