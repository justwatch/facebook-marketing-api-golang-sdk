package v19

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/justwatch/facebook-marketing-api-golang-sdk/fb"
	"golang.org/x/sync/errgroup"
)

const (
	adCreativeReadListLimit = 50
)

// AdCreativeService works on adcreatives.
type AdCreativeService struct {
	c *fb.Client
	*fb.StatsContainer
}

// Get return a single creative.
func (as *AdCreativeService) Get(ctx context.Context, id string) (*AdCreative, error) {
	res := &AdCreative{}
	err := as.c.GetJSON(ctx, fb.NewRoute(Version, "/%s", id).Fields(Adcreativefields...).String(), res)
	if err != nil {
		if fb.IsNotFound(err) {
			return nil, nil
		}

		return nil, err
	}

	return res, nil
}

// Create uploads a new adcreative and returns the ID.
func (as *AdCreativeService) Create(ctx context.Context, a AdCreative) (string, string, error) {
	if a.ID != "" {
		return "", "", fmt.Errorf("cannot create adcreative that already exists: %s", a.ID)
	} else if a.AccountID == "" {
		return "", "", errors.New("cannot create adcreative without account id")
	}

	// The enroll_status parameter for Standard Enhancements is now required for eligible ad creation requests.
	if a.DegreesOfFreedomSpec == nil || a.DegreesOfFreedomSpec.CreativeFeaturesSpec.StandardEnhancements.EnrollStatus == "" {
		a.DegreesOfFreedomSpec = &DegreesOfFreedomSpec{
			CreativeFeaturesSpec: CreativeFeaturesSpec{
				StandardEnhancements: StandardEnhancements{
					EnrollStatus: "OPT_OUT",
				},
			},
		}
	}

	res := struct {
		fb.ID
		fb.ErrorContainer
		EffectiveObjectStoryID string `json:"effective_object_story_id"`
	}{}
	err := as.c.PostJSON(ctx, fb.NewRoute(Version, "/act_%s/adcreatives", a.AccountID).Fields("id", "effective_object_story_id").String(), a, &res)
	if err != nil {
		return "", "", err
	} else if err = res.GetError(); err != nil {
		return "", "", err
	} else if res.ID.ID == "" {
		return "", "", fmt.Errorf("creating adcreative failed")
	}

	return res.ID.ID, res.EffectiveObjectStoryID, nil
}

// GetPreviewURL returns the preview URL of a creative.
func (as *AdCreativeService) GetPreviewURL(ctx context.Context, id, format string) (string, error) {
	b := []struct {
		Body string `json:"body"`
	}{}
	err := as.c.GetList(ctx, fb.NewRoute(Version, "/%s/previews", id).AdFormat(format).String(), &b)
	if err != nil {
		return "", err
	} else if len(b) != 1 {
		return "", fmt.Errorf("expected one preview for external creative '%s', got %d", id, len(b))
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(b[0].Body))
	if err != nil {
		return "", err
	}

	link, ok := doc.Find("iframe").First().Attr("src")
	if !ok {
		return "", errors.New("did not find iframe")
	}

	return link, nil
}

type AdCreativeListCall struct {
	*fb.RouteBuilder
	c *fb.Client
}

func (s *AdCreativeService) List(act string, fields []string) *AdCreativeListCall {
	if len(fields) == 0 {
		fields = Adcreativefields
	}
	return &AdCreativeListCall{
		c:            s.c,
		RouteBuilder: fb.NewRoute(Version, "/act_%s/ads", act).Limit(adCreativeReadListLimit).Fields(fmt.Sprintf("adcreatives{%v}", strings.Join(fields, ","))),
	}
}

