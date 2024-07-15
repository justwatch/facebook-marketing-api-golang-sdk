package meta

import (
	"context"
	"log"

	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
)

const (
	reachEstimateVersion = "v20.0"
)

type MetaService struct {
	log    *log.Logger
	client fb.Client
}

// GetAudienceSize returns the upper bound of the audience size.
// Uses Graph API v20.0.
func (ms MetaService) GetAudienceSize(ctx context.Context, accountID string, t *Targeting) (uint64, error) {
	r := fb.NewRoute(reachEstimateVersion, "/act_%s/reachestimate", accountID)

	if t != nil {
		r.TargetingSpec(t)
	}

	var reachEstimate ReachEstimate
	err := ms.client.GetJSON(ctx, r.String(), &reachEstimate)
	if err != nil {
		return 0, err
	}

	if !reachEstimate.Data.EstimateReady {
		return 0, nil
	}

	return uint64(reachEstimate.Data.UpperBound), nil
}

// Data is a single reach estimate.
type ReachEstimateData struct {
	LowerBound    int64 `json:"users_lower_bound"`
	UpperBound    int64 `json:"users_upper_bound"`
	EstimateReady bool  `json:"estimate_ready"`
}

// ReachEstimate is a collection of Data structs.
type ReachEstimate struct {
	Data ReachEstimateData `json:"data"`
}
