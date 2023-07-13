package fb

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// RouteBuilder helps building facebook API request routes.
type RouteBuilder struct {
	err     error
	version string
	path    string
	v       url.Values
}

// NewRoute starts building a new route.
func NewRoute(version, format string, a ...interface{}) *RouteBuilder {
	return &RouteBuilder{
		version: version,
		v:       url.Values{},
		path:    fmt.Sprintf(format, a...),
	}
}

// Fields sets the fields query param.
func (rb *RouteBuilder) Fields(f ...string) *RouteBuilder {
	if len(f) > 0 {
		rb.v.Set("fields", strings.Join(f, ","))
	} else {
		rb.v.Del("fields")
	}

	return rb
}

// Limit sets the limit param.
func (rb *RouteBuilder) Limit(limit int) *RouteBuilder {
	if limit > -1 {
		rb.v.Set("limit", strconv.Itoa(limit))
	} else {
		rb.v.Del("limit")
	}

	return rb
}

// Type sets the type param.
func (rb *RouteBuilder) Type(s string) *RouteBuilder {
	if s != "" {
		rb.v.Set("type", s)
	} else {
		rb.v.Del("type")
	}

	return rb
}

// Class sets the type param.
func (rb *RouteBuilder) Class(s string) *RouteBuilder {
	if s != "" {
		rb.v.Set("class", s)
	} else {
		rb.v.Del("class")
	}

	return rb
}

// LocationTypes sets the location_types array param.
func (rb *RouteBuilder) LocationTypes(s ...string) *RouteBuilder {
	if len(s) > 0 {
		rb.v.Set("location_types", fmt.Sprintf("['%s']", strings.Join(s, "','")))
	} else {
		rb.v.Del("location_types")
	}

	return rb
}

// ActionBreakdowns sets the action_breakdowns param.
func (rb *RouteBuilder) ActionBreakdowns(s string) *RouteBuilder {
	if s != "" {
		rb.v.Set("action_breakdowns", s)
	} else {
		rb.v.Del("action_breakdowns")
	}

	return rb
}

// Breakdowns sets the breakdowns array param.
func (rb *RouteBuilder) Breakdowns(s ...string) *RouteBuilder {
	if len(s) > 0 {
		rb.v.Set("breakdowns", strings.Join(s, ","))
	} else {
		rb.v.Del("breakdowns")
	}

	return rb
}

// Level sets the location_types level param.
func (rb *RouteBuilder) Level(s string) *RouteBuilder {
	if s != "" {
		rb.v.Set("level", s)
	} else {
		rb.v.Del("level")
	}

	return rb
}

// DailyTimeIncrement sets whether time_increment should be 1.
func (rb *RouteBuilder) DailyTimeIncrement(b bool) *RouteBuilder {
	if b {
		rb.v.Set("time_increment", "1")
	} else {
		rb.v.Del("time_increment")
	}

	return rb
}

// ExportFormat sets the export_format level param.
func (rb *RouteBuilder) ExportFormat(s string) *RouteBuilder {
	if s != "" {
		rb.v.Set("export_format", s)
	} else {
		rb.v.Del("export_format")
	}

	return rb
}

// TimeRange sets time_range param and deletes the date_preset one.
func (rb *RouteBuilder) TimeRange(minDate, maxDate time.Time) *RouteBuilder {
	if minDate.IsZero() {
		rb.v.Del("time_range")
	} else {
		rb.v.Del("date_preset")
		if maxDate.IsZero() {
			maxDate = time.Now()
		}
		b, _ := json.Marshal(TimeRange{
			Since: minDate.Format("2006-01-02"),
			Until: maxDate.Format("2006-01-02"),
		})
		rb.v.Set("time_range", string(b))
	}

	return rb
}

// DatePreset sets date_preset param and deletes the time_range one.
func (rb *RouteBuilder) DatePreset(s string) *RouteBuilder {
	if s != "" {
		rb.v.Del("time_range")
		if s == "lifetime" {
			s = "maximum"
		}
		rb.v.Set("date_preset", s)
	} else {
		rb.v.Del("date_preset")
	}

	return rb
}

// DefaultSummary sets default_summary param or deletes it.
func (rb *RouteBuilder) DefaultSummary(t bool) *RouteBuilder {
	if t {
		rb.v.Set("default_summary", "true")
	} else {
		rb.v.Del("default_summary")
	}

	return rb
}

