package v19

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
)

// BatchMaxIDsSequence we upload.
const BatchMaxIDsSequence = 10000

// AudienceService contains all methods for working on audiences.
type AudienceService struct {
	c *fb.Client
}

// Create uploads a new custom audience and returns the id of the custom audience.
func (as *AudienceService) Create(ctx context.Context, act string, a CustomAudience) (string, error) {
	if a.ID != "" {
		return "", fmt.Errorf("cannot create audience that already exists: %s", a.ID)
	} else if act == "" {
		return "", errors.New("cannot create audience without account id")
	}

	res := &fb.MinimalResponse{}
	err := as.c.PostJSON(ctx, fb.NewRoute(Version, "/act_%s/customaudiences", act).String(), a, res)
	if err != nil {
		return "", err
	} else if err = res.GetError(); err != nil {
		return "", err
	} else if res.ID == "" {
		return "", fmt.Errorf("creating custom audience failed")
	}

	return res.ID, nil
}

// CreateLookalike creates new lookalike
func (as *AudienceService) CreateLookalike(ctx context.Context, adaccountID, orginAudienceID, customAudienceName string, lookalikeSpec *LookalikeSpec) (string, error) {

	type createLookalikeRequest struct {
		OriginAudienceID string         `json:"origin_audience_id"`
		Name             string         `json:"name"`
		Subtype          string         `json:"subtype"`
		LookalikeSpec    *LookalikeSpec `json:"lookalike_spec"`
	}

	res := &fb.MinimalResponse{}
	err := as.c.PostJSON(ctx, fb.NewRoute(Version, "/act_%s/customaudiences", adaccountID).String(), createLookalikeRequest{
		OriginAudienceID: orginAudienceID,
		Name:             customAudienceName,
		Subtype:          "LOOKALIKE",
		LookalikeSpec:    lookalikeSpec,
	}, &res)
	if err != nil {
		return "", err
	} else if err = res.GetError(); err != nil {
		return "", err
	} else if res.ID == "" {
		return "", fmt.Errorf("creating lookalike audience failed")
	}

	return res.ID, nil
}

// Update updates an audience.
func (as *AudienceService) Update(ctx context.Context, a CustomAudience) error {
	if a.ID == "" {
		return errors.New("cannot update audience without id")
	}

	res := &fb.MinimalResponse{}
	err := as.c.PostJSON(ctx, fb.NewRoute(Version, "/%s", a.ID).String(), a, res)
	if err != nil {
		return err
	} else if err = res.GetError(); err != nil {
		return err
	} else if !res.Success && res.ID == "" {
		return fmt.Errorf("updating the audience failed")
	}

	return nil
}

// Share shares a custom audience with the provided adaccounts.
func (as *AudienceService) Share(ctx context.Context, customAudienceID string, adaccountIDs []string) error {
	if len(adaccountIDs) == 0 {
		return nil
	}
	ca, err := as.Get(ctx, customAudienceID)
	if err != nil {
		return err
	}

	existingAdaccountIDs := []json.Number{}
	if ca.AccountID != "" {
		existingAdaccountIDs = append(existingAdaccountIDs, json.Number(ca.AccountID))
	}
	adaccountsData := []json.Number{}
	if ca.Adaccounts != nil && ca.Adaccounts.Data != nil {
		for _, adaccountID := range ca.Adaccounts.Data {
			existingAdaccountIDs = append(existingAdaccountIDs, adaccountID)
			adaccountsData = append(adaccountsData, adaccountID)
		}
	}

	changed := false
	for _, adaccountIDToShare := range adaccountIDs {
		found := false
		for _, existingAdAccountID := range existingAdaccountIDs {
			if existingAdAccountID == json.Number(adaccountIDToShare) {
				found = true

				break
			}
		}
		if !found {
			adaccountsData = append(adaccountsData, json.Number(adaccountIDToShare))
			changed = true
		}
	}

	if changed {
		ca.Adaccounts = &Adaccounts{
			Data: adaccountsData,
		}
	}

	return as.Update(ctx, *ca)
}

// ShareCustom shares a custom audience with the provided adaccounts.
func (as *AudienceService) ShareCustom(ctx context.Context, customAudienceID string, adaccountIDs, relationshipTypes []string) error {
	if len(adaccountIDs) == 0 {
		return nil
	}

	return as.c.PostJSON(ctx, fb.NewRoute(Version, "/%s/adaccounts", customAudienceID).String(), struct {
		Adaccounts       []string `json:"adaccounts"`
		RelationshipType []string `json:"relationship_type"`
	}{adaccountIDs, relationshipTypes}, &struct{}{})
}

