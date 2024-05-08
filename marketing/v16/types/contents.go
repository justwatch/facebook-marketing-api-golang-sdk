package types

type Contents []Content

// Contents entity is a part of standart parameters. A list of JSON objects that contain the product IDs associated with the event plus information about the products
type Content struct {
	ID               string `json:"id,omitempty"`
	Quantity         int    `json:"quantity,omitempty"`
	DeliveryCategory string `json:"delivery_category,omitempty"`
}

func NewContents() Contents {
	return Contents{}
}

func (c Contents) AddContent(content Content) Contents {
	return append(c, content)
}
