package v22

import (
	"context"

	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
)

type InstagramPost struct {
	ID        string `json:"id,omitempty"`
	IgID      string `json:"ig_id,omitempty"`
	Shortcode string `json:"shortcode,omitempty"`
	Permalink string `json:"permalink,omitempty"`
	Owner     struct {
		ID string `json:"id,omitempty"`
	} `json:"owner,omitempty"`
	Caption              string `json:"caption,omitempty"`
	LikeCount            uint64 `json:"like_count,omitempty"`
	CommentsCount        uint64 `json:"comments_count,omitempty"`
	MediaType            string `json:"media_type,omitempty"`
	BoostEligibilityInfo struct {
		EligibleToBoost bool `json:"eligible_to_boost,omitempty"`
	} `json:"boost_eligibility_info,omitempty"`
}

var instaPostFields = []string{"id", "ig_id", "shortcode", "permalink", "owner", "boost_eligibility_info", "like_count", "comments_count", "media_type", "caption"}

// GetClientPages returns all client pages.
func (ps *PostService) ListInstagramPosts(ctx context.Context, igUserID string, c chan<- InstagramPost) (uint64, error) {
	defer close(c)
	url := fb.NewRoute(Version, "/%s/media", igUserID).Limit(100).Fields(instaPostFields...).String()
	var count uint64
	for url != "" {
		resp := &struct {
			fb.Paging
			Data []InstagramPost `json:"data"`
		}{}

		err := ps.c.GetJSON(ctx, url, resp)
		if err != nil {
			return 0, err
		}

		for _, d := range resp.Data {
			count++
			c <- d
		}
		url = resp.Paging.Paging.Next
	}

	return count, nil
}

func (ps *PostService) GetInstagramPost(ctx context.Context, postID string) (*InstagramPost, error) {
	res := InstagramPost{}
	err := ps.c.GetJSON(ctx, fb.NewRoute(Version, "/%s", postID).Fields(instaPostFields...).String(), &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

type InstagramComment struct {
	ID        string  `json:"id,omitempty"`
	Text      string  `json:"text,omitempty"`
	LikeCount uint64  `json:"like_count,omitempty"`
	Timestamp fb.Time `json:"timestamp,omitempty"`
}

var instaCommentFields = []string{"id", "text", "like_count", "timestamp"}

func (ps *PostService) ListInstagramComments(ctx context.Context, postID string, c chan<- InstagramComment) (uint64, error) {
	defer close(c)
	url := fb.NewRoute(Version, "/%s/comments", postID).Limit(50).Fields(instaCommentFields...).String()
	var count uint64
	for url != "" {
		resp := &struct {
			fb.Paging
			Data []InstagramComment `json:"data"`
		}{}

		err := ps.c.GetJSON(ctx, url, resp)
		if err != nil {
			return 0, err
		}

		for _, d := range resp.Data {
			count++
			c <- d
		}
		url = resp.Paging.Paging.Next
	}

	return count, nil
}
