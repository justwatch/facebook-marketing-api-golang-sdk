![jw-logo](logo.png)
# We are hiring ! 
# Facebook Marketing API sdk in Golang

<!-- [![Go reference](https://pkg.go.dev/https://github.com/justwatchcom/facebook-marketing-api-golang-sdk)](https://goreportcard.com/report/https://pkg.go.dev/https://github.com/justwatchcom/facebook-marketing-api-golang-sdk) -->

[![Go Report Card](https://goreportcard.com/badge/github.com/justwatchcom/facebook-marketing-api-golang-sdk)](https://goreportcard.com/report/github.com/justwatchcom/facebook-marketing-api-golang-sdk)
[![](https://godoc.org/github.com/justwatchcom/facebook-marketing-api-golang-sdk?status.svg)](http://godoc.org/github.com/justwatchcom/facebook-marketing-api-golang-sdk)

This go package provides a comprehensive list of methods for interacting with Facebook's Graph Marketing api.
The SDK supports crud operations for the following entities:

- ad_account
- adset
- custom_conversion
- insights
- post
- videos
- adcreative
- audience
- event
- interest
- search
- ads
- campaign
- image
- page
- service

## Usage

### Create a new fbService client

```
fbService, err := v12.New(l, accessToken, appSecret)
```

### Create a campaign

```
c := v12.Campaign{
		ID:                  string(scg.ExternalID),
		AccountID:           sa.ExternalID,
		Name:                scg.ExternalName,
		Status:              strings.ToUpper(scg.State),
		SpendCap:            uint64(spendCap),
		Objective:           strings.ToUpper(scg.Objective),
		CanUseSpendCap:      scg.CanUseSpendCap,
		BuyingType:          strings.ToUpper(scg.BudgetType),
		StartTime:           fb.Time(scg.StartsAt),
		StopTime:            fb.Time(scg.EndsAt),
		DailyBudget:         dailyBudget,
		LifeTimeBudget:      lifetimeBudget,
		BidStrategy:         bidStrategyType,
		SpecialAdCategories: []string{"NONE"},
	}
    id, err := fbService.Campaigns.Create(ctx, c)
    if err !=nil {
        return err
    }
```

### Upload an external asset to Facebook

```
file, err := os.Open("path to image")
// if err != nil { ... }
id := "account_id"
im ,err := fbService.Images.Upload(ctx, id, "my_pic", file)
// if err != nil {...}
// you have the ID here
im.ID
```

```
file, err := os.Open("path to video file")
// if err != nil { ... }
id := "account_id"
vid, err := fbService.Videos.Upload(ctx, id, "my_vid", file)
// if err != nil {...}

// you have the ID here
vid.ID
```

### Read campaigns from an account

```
id := "account_id"
campaigns, err := p.fbService.Campaigns.List(id).Do(ctx)
	if err != nil {

	}
```

### Get reporting data for an account at adset level

```
// put the columns you need for the report
columns := []string {}

//the fb ID for the account you want to the report
id := "account_id"


report := fbService.Insights.NewReport(id)
report.Level("adset").
DailyTimeIncrement(true). // get day by day reporting
Fields(columns...). // the fields you want your report to have
DatePreset("lifetime")// the time period for the report

// pass a channel where you get your results
ch := make(chan v12.Insight)
nRecords ,err := report.GenerateReport(ctx,ch)

//range over the channel to get Insight objects
for insight := range ch {
    //...
}
```
