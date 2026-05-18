package v24

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
	"golang.org/x/sync/errgroup"
)

// VideoService works with advideos.
type VideoService struct {
	c *fb.Client
}

// Get returns a single Video.
func (vs *VideoService) Get(ctx context.Context, id string) (*Video, error) {
	res := &Video{}
	err := vs.c.GetJSON(ctx, fb.NewRoute(Version, "/%s", id).Fields(advideoFields...).String(), res)
	if err != nil {
		if fb.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return res, nil
}

// Upload uploads a video from r into an account and returns the video with the
// fields listed in advideoFields populated.
//
// The trailing Get happens immediately after the upload finishes, while Meta
// is still asynchronously rendering format variants. Requesting `format` (and
// to a lesser extent `picture`) at that moment can trip Meta's response-size
// guard and return "Please reduce the amount of data you're asking for".
// Callers that only need the resulting video ID should prefer
// UploadWithoutFetch.
func (vs *VideoService) Upload(ctx context.Context, act, title string, size int64, r io.Reader) (*Video, error) {
	videoID, err := vs.uploadChunks(ctx, act, title, size, r)
	if err != nil {
		return nil, err
	}

	return vs.Get(ctx, videoID)
}

// UploadWithoutFetch uploads a video from r into an account and returns a
// minimally populated Video containing only the resulting ID. It skips the
// trailing fields fetch that Upload performs, avoiding "Please reduce the
// amount of data you're asking for" errors that Meta intermittently returns
// while a freshly uploaded video is still being transcoded.
//
// Use this when the caller only needs the video ID (e.g. to attach the asset
// to a creative). Callers that need title/picture/format/etc. should either
// use Upload or call Get separately, after Meta has had time to finish
// encoding.
func (vs *VideoService) UploadWithoutFetch(ctx context.Context, act, title string, size int64, r io.Reader) (*Video, error) {
	videoID, err := vs.uploadChunks(ctx, act, title, size, r)
	if err != nil {
		return nil, err
	}

	return &Video{ID: videoID}, nil
}

// uploadChunks runs the start/transfer/finish chunked-upload protocol and
// returns the resulting video ID. Meta returns video_id in the start-phase
// response, so we already have it before the chunked transfer even begins.
func (vs *VideoService) uploadChunks(ctx context.Context, act, title string, size int64, r io.Reader) (string, error) {
	url := fb.NewRoute(Version, "/act_%s/advideos", act).String()

	res := uploadVideoResponse{}
	err := vs.c.PostJSON(ctx, url, uploadVideoRequestStart{
		UploadPhase: "start",
		FileSize:    size,
	}, &res)
	if err != nil {
		return "", err
	}

	for size > 0 {
		chunksize := res.EndOffset - res.StartOffset
		if chunksize > size {
			chunksize = size
		}
		size -= chunksize
		err := vs.c.UploadFile(ctx, url, title, io.LimitReader(r, chunksize), map[string]string{
			"upload_phase":      "transfer",
			"upload_session_id": res.UploadSessionID,
			"start_offset":      fmt.Sprintf("%d", res.StartOffset),
		}, &res)
		if err != nil {
			return "", err
		}
	}

	fr := finishResponse{}
	err = vs.c.PostJSON(ctx, url, uploadVideoRequestEnd{
		UploadPhase:     "finish",
		UploadSessionID: res.UploadSessionID,
		Title:           title,
	}, &fr)
	if err != nil {
		return "", err
	}

	return res.VideoID, nil
}

// ReadList returns all videos from an account and writes them to a channel.
func (vs *VideoService) ReadList(ctx context.Context, act string, res chan<- Video) error {
	jres := make(chan json.RawMessage)
	wg := errgroup.Group{}
	wg.Go(func() error {
		defer close(jres)

		return vs.c.ReadList(ctx, fb.NewRoute(Version, "/act_%s/advideos", act).Fields(advideoFields...).Limit(200).String(), jres)
	})
	wg.Go(func() error {
		for e := range jres {
			v := Video{}
			err := json.Unmarshal(e, &v)
			if err != nil {
				return err
			}
			res <- v
		}

		return nil
	})

	return wg.Wait()
}

// Thumbnails returns the thumbnails edge for a video
// (GET /{video_id}/thumbnails). Unlike Video.Picture — which can return
// Facebook's generic placeholder while the video is still processing — this
// endpoint returns extracted frames with an explicit is_preferred flag.
func (vs *VideoService) Thumbnails(ctx context.Context, videoID string) ([]VideoThumbnail, error) {
	res := &videoThumbnailsResponse{}
	err := vs.c.GetJSON(
		ctx,
		fb.NewRoute(Version, "/%s/thumbnails", videoID).Fields(videoThumbnailFields...).String(),
		res,
	)
	if err != nil {
		if fb.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return res.Data, nil
}

var advideoFields = []string{"title", "id", "picture", "description", "from", "format", "length", "status"}

var videoThumbnailFields = []string{"id", "uri", "is_preferred", "height", "width", "scale"}

// VideoThumbnail represents a single entry on the /{video_id}/thumbnails edge.
type VideoThumbnail struct {
	ID          string  `json:"id"`
	URI         string  `json:"uri"`
	IsPreferred bool    `json:"is_preferred"`
	Height      int     `json:"height"`
	Width       int     `json:"width"`
	Scale       float64 `json:"scale"`
}

type videoThumbnailsResponse struct {
	Data []VideoThumbnail `json:"data"`
}

type uploadVideoRequestStart struct {
	UploadPhase string `json:"upload_phase"`
	FileSize    int64  `json:"file_size"`
}

type uploadVideoRequestEnd struct {
	UploadPhase     string `json:"upload_phase"`
	UploadSessionID string `json:"upload_session_id"`
	Title           string `json:"title"`
}

type uploadVideoResponse struct {
	UploadSessionID string `json:"upload_session_id"`
	VideoID         string `json:"video_id"`
	StartOffset     int64  `json:"start_offset,string"`
	EndOffset       int64  `json:"end_offset,string"`
}

type finishResponse struct {
	Success bool `json:"success"`
}

// Video represents an ad video.
type Video struct {
	ContentCategory        string  `json:"content_category"`
	CreatedTime            string  `json:"created_time"`
	Description            string  `json:"description"`
	EmbedHTML              string  `json:"embed_html"`
	Embeddable             bool    `json:"embeddable"`
	ID                     string  `json:"id"`
	Icon                   string  `json:"icon"`
	Length                 float64 `json:"length"`
	MonetizationStatus     string  `json:"monetization_status"`
	Picture                string  `json:"picture"`
	IsCrosspostVideo       bool    `json:"is_crosspost_video"`
	IsCrosspostingEligible bool    `json:"is_crossposting_eligible"`
	IsInstagramEligible    bool    `json:"is_instagram_eligible"`
	PermalinkURL           string  `json:"permalink_url"`
	Published              bool    `json:"published"`
	Source                 string  `json:"source"`
	UpdatedTime            string  `json:"updated_time"`
	Title                  string  `json:"title,omitempty"`
	AutoGeneratedCaptions  struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
		Paging struct {
			Cursors struct {
				Before string `json:"before"`
				After  string `json:"after"`
			} `json:"cursors"`
		} `json:"paging"`
	} `json:"auto_generated_captions,omitempty"`
	Format []struct {
		EmbedHTML string `json:"embed_html"`
		Filter    string `json:"filter"`
		Height    int    `json:"height"`
		Picture   string `json:"picture"`
		Width     int    `json:"width"`
	} `json:"format"`
	From struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"from"`
	Privacy struct {
		Allow       string `json:"allow"`
		Deny        string `json:"deny"`
		Description string `json:"description"`
		Friends     string `json:"friends"`
		Networks    string `json:"networks"`
		Value       string `json:"value"`
	} `json:"privacy"`
	Status struct {
		VideoStatus string `json:"video_status"`
	} `json:"status"`
}