// UnshareCustom unshares a custom audience with the provided adaccounts.
func (as *AudienceService) UnshareCustom(ctx context.Context, customAudienceID string, adaccountIDs, relationshipTypes []string) error {
	if len(adaccountIDs) == 0 {
		return nil
	}

	return as.c.DeleteJSON(ctx, fb.NewRoute(Version, "/%s/adaccounts", customAudienceID).String(), struct {
		Adaccounts       []string `json:"adaccounts"`
		RelationshipType []string `json:"relationship_type"`
	}{adaccountIDs, relationshipTypes}, &struct{}{})
}

// ListAdAccounts lists the accounts an audience is shared to.
func (as *AudienceService) ListAdAccounts(ctx context.Context, audienceID string) ([]string, error) {
	res := struct {
		Data []string `json:"data"`
	}{}
	err := as.c.GetJSON(ctx, fb.NewRoute(Version, "/%s/adaccounts", audienceID).String(), &res)
	if err != nil {
		return nil, err
	}

	return res.Data, nil
}

// Delete removes a single audience.
func (as *AudienceService) Delete(ctx context.Context, id string) error {
	return as.c.Delete(ctx, fb.NewRoute(Version, "/%s", id).String())
}

// DeleteLookalikes removes all lookalikes of an audience.
func (as *AudienceService) DeleteLookalikes(ctx context.Context, id string) error {
	ca, err := as.Get(ctx, id)
	if err != nil {
		return err
	} else if ca == nil {
		return fmt.Errorf("did not find custom audience %s", id)
	}

	for _, id := range ca.Lookalikes {
		err = as.Delete(ctx, id)
		if err != nil {
			return err
		}
	}

	return nil
}

