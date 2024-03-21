package v19

// PromotedObject contains the id of a promoted page.
type PromotedObject struct {
	PageID             string `json:"page_id,omitempty"`
	PixelID            string `json:"pixel_id,omitempty"`
	PixelRule          string `json:"pixel_rule,omitempty"`
	CustomEventType    string `json:"custom_event_type,omitempty"`
	CustomConversionID string `json:"custom_conversion_id,omitempty"`
	ApplicationID      string `json:"application_id,omitempty"`
	ObjectStoreURL     string `json:"object_store_url,omitempty"`
}
