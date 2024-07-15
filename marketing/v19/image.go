package v19

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"path"
	"regexp"

	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
	"golang.org/x/sync/errgroup"
)

var regexImageFilename = regexp.MustCompile(`^\d+_(\d+)_\d+_[a-z]\.([a-z]+)$`)

// ImageService works with ad images.
type ImageService struct {
	c *fb.Client
}

// ReadList writes all ad images from an account to res.
func (is *ImageService) ReadList(ctx context.Context, act string, res chan<- Image) error {
	jres := make(chan json.RawMessage)
	wg := errgroup.Group{}
	wg.Go(func() error {
		defer close(jres)

		return is.c.ReadList(ctx, fb.NewRoute(Version, "/act_%s/adimages", act).Fields("name", "hash", "url", "width", "height").Limit(1000).String(), jres)
	})
	wg.Go(func() error {
		for e := range jres {
			v := Image{}
			err := json.Unmarshal(e, &v)
			if err != nil {
				return err
			}

			v.ID, err = is.getImageID(v.URL)
			if err != nil {
				return err
			}

			res <- v
		}

		return nil
	})

	return wg.Wait()
}

// Upload uploads an image to Facebook.
func (is *ImageService) Upload(ctx context.Context, act, name string, r io.Reader) (*Image, error) {
	fur := &fileUploadResponse{}
	err := is.c.UploadFile(ctx, fb.NewRoute(Version, "/act_%s/adimages", act).String(), name, r, nil, fur)
	if err != nil {
		return nil, err
	}

	im := fur.Images[name]
	if im == nil {
		return nil, errors.New("did not get image metadata response")
	}

	id, err := is.getImageID(im.URL)
	if err != nil {
		return nil, err
	}

	im.ID = id

	return im, nil
}

func (is *ImageService) getImageID(s string) (string, error) {
	parsed, err := url.Parse(s)
	if err != nil {
		return "", err
	}
	_, file := path.Split(parsed.Path)
	parts := regexImageFilename.FindStringSubmatch(file)
	if len(parts) < 3 {
		return "", nil
	}

	return parts[1], nil
}

// Image represents an image being uploaded to the creative library.
type Image struct {
	ID                              string   `json:"id,omitempty"`
	Height                          int      `json:"height,omitempty"`
	Hash                            string   `json:"hash,omitempty"`
	IsAssociatedCreativesInAdgroups bool     `json:"is_associated_creatives_in_adgroups,omitempty"`
	Name                            string   `json:"name,omitempty"`
	OriginalHeight                  int      `json:"original_height,omitempty"`
	OriginalWidth                   int      `json:"original_width,omitempty"`
	PermalinkURL                    string   `json:"permalink_url,omitempty"`
	Status                          string   `json:"status,omitempty"`
	UpdatedTime                     string   `json:"updated_time,omitempty"`
	CreatedTime                     string   `json:"created_time,omitempty"`
	URL                             string   `json:"url,omitempty"`
	AccountID                       string   `json:"account_id,omitempty"`
	URL128                          string   `json:"url_128,omitempty"`
	Width                           int      `json:"width,omitempty"`
	Creatives                       []string `json:"creatives,omitempty"`
}

type fileUploadResponse struct {
	Images map[string]*Image `json:"images"`
}
