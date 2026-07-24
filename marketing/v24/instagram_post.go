package v24

import (
	"context"
	"net/url"

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

// ListOfInstagramUser returns an InstagramPostListCall for listing media of an IG user.
func (ps *PostService) ListOfInstagramUser(igUserID string) *InstagramPostListCall {
	return &InstagramPostListCall{
		RouteBuilder: fb.NewRoute(Version, "/%s/media", igUserID).Fields(instaPostFields...).Limit(100),
		c:            ps.c,
	}
}

// InstagramPostListCall is used for listing media posts of an IG user.
type InstagramPostListCall struct {
	*fb.RouteBuilder
	c *fb.Client
}

// Do calls the graph API and returns all matching posts as a slice.
func (ilc *InstagramPostListCall) Do(ctx context.Context) ([]InstagramPost, error) {
	res := []InstagramPost{}
	if err := ilc.c.GetList(ctx, ilc.RouteBuilder.String(), &res); err != nil {
		return nil, err
	}

	return res, nil
}

func (ps *PostService) GetInstagramPost(ctx context.Context, postID string) (*InstagramPost, error) {
	res := InstagramPost{}
	err := ps.c.GetJSON(ctx, fb.NewRoute(Version, "/%s", postID).Fields(instaPostFields...).String(), &res)
	if err != nil {
		return nil, err
	}

	return &res, nil
}

// GetInstagramPermalinksByMediaIDs returns permalinks keyed by media ID.
func (ps *PostService) GetInstagramPermalinksByMediaIDs(ctx context.Context, mediaIDs []string) (map[string]string, error) {
	const batchSize = 25

	relativeURLs := make([]string, len(mediaIDs))
	for i, mediaID := range mediaIDs {
		requestURL := url.URL{Path: mediaID}
		query := requestURL.Query()
		query.Set("fields", "permalink")
		requestURL.RawQuery = query.Encode()
		relativeURLs[i] = requestURL.String()
	}
	batchResponses, err := ps.getBatches(ctx, relativeURLs, batchSize, 4)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(mediaIDs))
	for i, batchResponse := range batchResponses {
		post := struct {
			Permalink string `json:"permalink"`
		}{}
		if err := decodeGraphBatchResponse(batchResponse, &post); err != nil {
			return nil, err
		}
		result[mediaIDs[i]] = post.Permalink
	}
	return result, nil
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

// ListInstagramCommentsByMediaIDs returns comments and replies grouped by media ID.
func (ps *PostService) ListInstagramCommentsByMediaIDs(ctx context.Context, mediaIDs []string) (map[string][]InstagramComment, error) {
	const batchSize = 25

	result := make(map[string][]InstagramComment, len(mediaIDs))
	for _, mediaID := range mediaIDs {
		result[mediaID] = []InstagramComment{}
	}

	relativeURLs := make([]string, len(mediaIDs))
	for i, mediaID := range mediaIDs {
		requestURL := url.URL{Path: mediaID}
		query := requestURL.Query()
		query.Set("fields", "comments.limit(1000){id,text,replies.limit(1000){id,text}}")
		requestURL.RawQuery = query.Encode()
		relativeURLs[i] = requestURL.String()
	}
	batchResponses, err := ps.getBatches(ctx, relativeURLs, batchSize, 4)
	if err != nil {
		return nil, err
	}
	for i, batchResponse := range batchResponses {
		post := struct {
			Comments instagramComments `json:"comments"`
		}{}
		if err := decodeGraphBatchResponse(batchResponse, &post); err != nil {
			return nil, err
		}
		result[mediaIDs[i]] = post.Comments.flatten()
	}

	return result, nil
}

type instagramCommentNode struct {
	InstagramComment
	Replies instagramComments `json:"replies"`
}

type instagramComments struct {
	Data []instagramCommentNode `json:"data"`
}

func (c instagramComments) flatten() []InstagramComment {
	result := []InstagramComment{}
	for _, comment := range c.Data {
		if comment.Text != "" {
			result = append(result, comment.InstagramComment)
		}
		result = append(result, comment.Replies.flatten()...)
	}
	return result
}