// UnifiedAttributionSettings sets unified_attribution_setting param or deletes it.
func (rb *RouteBuilder) UnifiedAttributionSettings(t bool) *RouteBuilder {
	if t {
		rb.v.Set("use_unified_attribution_setting", "true")
	} else {
		rb.v.Del("use_unified_attribution_setting")
	}

	return rb
}

// Filtering sets filtering param or deletes it.
func (rb *RouteBuilder) Filtering(f ...Filter) *RouteBuilder {
	if len(f) > 0 {
		b, err := json.Marshal(f)
		if err != nil {
			rb.err = err
		}

		rb.v.Set("filtering", string(b))
	} else {
		rb.v.Del("filtering")
	}

	return rb
}

// EffectiveStatus sets the effective_status param or deletes it.
func (rb *RouteBuilder) EffectiveStatus(s ...string) *RouteBuilder {
	if len(s) > 0 {
		rb.v.Set("effective_status", `["`+strings.Join(s, `","`)+`"]`)
	} else {
		rb.v.Del("effective_status")
	}

	return rb
}

// AdFormat sets the ad_format param or deletes it.
func (rb *RouteBuilder) AdFormat(s string) *RouteBuilder {
	if len(s) > 0 {
		rb.v.Set("ad_format", s)
	} else {
		rb.v.Del("ad_format")
	}

	return rb
}

// Metadata sets the ad_format param or deletes it.
func (rb *RouteBuilder) Metadata(t bool) *RouteBuilder {
	if t {
		rb.v.Set("metadata", "1")
	} else {
		rb.v.Del("metadata")
	}

	return rb
}

// Order sets the order param or deletes it.
func (rb *RouteBuilder) Order(s string) *RouteBuilder {
	if s != "" {
		rb.v.Set("order", s)
	} else {
		rb.v.Del("order")
	}

	return rb
}

// Filter sets the filter param or deletes it.
func (rb *RouteBuilder) Filter(s string) *RouteBuilder {
	if s != "" {
		rb.v.Set("filter", s)
	} else {
		rb.v.Del("filter")
	}

	return rb
}

// Summary sets the summary param or deletes it.
func (rb *RouteBuilder) Summary(s string) *RouteBuilder {
	if s != "" {
		rb.v.Set("summary", s)
	} else {
		rb.v.Del("summary")
	}

	return rb
}

// Q sets the q param or deletes it.
func (rb *RouteBuilder) Q(s string) *RouteBuilder {
	if s != "" {
		rb.v.Set("q", s)
	} else {
		rb.v.Del("q")
	}

	return rb
}

// Aggregation sets the aggregation param or deletes it.
func (rb *RouteBuilder) Aggregation(s string) *RouteBuilder {
	if s != "" {
		rb.v.Set("aggregation", s)
	} else {
		rb.v.Del("aggregation")
	}

	return rb
}

// ActionAttributionWindows sets the action_attribution_windows param or deletes it.
func (rb *RouteBuilder) ActionAttributionWindows(s ...string) *RouteBuilder {
	if len(s) > 0 {
		rb.v.Set("action_attribution_windows", strings.Join(s, ","))
	} else {
		rb.v.Del("action_attribution_windows")
	}

	return rb
}

// TargetingSpec sets the action_attribution_windows param or deletes it.
func (rb *RouteBuilder) TargetingSpec(ts interface{}) *RouteBuilder {
	b, _ := json.Marshal(ts)
	rb.v.Set("targeting_spec", string(b))

	return rb
}

// TargetingOptionList sets the targeting_option_list param or deletes it.
func (rb *RouteBuilder) TargetingOptionList(s ...string) *RouteBuilder {
	if len(s) > 0 {
		rb.v.Set("targeting_option_list", `["`+strings.Join(s, `","`)+`"]`)
	} else {
		rb.v.Del("targeting_option_list")
	}

	return rb
}

// String implements fmt.Stringer and returns the finished url.
func (rb *RouteBuilder) String() string {
	if rb.err != nil {
		return "err: " + rb.err.Error()
	}

	return (&url.URL{
		Scheme:   "https",
		Host:     "graph.facebook.com",
		Path:     "/" + rb.version + rb.path,
		RawQuery: (rb.v).Encode(),
	}).String()
}

// Filter is used for filtering lists.
type Filter struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}
