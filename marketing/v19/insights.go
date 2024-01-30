package v19

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
)

// InsightsService contains all methods for working on audiences.
type InsightsService struct {
	l log.Logger
	c *fb.Client
	*fb.StatsContainer
}

func newInsightsService(l log.Logger, c *fb.Client) *InsightsService {
	return &InsightsService{
		l:              l,
		c:              c,
		StatsContainer: fb.NewStatsContainer(),
	}
}

// NewReport creates a new InsightsRequest.
func (is *InsightsService) NewReport(account string) *InsightsRequest {
	return &InsightsRequest{
		InsightsService: is,
		RouteBuilder:    fb.NewRoute(Version, "/act_%s/insights", account),
	}
}

// NewReportOfCampaign creates a new InsightsRequest.
func (is *InsightsService) NewReportOfCampaign(campaignID string) *InsightsRequest {
	return &InsightsRequest{
		InsightsService: is,
		RouteBuilder:    fb.NewRoute(Version, "/%s/insights", campaignID),
	}
}

// InsightsRequest is used to build a new request for creating an insights run.
type InsightsRequest struct {
	*InsightsService
	*fb.RouteBuilder
}

// Download returns all insights from the request in one slice.
func (ir *InsightsRequest) Download(ctx context.Context) ([]Insight, error) {
	res := []Insight{}
	err := ir.c.GetList(ctx, ir.RouteBuilder.String(), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// GenerateReport creates the insights report, waits until it's finished building, reads to c and then deletes it.
func (ir *InsightsRequest) GenerateReport(ctx context.Context, c chan<- Insight) (uint64, error) {
	run := &struct {
		ReportRunID            string `json:"report_run_id"`
		AccountID              string `json:"account_id"`
		TimeRef                int    `json:"time_ref"`
		TimeCompleted          int    `json:"time_completed"`
		AsyncStatus            string `json:"async_status"`
		IsRunning              bool   `json:"is_running"`
		AsyncPercentCompletion int    `json:"async_percent_completion"`
		DateStart              string `json:"date_start"`
		DateStop               string `json:"date_stop"`
	}{}
	ir.RouteBuilder.DefaultSummary(true)
	ir.RouteBuilder.UnifiedAttributionSettings(true)
	err := ir.c.PostJSON(ctx, ir.RouteBuilder.String(), nil, run)
	if err != nil {
		return 0, err
	} else if run.ReportRunID == "" {
		return 0, errors.New("did not get report run id")
	}

	stats := ir.StatsContainer.AddStats(run.ReportRunID)
	if stats == nil {
		return 0, fmt.Errorf("report run %s already being downloaded", run.ReportRunID)
	}

	defer func() {
		ir.StatsContainer.RemoveStats(run.ReportRunID)
		url := fb.NewRoute(Version, "/%s", run.ReportRunID).String()
		e := ir.c.Delete(ctx, url)
		if e != nil {
			_ = level.Warn(ir.l).Log("msg", "err deleting report run", "id", run.ReportRunID, "err", e, "url", url)
		}
	}()

	t := time.NewTicker(15 * time.Second)
	timeout := time.NewTimer(10 * time.Minute)
	lastPercentage := 0
	defer timeout.Stop()
	defer t.Stop()
	for range t.C {
		// non blocking timeout
		select {
		case <-timeout.C:
			return 0, errors.New("report timeout")
		default:
		}
		run.IsRunning = false // field is omitted when it is false, so we need to set it to false manually
		err = ir.c.GetJSON(ctx, fb.NewRoute(Version, "/%s", run.ReportRunID).String(), run)
		if err != nil {
			return 0, err
		}

		if run.AsyncStatus == "Job Completed" && run.AsyncPercentCompletion == 100 && !run.IsRunning {
			stats.SetCreated()

			break
		}
		stats.SetProgress(uint64(run.AsyncPercentCompletion), 100)

		if run.AsyncStatus == "Job Failed" {
			return 0, errors.New("job failed")
		}
		// update timeline if report progresses
		if run.AsyncPercentCompletion > lastPercentage {
			lastPercentage = run.AsyncPercentCompletion
			timeout.Reset(10 * time.Minute)
		}
	}

	url := fb.NewRoute(Version, "/%s/insights", run.ReportRunID).Limit(100).String()
	var count, impressions uint64
	for url != "" {
		resp := &struct {
			fb.Paging
			Summary struct {
				Impressions uint64 `json:"impressions,string"`
			} `json:"summary"`
			Data []Insight `json:"data"`
		}{}

		err := ir.c.GetJSON(ctx, url, resp)
		if err != nil {
			return 0, err
		}

		for _, d := range resp.Data {
			count++
			impressions += d.Impressions
			c <- d
		}
		stats.SetProgress(impressions, resp.Summary.Impressions)
		url = resp.Paging.Paging.Next
	}

	return count, nil
}

// Insight contains insight data for an facebook graph API object, broken down by the desired day.
type Insight struct {
	AccountID                        string                 `json:"account_id"`
	Actions                          ActionTypeValue        `json:"actions"`
	AdsetID                          string                 `json:"adset_id"`
	AdID                             string                 `json:"ad_id"`
	Objective                        string                 `json:"objective"`
	AdsetName                        string                 `json:"adset_name"`
	Age                              string                 `json:"age"`
	CampaignID                       string                 `json:"campaign_id"`
	CampaignName                     string                 `json:"campaign_name"`
	PublisherPlatform                string                 `json:"publisher_platform"`
	PlatformPosition                 string                 `json:"platform_position"`
	Clicks                           uint64                 `json:"clicks,string"`
	DateStart                        string                 `json:"date_start"`
	DateStop                         string                 `json:"date_stop"`
	Frequency                        float64                `json:"frequency,string"`
	Gender                           string                 `json:"gender"`
	Impressions                      uint64                 `json:"impressions,string"`
	Reach                            float64                `json:"reach,string"`
	Spend                            float64                `json:"spend,string"`
	VideoContinues2SecWatchedActions ActionTypeValue        `json:"video_continuous_2_sec_watched_actions"`
	Video15SecWatchedActions         ActionTypeValue        `json:"video_15_sec_watched_actions"`
	VideoThruplayWatchedActions      ActionTypeValue        `json:"video_thruplay_watched_actions"`
	Video30SecWatchedActions         ActionTypeValue        `json:"video_30_sec_watched_actions"`
	VideoAvgTimeWatchedActions       ActionTypeValue        `json:"video_avg_time_watched_actions"`
	VideoP100WatchedActions          ActionTypeValue        `json:"video_p100_watched_actions"`
	VideoP25WatchedActions           ActionTypeValue        `json:"video_p25_watched_actions"`
	VideoP50WatchedActions           ActionTypeValue        `json:"video_p50_watched_actions"`
	VideoP75WatchedActions           ActionTypeValue        `json:"video_p75_watched_actions"`
	VideoP95WatchedActions           ActionTypeValue        `json:"video_p95_watched_actions"`
	VideoPlayActions                 ActionTypeValue        `json:"video_play_actions"`
	InteractiveComponentTap          []InteractiveComponent `json:"interactive_component_tap"`
	DeviceType                       string                 `json:"impression_device"`
	Region                           string                 `json:"region"`
	Country                          string                 `json:"country"`
}

// GetAge returns the min and max age from the insights age field.
func (i Insight) GetAge() (uint64, uint64, error) {
	parts := strings.Split(i.Age, "-")
	var minAge, maxAge uint64
	if len(parts) == 2 {
		var err error
		minAge, err = strconv.ParseUint(parts[0], 10, 64)
		if err != nil {
			return 0, 0, err
		}
		maxAge, err = strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return 0, 0, err
		}
	} else if i.Age == "65+" {
		minAge = 65
		maxAge = 100
	}

	return minAge, maxAge, nil
}

// ActionTypeValue is a kv store.
type ActionTypeValue []struct {
	ActionType       string  `json:"action_type"`
	ActionVideoSound string  `json:"action_video_sound"`
	ActionReaction   string  `json:"action_reaction"`
	Value            float64 `json:"value,string"`

	// For action_attribution_windows
	View1d   float64 `json:"1d_view,string"`
	View7d   float64 `json:"7d_view,string"`
	View28d  float64 `json:"28d_view,string"`
	Click1d  float64 `json:"1d_click,string"`
	Click7d  float64 `json:"7d_click,string"`
	Click28d float64 `json:"28d_click,string"`
}

// GetValue returns the sum of the values with the given action type.
func (atv ActionTypeValue) GetValue(actionType string) float64 {
	var value float64
	for _, a := range atv {
		if a.ActionType == actionType {
			value += a.Value
		}
	}

	return value
}

// GetCustomConversion returns the custom conversions of the values with the given action type.
func (atv ActionTypeValue) GetCustomConversion() float64 {
	var value float64

	for _, a := range atv {
		if strings.Contains(a.ActionType, "offsite_conversion.custom") {
			value = a.Value
		}
	}

	return value
}

// GetReactions returns a map with reactions.
func (atv ActionTypeValue) GetReactions() map[string]uint64 {
	res := map[string]uint64{}
	for _, a := range atv {
		if a.ActionType == "post_reaction" {
			res[a.ActionReaction] = uint64(a.Value)
		}
	}

	return res
}

// FilterByActionTypePrefix returns a slice of ActionTypeValue for which all
// ActionType values starts with the input prefix.
func (atv ActionTypeValue) FilterByActionTypePrefix(prefix string) ActionTypeValue {
	customConversions := ActionTypeValue{}
	for _, a := range atv {
		if strings.HasPrefix(a.ActionType, prefix) {
			customConversions = append(customConversions, a)
		}
	}

	return customConversions
}

// InteractiveComponent represents poll results.
type InteractiveComponent struct {
	InteractiveComponentStickerID       string `json:"interactive_component_sticker_id"`
	InteractiveComponentStickerResponse string `json:"interactive_component_sticker_response"`
	Value                               string `json:"value"`
}