// Get returns a single audience.
func (as *AudienceService) Get(ctx context.Context, id string) (*CustomAudience, error) {
	res := &CustomAudience{}
	err := as.c.GetJSON(ctx, fb.NewRoute(Version, "/%s", id).Fields("id", "name", "description", "subtype", "approximate_count_upper_bound", "approximate_count_lower_bound", "rule", "customer_file_source", "lookalike_audience_ids", "adaccounts").String(), res)
	if err != nil {
		if fb.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return res, nil
}

// ListCustom returns all custom audiences for that account.
func (as *AudienceService) ListCustom(ctx context.Context, act string) ([]CustomAudience, error) {
	res := []CustomAudience{}
	route := fb.NewRoute(Version, "/act_%s/customaudiences", act).
		Limit(250).
		Fields("id", "name", "description", "approximate_count_upper_bound", "approximate_count_lower_bound", "subtype", "adaccounts", "lookalike_spec") // , "rule")
	err := as.c.GetList(ctx, route.String(), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ListCustomFiltered returns a filtered list of custom audiences for that account.
// ...&filtering=[{field:'subtype',operator:'EQUAL',value:'WEBSITE'}].
func (as *AudienceService) ListCustomFiltered(ctx context.Context, act string, filtering []fb.Filter) ([]CustomAudience, error) {
	res := []CustomAudience{}
	route := fb.NewRoute(Version, "/act_%s/customaudiences", act).
		Limit(250).
		Fields("id", "name", "account_id", "description", "approximate_count_upper_bound", "approximate_count_lower_bound", "subtype", "adaccounts"). // , "rule")
		Filtering(filtering...)
	err := as.c.GetList(ctx, route.String(), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// EditIDs starts adding or removing ids from a custom audience.
func (as *AudienceService) EditIDs(ctx context.Context, audienceID string, c <-chan string, doRemove bool) error {
	bigN, err := rand.Int(rand.Reader, big.NewInt(math.MaxUint32))
	if err != nil {
		return fmt.Errorf("failed to generate session ID int EditIDs: %w", err)
	}
	sessionID := bigN.Int64()
	doWork := true
	var total, received, failed uint64
	var leftOver string
	for batchSequence := 1; doWork; batchSequence++ {
		var ids []string
		ids, leftOver, doWork = readBatch(BatchMaxIDsSequence, c, leftOver)
		if len(ids) == 0 {
			break
		}
		total += uint64(len(ids))

		route := fb.NewRoute(Version, "/%s/users", audienceID).String()
		req := editAudienceIDsRequest{
			Session: uploadSession{
				SessionID:     uint32(sessionID),
				BatchSequence: batchSequence,
				LastBatchFlag: !doWork,
			},
			Payload: uploadPayload{
				Schema: "MOBILE_ADVERTISER_ID",
				Data:   ids,
			},
		}
		res := &editAudienceIDsResponse{}
		var err error
		if doRemove {
			err = as.c.DeleteJSON(ctx, route, req, res)
		} else {
			err = as.c.PostJSON(ctx, route, req, res)
		}
		if err != nil {
			return err
		}
		received = res.NumReceived
		failed = res.NumInvalidEntries
	}
	if total != received {
		return &UploadError{
			Total:    total,
			Received: received,
			Failed:   failed,
		}
	}

	return nil
}

func readBatch(max int, c <-chan string, leftOver string) ([]string, string, bool) {
	s := make([]string, 0, max)
	ok := true
	index := 0
	if leftOver != "" {
		s = append(s, leftOver)
		leftOver = ""
		index++
	}
	for ; index < max && ok; index++ {
		var id string
		id, ok = read(c)
		if id == "" {
			index--

			continue
		}
		s = append(s, id)
	}
	if len(s) == max {
		leftOver, ok = read(c)
	}

	return s, leftOver, ok
}

func read(c <-chan string) (string, bool) {
	var res string
	ok := true
	for ok && res == "" {
		res, ok = <-c
	}

	return res, ok
}

type editAudienceIDsRequest struct {
	Session uploadSession `json:"session"`
	Payload uploadPayload `json:"payload"`
}

type uploadSession struct {
	SessionID     uint32 `json:"session_id"`
	BatchSequence int    `json:"batch_seq"`
	LastBatchFlag bool   `json:"last_batch_flag"`
}

type uploadPayload struct {
	Schema string   `json:"schema"`
	Data   []string `json:"data"`
}

type editAudienceIDsResponse struct {
	UserSegmentID     uint64 `json:"user_segment_id"`
	SessionID         string `json:"session_id"`
	NumReceived       uint64 `json:"num_received"`
	NumInvalidEntries uint64 `json:"num_invalid_entries"`
}

// UploadError gets returned when the number of total lines does not match the number of received lines or when the number of failed lines is greater than zero.
type UploadError struct {
	Total    uint64
	Received uint64
	Failed   uint64
}

func (ue *UploadError) Error() string {
	return fmt.Sprintf("uploaded %d ids, received %d, failed uploading %d", ue.Total, ue.Received, ue.Failed)
}

// CustomAudience https://developers.facebook.com/docs/marketing-api/reference/custom-audience
// unused fields: ApproximateCount   int            `json:"approximate_count,omitempty"`.
type CustomAudience struct {
	ID                         string `json:"id,omitempty"`
	Name                       string `json:"name,omitempty"`
	AccountID                  string `json:"account_id,omitempty"`
	Description                string `json:"description,omitempty"`
	Subtype                    string `json:"subtype,omitempty"`
	ApproximateCountUpperBound int    `json:"approximate_count_upper_bound,omitempty"`
	ApproximateCountLowerBound int    `json:"approximate_count_lower_bound,omitempty"`

	Rule               string         `json:"rule,omitempty"`
	CustomerFileSource string         `json:"customer_file_source,omitempty"`
	Lookalikes         []string       `json:"lookalike_audience_ids,omitempty"`
	Adaccounts         *Adaccounts    `json:"adaccounts,omitempty"`
	LookalikeSpec      *LookalikeSpec `json:"lookalike_spec,omitempty"`
	OriginAudienceID   string         `json:"origin_audience_id,omitempty"`
}

// LookalikeSpec contains the metadata of lookalike audiences.
type LookalikeSpec struct {
	Country string             `json:"country,omitempty"`
	Origin  []LookalikeOrigion `json:"origin,omitempty"`
	Ratio   float64            `json:"ratio,omitempty"`
	Type    string             `json:"type,omitempty"`
}

// LocationSpec ...
type LocationSpec struct {
	GeoLocations         *GeoLocation `json:"geo_locations,omitempty"`
	ExcludedGeoLocations *GeoLocation `json:"excluded_geo_locations,omitempty"`
}

// GeoLocation ...
type GeoLocation struct {
	Countries     []string `json:"countries,omitempty"`
	CountryGroups []string `json:"country_groups,omitempty"`
}

// LookalikeOrigion tells which audience a lookalike one is related to.
type LookalikeOrigion struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// Adaccounts https://developers.facebook.com/docs/marketing-api/reference/custom-audience/adaccounts/
type Adaccounts struct {
	Data []json.Number `json:"data,omitempty"`
}