// Do calls the graph API.
func (s *AdCreativeListCall) Do(ctx context.Context) ([]AdCreative, error) {
	res := []AdCreative{}
	err := s.c.GetList(ctx, s.RouteBuilder.String(), &res)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ReadList writes all adcreatives from an account to res.
func (s *AdCreativeListCall) ReadList(ctx context.Context, act string, res chan<- AdCreative) error {
	jres := make(chan json.RawMessage)
	wg := errgroup.Group{}
	wg.Go(func() error {
		defer close(jres)
		return s.c.ReadList(ctx, s.RouteBuilder.String(), jres)
	})
	wg.Go(func() error {
		for e := range jres {
			v := adCreativeContainer{}
			err := json.Unmarshal(e, &v)
			if err != nil {
				return err
			}
			for i := range v.Adcreatives.Data {
				res <- v.Adcreatives.Data[i]
			}
		}

		return nil
	})

	return wg.Wait()
}

// Adcreativefields are the fields of an adcreative.
var Adcreativefields = []string{
	"id",
	"account_id",
	"body", // The body of the ad. Not supported for video post creatives.
	"call_to_action_type",
	"effective_instagram_story_id",
	"effective_object_story_id",
	"image_hash",
	"image_url",
	"instagram_actor_id",
	"instagram_permalink_url",
	"instagram_story_id",
	"link_og_id",
	"link_url",
	"name",
	"object_id",
	"object_story_id",
	"object_story_spec",
	"object_type",
	"object_url",
	"status",
	"thumbnail_url",
	"title",
	"video_id",
}

// AdCreative https://developers.facebook.com/docs/marketing-api/reference/ad-creative
type AdCreative struct {
	ID         string `json:"id,omitempty"`
	CreativeID string `json:"creative_id,omitempty"`
	AccountID  string `json:"account_id,omitempty"`
	// PageID
	ActorID string `json:"actor_id,omitempty"`
	// Deep link fallback behavior for dynamic product ads if the app is not installed.
	ApplinkTreatment string `json:"applink_treatment,omitempty"`
	// The body of the ad. Not supported for video post creatives.
	Body string `json:"body,omitempty"`
	// Branded Content sponsor ID, creating ads using existing BC posts
	BrandedContentSponsorPageID string `json:"branded_content_sponsor_page_id,omitempty"`

	CallToActionType string `json:"call_to_action_type,omitempty"`
	// The ID of an Instagram post to use in an ad.
	EffectiveInstagramStoryID string `json:"effective_instagram_story_id,omitempty"`
	// The ID of a page post to use in an ad, regardless of whether it's an organic or unpublished page post.
	EffectiveObjectStoryID string `json:"effective_object_story_id,omitempty"`
	// Image hash for an image you can use in creatives. If provided do not provide image_url. See image library for more details
	ImageHash string `json:"image_hash,omitempty"`
	// A URL for the image for this creative. We save the image at this URL to the ad account's image library. If provided do not include image_hash.
	ImageURL string `json:"image_url,omitempty"`
	// Instagram actor ID
	InstagramActorID string `json:"instagram_actor_id,omitempty"`
	// Instagram permalink
	InstagramPermalinkURL string `json:"instagram_permalink_url,omitempty"`
	// The ID of an Instagram post for creating ads.
	InstagramStoryID string `json:"instagram_story_id,omitempty"`
	// Used for creating video polls
	InteractiveComponentsSpec *InteractiveComponentsSpec `json:"interactive_components_spec,omitempty"`
	// The Open Graph (OG) ID for the link in this creative if the landing page has OG tags
	LinkOgID string `json:"link_og_id,omitempty"`
	// Used to identify a specific landing tab on the Page (e.g. a Page tab app)
	// by the Page tab's URL. See connection objects for retrieving Page tabs' URLs.
	// The likes tab is not supported. app_data parameters may be added to the url to pass data to a tab app
	LinkURL string `json:"link_url,omitempty"`
	// The JSON string of messenger sponsored message for this creative. See (docs/messenger-platform/reference/send-api) for more detail
	MessengerSponsoredMessage string `json:"messenger_sponsored_message,omitempty"`
	// The name of the creative in the creative library.
	Name string `json:"name,omitempty"`
	// The ID of the promoted_object or object that is relevant to the ad and ad type
	ObjectID string `json:"object_id,omitempty"`
	// The ID of a page post to use in an ad. This ID can be retrieved by using
	// the graph API to query the posts of the page. If an image is used in the post,
	// it will be downloaded and available in your account's image library.
	// If you create an unpublished page post inline via object_story_spec at
	// the same time as creating the ad, this ID will be null. However the
	// effective_object_story_id will be the ID of the page post regardless of whether it's an organic or unpublished page post.
	ObjectStoryID string `json:"object_story_id,omitempty"`
	// The type of object that is being advertised.
	// PAGE, DOMAIN, EVENT, STORE_ITEM, SHARE, PHOTO, STATUS, VIDEO, APPLICATION, INVALID
	ObjectType string `json:"object_type,omitempty"`
	// Destination URL for a link ads not connected to a page
	ObjectURL string `json:"object_url,omitempty"`
	// The ID of the product set for this creative. See dynamic product ads for more detail
	ProductSetID string `json:"product_set_id,omitempty"`
	// The status of this creative.
	Status string `json:"status,omitempty"`
	// The Tracking URL for dynamic product ads. See dynamic product ads for more detail
	TemplateURL string `json:"template_url,omitempty"`
	// The URL to a thumbnail for this creative.
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	// Title for a link ad (not connected to a Page)
	Title string `json:"title,omitempty"`
	// A set of query string parameters which will replace or be appended to urls
	// clicked from page post ads, message of the post, and canvas app install creatives only
	URLTags string `json:"url_tags,omitempty"`
	// The ID of the video in this creative
	VideoID string `json:"video_id,omitempty"`
	// If this is true, we will show the page actor for mobile app ads
	UsePageActorOverride bool `json:"use_page_actor_override,omitempty"`
	// A JSON object defining crop dimensions for the image specified. See image crop reference for more details
	ImageCrops json.RawMessage `json:"image_crops,omitempty"`
	// The page id and the content to create a new unpublished page post specified using one of link_data, photo_data, video_data, text_data or template_data
	ObjectStorySpec *ObjectStorySpec `json:"object_story_spec,omitempty"`
	// Use this field to customize the media for different Facebook placements.
	// Currently you can use this field for customizing images only. The media
	// specified here replaces the original media defined in the ad creative when
	// the ad displays on those placements. For example, if you define a media
	// here for the instagram key, Facebook uses that media instead of the media
	// defined in the ad creative when showing the ad on Instagram.
	PlatformCustomizations json.RawMessage `json:"platform_customizations,omitempty"`
	// The recommender settings that can be used to control recommendations for Dynamic Ads.
	RecommenderSettings json.RawMessage `json:"recommender_settings,omitempty"`
	// Use this field to create url templates for dynamic product ads. See dynamic product ads for more detail
	TemplateURLSpec json.RawMessage `json:"template_url_spec,omitempty"`
	// Ad Labels that are associated with this creative
	Adlabels []json.RawMessage `json:"adlabels,omitempty"`
	// DegreesOfFreedomSpec Specifies the types of transformations that are enabled for the given creative. For more information, see Ad Creative Degrees Of Freedom Spec, Reference.
	DegreesOfFreedomSpec *DegreesOfFreedomSpec `json:"degrees_of_freedom_spec,omitempty"`
}

type adCreativeContainer struct {
	Adcreatives struct {
		Data []AdCreative
	} `json:"adcreatives"`
}

// GetLandingPageURL returns the landing page URL of the creative.
func (ac AdCreative) GetLandingPageURL() string {
	if ac.ObjectStorySpec == nil {
		return ""
	}

	if ac.ObjectStorySpec.LinkData != nil {
		return ac.ObjectStorySpec.LinkData.Link
	} else if ac.ObjectStorySpec.VideoData != nil && ac.ObjectStorySpec.VideoData.CallToAction != nil && ac.ObjectStorySpec.VideoData.CallToAction.Value != nil {
		return ac.ObjectStorySpec.VideoData.CallToAction.Value.Link
	}

	return ""
}

// ObjectStorySpec contains the media of a creative.
type ObjectStorySpec struct {
	PageID           string               `json:"page_id,omitempty"`
	InstagramActorID string               `json:"instagram_actor_id,omitempty"`
	VideoData        *VideoData           `json:"video_data,omitempty"`
	LinkData         *AdCreativeLinkData  `json:"link_data,omitempty"`
	PhotoData        *AdCreativePhotoData `json:"photo_data,omitempty"`
}

type DegreesOfFreedomSpec struct {
	CreativeFeaturesSpec CreativeFeaturesSpec `json:"creative_features_spec"`
}
type StandardEnhancements struct {
	EnrollStatus string `json:"enroll_status"`
}
type CreativeFeaturesSpec struct {
	StandardEnhancements StandardEnhancements `json:"standard_enhancements"`
}

// InteractiveComponentsSpec is mainly used for Video Poll Ads.
type InteractiveComponentsSpec struct {
	Components []*Component `json:"components"`
}

// Component of the Interactive component struct.
type Component struct {
	Type         string        `json:"type"`
	PositionSpec *PositionSpec `json:"position_spec"`
	PollSpec     *PollSpec     `json:"poll_spec"`
}

// PollSpec represents the questions and answers of a poll.
type PollSpec struct {
	QuestionText        string              `json:"question_text"`
	OptionAText         string              `json:"option_a_text"`
	OptionBText         string              `json:"option_b_text"`
	ThemeColor          string              `json:"theme_color,omitempty"`
	OptionACallToAction *OptionCallToAction `json:"option_a_call_to_action,omitempty"`
	OptionBCallToAction *OptionCallToAction `json:"option_b_call_to_action,omitempty"`
	LinkDisplay         string              `json:"link_display,omitempty"`
}

// PositionSpec describes the position of an interactive component.
type PositionSpec struct {
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Width    float64 `json:"width"`
	Height   float64 `json:"height"`
	Rotation float64 `json:"rotation"`
}

// OptionCallToAction represents the action and call to action of an answer of a poll.
type OptionCallToAction struct {
	Value *OptionCallToActionValue `json:"value"`
	Type  string                   `json:"type"`
}

// OptionCallToActionValue describes the link of a call to action answer.
type OptionCallToActionValue struct {
	Link       string `json:"link"`
	LinkFormat string `json:"link_format,omitempty"`
}

// AdCreativePhotoData is the specific part of a creative that only photo posts do have.
type AdCreativePhotoData struct {
	BrandedContentSharedToSponsorStatus string `json:"branded_content_shared_to_sponsor_status"`
	BrandedContentSponsorPageID         string `json:"branded_content_sponsor_page_id"`
	BrandedContentSponsorRelationship   string `json:"branded_content_sponsor_relationship"`
	Caption                             string `json:"caption"`
	ImageHash                           string `json:"image_hash"`
	PageWelcomeMessage                  string `json:"page_welcome_message"`
	URL                                 string `json:"url"`
}

// VideoData is the specific part of a creative that only video posts do have.
type VideoData struct {
	ImageHash       string                          `json:"image_hash,omitempty"`
	ImageURL        string                          `json:"image_url,omitempty"`
	LinkDescription string                          `json:"link_description,omitempty"`
	Message         string                          `json:"message,omitempty"`
	Title           string                          `json:"title,omitempty"`
	VideoID         string                          `json:"video_id,omitempty"`
	CallToAction    *AdCreativeLinkDataCallToAction `json:"call_to_action,omitempty"`
}

// AdCreativeLinkData see https://developers.facebook.com/docs/marketing-api/reference/ad-creative-link-data/
type AdCreativeLinkData struct {
	// The index (zero based) of the image from the additional images array to use as the ad image for a dynamic product ad
	AdditionalImageIndex int32 `json:"additional_image_index,omitempty"`
	// Native deeplinks attached to the post
	AppLinkSpec *AdCreativeLinkDataAppLinkSpec `json:"app_link_spec,omitempty"`
	// The style of the attachment
	AttachmentStyle string `json:"attachment_style,omitempty"`
	// The branded content shared to sponsor option
	BrandedContentSharedToSponsorStatus string `json:"branded_content_shared_to_sponsor_status,omitempty"`
	// The branded content sponsor page id
	BrandedContentSponsorPageID string `json:"branded_content_sponsor_page_id,omitempty"`
	// The branded content sponsor relationship option
	BrandedContentSponsorRelationship string `json:"branded_content_sponsor_relationship,omitempty"`
	// An optional call to action button. If not specified, on Instagram, a default CTA would be used, {"type":"LEARN_MORE","value": {"link":<LINK VALUE OF LINK_DATA>,}}. Note that LIKE_PAGE is not supported
	CallToAction *AdCreativeLinkDataCallToAction `json:"call_to_action,omitempty"`
	// Link caption. Overwrites the caption under the title in the link. The caption must be an actual URLs and should accurately reflect the URL and associated advertiser or business someone visits when they click on it. See post for more info. This setting is not used on Instagram
	Caption string `json:"caption,omitempty"`
	// A 2-5 element array of link objects required for carousel ads. If multi_share_optimized is set to true, this array could have up to 10 objects. Facebook will automatically optimize the order in which the carousel cards are shown and display the top 5. We strongly recommend that you use at least 3 attachments for achieving optimal performance; allowing minimum of 2 attachments is for enabling lightweight integrations and using 2 objects might result in sub-optimal campaign results. If this ad creative is used for an Instagram Carousel ad, you will need to have at least 3 attachments for MOBILE_APP_INSTALLS ads and 2 for the other objectives. If more than 5 are given, only the first 5 will be shown on Instagram, even if multi_share_optimized is true.
	ChildAttachments []AdCreativeLinkDataChildAttachment `json:"child_attachments,omitempty"`
	// Customization rules for a dynamic ad
	// CustomizationRulesSpec []AdCustomizationRuleSpec `json:"customization_rules_spec,omitempty"`
	// Link description. Overwrites the description in the link on Facebook. See post for more info. This setting is not used on Instagram
	Description string `json:"description,omitempty"`
	// The id of a Facebook event. This is only to be used if this creative is for a Website Clicks campaign, the Call To Action is Buy Tickets, and the link points to the ticketing website of this Facebook event
	EventID string `json:"event_id,omitempty"`
	// Whether to force the post to render in a single link format
	ForceSingleLink bool `json:"force_single_link,omitempty"`
	// How to the image should be cropped. Different placements use different crop specs. For example, Facebook News Feed uses the crop spec with 191x100 key, and Instagram uses 100x100 crop spec
	// ImageCrops *AdsImageCrops `json:"image_crops,omitempty"`
	// Hash of an image in your image library with Facebook. Specify this field or picture but not both
	ImageHash string `json:"image_hash,omitempty"`
	// How to render image overlays on a dynamic item in Dynamic Ads
	// ImageOverlaySpec *AdCreativeLinkDataImageOverlaySpec `json:"image_overlay_spec,omitempty"`
	// Link url. This url is required to be the same as the CTA link url. See post for more info. This field is required for a carousel ad
	Link string `json:"link,omitempty"`
	// The main body of the post. See post for more info. This field is required for a carousel ad
	Message string `json:"message,omitempty"`
	// If set to false, removes the end card which displays the page icon. Default is true. Used by carousel ads
	MultiShareEndCard bool `json:"multi_share_end_card,omitempty"`
	// If set to true, automatically select and order images and links. Default is true. Used by carousel ads
	MultiShareOptimized bool `json:"multi_share_optimized,omitempty"`
	// Name of the link. Overwrites the title of the link preview. See post for more info
	Name string `json:"name,omitempty"`
	// The id of a Facebook native offer
	OfferID string `json:"offer_id,omitempty"`
	// A welcome text from page to user on Messenger once a user performs send message action on an ad
	PageWelcomeMessage string `json:"page_welcome_message,omitempty"`
	// URL of a picture to use in the post. Specify this field or image_hash but not both. See post for more info. The image specified at the URL will be saved into the ad accounts image library
	Picture string `json:"picture,omitempty"`
	// Customized contents provided by the advertiser for the ad post-click experience
	// PostClickConfiguration AdCreativePostClickConfiguration `json:"post_click_configuration,omitempty"`
	// List of product IDs provided by the advertiser for Collections
	RetailerItemIDs []string `json:"retailer_item_ids,omitempty"`
	// Use with force_single_link = true in order to show a single dynamic item but in carousel format using multiple images from the catalog. See dynamic product ad
	ShowMultipleImages bool `json:"show_multiple_images,omitempty"`
}

// AdCreativeLinkDataChildAttachment see https://developers.facebook.com/docs/marketing-api/reference/ad-creative-link-data-child-attachment/
type AdCreativeLinkDataChildAttachment struct {
	// Call to action of this attachment. On Facebook, we support one optional CTA per attachment. If it not specified, there will be no CTA for this attachment. On Instagram, there is one CTA per attachment. If the CTA is not specified, a CTA will be created by the system, using "Learn more" as the type, and the link from this child attachment as the link. If the CTA is specified, its link must be the same as the link of this child attachment.
	CallToAction *AdCreativeLinkDataCallToAction `json:"call_to_action,omitempty"`
	// The display url shown at the end of a video, if the attachment is a video
	Caption string `json:"caption,omitempty"`
	// Overwrites the description of each attachment on Facebook, not used on Instagram.
	Description string `json:"description,omitempty"`
	// Image crops, using the crop spec with 100x100 key for Carousel ads. If no 100x100 crop spec is provided, the image would be cropped automatically, unless the image is square already. The final cropped image size needs to be at least 200x200 pixels for Facebook, or 600x600 for Instagram.
	// ImageCrops *AdsImageCrops `json:"image_crops,omitempty"`
	// The image hash of an uploaded image for this attachment. For an ad on Facebook, if neither picture nor image_hash is set, the image of link_data above will be used. For an ad on Instagram, either picture or image_hash is required.
	ImageHash string `json:"image_hash,omitempty"`
	// The link of this attachment.
	Link string `json:"link,omitempty"`
	// Overwrites the title of the attachment on Facebook, not used on Instagram.
	Name string `json:"name,omitempty"`
	// The url of an image for this attachment. For an ad on Facebook, if neither picture nor image_hash is set, the image specified in link_data above will be used. For an ad on Instagram, either picture or image_hash is required.
	Picture string `json:"picture,omitempty"`
	// Whether to force the card to render statically, even in a dynamic ad.
	StaticCard bool `json:"static_card,omitempty"`
	// ID of an uploaded video, if this attachment is a video. Not supported for Instagram ads.
	VideoID string `json:"video_id,omitempty"`
}

// AdCreativeLinkDataAppLinkSpec see https://developers.facebook.com/docs/marketing-api/reference/ad-creative-link-data-app-link-spec/
type AdCreativeLinkDataAppLinkSpec struct {
	// Native deeplinks to use on Android
	Android []AndroidAppLink `json:"android,omitempty"`
	// Native deeplinks to use on iOS
	Ios []IosAppLink `json:"ios,omitempty"`
	// Native deeplinks to use on iPad
	Ipad []IosAppLink `json:"ipad,omitempty"`
	// Native deeplinks to use on iPhone
	Iphone []IosAppLink `json:"iphone,omitempty"`
}

// AdCreativeLinkDataCallToAction see https://developers.facebook.com/docs/marketing-api/reference/ad-creative-link-data-call-to-action/
type AdCreativeLinkDataCallToAction struct {
	// The type of the action. Not all types can be used for all ads. Check Ads Product Guide to see which type can be used based on the objective of your campaign.
	Type string `json:"type,omitempty"`
	// JSON containing the call to action data.
	Value *AdCreativeLinkDataCallToActionValue `json:"value,omitempty"`
}

// AdCreativeLinkDataCallToActionValue see https://developers.facebook.com/docs/marketing-api/reference/ad-creative-link-data-call-to-action-value/
type AdCreativeLinkDataCallToActionValue struct {
	// The app destination type.
	AppDestination string `json:"app_destination,omitempty"`
	// Deep link to the app.
	AppLink string `json:"app_link,omitempty"`
	// Application related to the action.
	Application string `json:"application,omitempty"`
	// ID of the Facebook event which the attachement show event info
	EventID string `json:"event_id,omitempty"`
	// The Lead Ad form id.
	LeadGenFormID string `json:"lead_gen_form_id,omitempty"`
	// The destination link when the CTA button is clicked. This is required to be same as the link url of the creative.
	Link string `json:"link,omitempty"`
	// Caption shown in the attachment. The caption must be an actual URL and should accurately reflect the URL and associated advertiser or business someone visits when they click on it.
	LinkCaption string `json:"link_caption,omitempty"`
	// Link format of video.
	LinkFormat string `json:"link_format,omitempty"`
	// ID of the Facebook page which the CTA button links to
	Page string `json:"page,omitempty"`
	// Open graph object url for canvas virtual good ads.
	ProductLink string `json:"product_link,omitempty"`
}

// AndroidAppLink see https://developers.facebook.com/docs/graph-api/reference/android-app-link/
type AndroidAppLink struct {
	// The native apps name in the Android store.
	AppName string `json:"app_name,omitempty"`
	// The fully classified class name of the app for intent generation.
	Class string `json:"class,omitempty"`
	// The fully classified package name of the app for intent generation.
	Package string `json:"package,omitempty"`
	// The native Android URL that will be navigated to.
	URL string `json:"url,omitempty"`
}

// IosAppLink see https://developers.facebook.com/docs/graph-api/reference/ios-app-link/
type IosAppLink struct {
	// The native apps name in the iTunes store.
	AppName string `json:"app_name,omitempty"`
	// The native apps ID in the iTunes store.
	AppStoreID string `json:"app_store_id,omitempty"`
	// The native iOS URL that will be navigated to.
	URL string `json:"url,omitempty"`
}
