package types

type UnixTime int64

// ServerEvent entity https://developers.facebook.com/docs/marketing-api/conversions-api/parameters/server-event
type ServerEvent struct {
	EventName      string              `json:"event_name"`
	EventID        string              `json:"event_id"`
	EventTime      UnixTime            `json:"event_time"`
	EventSourceURL string              `json:"event_source_url"`
	ActionSource   ActionSource        `json:"action_source"`
	UserData       CustomerInformation `json:"user_data,omitempty"`
	Contents       Contents            `json:"contents,omitempty"`

	// CustomData might represent your cutom struct with any fields.
	CustomData interface{} `json:"custom_data,omitempty"`
}

type ServerEvents []ServerEvent

func NewServerEvent(eventName, eventId string, eventTime UnixTime, actionSource ActionSource) ServerEvent {
	return ServerEvent{
		EventName:    eventName,
		EventID:      eventId,
		EventTime:    eventTime,
		ActionSource: actionSource,
	}
}

func (e ServerEvent) WithEventSourceURL(eventSourceURL string) ServerEvent {
	e.EventSourceURL = eventSourceURL
	return e
}

func (e ServerEvent) WithCustomData(customData interface{}) ServerEvent {
	e.CustomData = customData
	return e
}

func (e ServerEvent) WithUserData(customerInfo CustomerInformation) ServerEvent {
	e.UserData = customerInfo
	return e
}

func (e ServerEvent) WithContents(contents Contents) ServerEvent {
	e.Contents = contents
	return e
}

type ActionSource string

const (
	Email             ActionSource = "email"
	Website           ActionSource = "website"
	App               ActionSource = "app"
	PhoneCall         ActionSource = "phone_call"
	Chat              ActionSource = "chat"
	PhysicalStore     ActionSource = "physical_store"
	SystemGenerated   ActionSource = "system_generated"
	BusinessMessaging ActionSource = "business_messsaging"
	Other             ActionSource = "other"
)
