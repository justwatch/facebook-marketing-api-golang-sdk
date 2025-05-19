package v22

import (
	"context"
	"fmt"

	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
)

// PageService contains all methods for working on pages.
type PageService struct {
	c *fb.Client
}

// SetPageAccessToken tries to retrieve the access token for a facebook page and includes it in the passed context so the fb.Client can use it for making requests.
func (ps *PageService) SetPageAccessToken(ctx context.Context, pageID string) (context.Context, error) {
	tc := struct {
		AccessToken string `json:"access_token"`
	}{}
	err := ps.c.GetJSON(ctx, fb.NewRoute(Version, "/%s", pageID).Fields("access_token").String(), &tc)
	if err != nil {
		return ctx, err
	} else if tc.AccessToken == "" {
		return ctx, fmt.Errorf("could not get page access token for '%s'", pageID)
	}

	return fb.SetPageAccessToken(ctx, tc.AccessToken), nil
}

// GetPageBackedInstagramAccounts returns the instagram actor associated with a facebook page.
func (ps *PageService) GetInstagramBusinessAccount(ctx context.Context, pageID string) (*InstagramUser, error) {
	ctx, err := ps.SetPageAccessToken(ctx, pageID)
	if err != nil {
		return nil, err
	}

	fpiga := struct {
		InstagramBusinessAccount InstagramUser `json:"instagram_business_account"`
	}{}
	err = ps.c.GetJSON(ctx, fb.NewRoute(Version, "/%s", pageID).Fields("instagram_business_account{id,username}").String(), &fpiga)
	if err != nil {
		return nil, err
	}

	res := fpiga.InstagramBusinessAccount
	if res.ID == "" {
		return nil, fmt.Errorf("could not get page_backed_instagram_accounts ID for facebook page with external id %s", pageID)
	}
	if res.Username == "" {
		return nil, fmt.Errorf("could not get page_backed_instagram_accounts username for facebook page with external id %s", pageID)
	}

	return &res, nil
}

// GetClientPages returns all client pages.
func (ps *PageService) GetClientPages(ctx context.Context, businessID string) ([]Page, error) {
	res := []Page{}
	route := fb.NewRoute(Version, "/%s/client_pages", businessID).Limit(1000).Fields(pageFields...)
	err := ps.c.GetList(ctx, route.String(), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// GetOwnedPages returns all owned pages.
func (ps *PageService) GetOwnedPages(ctx context.Context, businessID string) ([]Page, error) {
	res := []Page{}
	route := fb.NewRoute(Version, "/%s/owned_pages", businessID).Limit(1000).Fields(pageFields...)
	err := ps.c.GetList(ctx, route.String(), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// GetInstagramUsers returns all instagram accounts.
func (ps *PageService) GetInstagramUsers(ctx context.Context, businessID string) ([]InstagramUser, error) {
	type Page struct {
		ID string `json:"id"`
	}
	var pages []Page
	pageRoute := fb.NewRoute(Version, "/%s/owned_pages", businessID).Fields("id").Limit(100)
	if err := ps.c.GetList(ctx, pageRoute.String(), &pages); err != nil {
		return nil, err
	}

	var igUsers []InstagramUser
	for _, page := range pages {
		var wrapper struct {
			InstagramAccount *InstagramUser `json:"instagram_business_account"`
		}
		igRoute := fb.NewRoute(Version, "/%s", page.ID).Fields("instagram_business_account{id,username}")
		if err := ps.c.GetJSON(ctx, igRoute.String(), &wrapper); err != nil {
			continue
		}
		if wrapper.InstagramAccount != nil {
			igUsers = append(igUsers, *wrapper.InstagramAccount)
		}
	}

	return igUsers, nil
}

func (ps *PageService) ListIGUsers(ctx context.Context) ([]InstagramUser, error) {
	type Page struct {
		ID string `json:"id"`
	}
	var pages []Page
	pageRoute := fb.NewRoute(Version, "/me/accounts").Fields("id").Limit(100)
	if err := ps.c.GetList(ctx, pageRoute.String(), &pages); err != nil {
		return nil, err
	}

	// Step 2: Get IG account from each page
	var igUsers []InstagramUser
	for _, page := range pages {
		var wrapper struct {
			InstagramAccount *InstagramUser `json:"instagram_business_account"`
		}
		igRoute := fb.NewRoute(Version, "/%s", page.ID).Fields("instagram_business_account{id,username}")
		if err := ps.c.GetJSON(ctx, igRoute.String(), &wrapper); err != nil {
			continue
		}
		if wrapper.InstagramAccount != nil {
			igUsers = append(igUsers, *wrapper.InstagramAccount)
		}
	}

	return igUsers, nil
}

// Get returns a single page.
func (ps *PageService) Get(ctx context.Context, id string) (*Page, error) {
	res := &Page{}
	route := fb.NewRoute(Version, "/%s", id).Fields(pageFields...)
	err := ps.c.GetJSON(ctx, route.String(), res)
	if err != nil {
		if fb.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return res, nil
}

// GetInstagramUser returns a single instagram user.
func (ps *PageService) GetInstagramUser(ctx context.Context, id string) (*InstagramUser, error) {
	res := &InstagramUser{}
	route := fb.NewRoute(Version, "/%s", id).Fields(instagramUserFields...)
	err := ps.c.GetJSON(ctx, route.String(), res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

var (
	pageFields          = []string{"id", "global_brand_page_name"}
	instagramUserFields = []string{"id", "username"}
)

// Page represents a facebook page.
type Page struct {
	ID                  string `json:"id"`
	GlobalBrandPageName string `json:"global_brand_page_name"`
}

// InstagramActor represents an instagram actor.
type InstagramUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}
