package meta

// Targeting contains all the targeting information of an adset.
type Targeting struct {
	// inventories
	PublisherPlatforms []string `json:"publisher_platforms,omitempty"`
	// sub inventories
	FacebookPositions        []string `json:"facebook_positions,omitempty"`
	InstagramPositions       []string `json:"instagram_positions,omitempty"`
	AudienceNetworkPositions []string `json:"audience_network_positions,omitempty"`
	MessengerPositions       []string `json:"messenger_positions,omitempty"`

	AgeMin  uint64 `json:"age_min,omitempty"`
	AgeMax  uint64 `json:"age_max,omitempty"`
	Genders []int  `json:"genders,omitempty"`

	AppInstallState string `json:"app_install_state,omitempty"`

	CustomAudiences         []IDContainer  `json:"custom_audiences,omitempty"`
	ExcludedCustomAudiences []IDContainer  `json:"excluded_custom_audiences,omitempty"`
	GeoLocations            *GeoLocations  `json:"geo_locations,omitempty"`
	ExcludedGeoLocations    *GeoLocations  `json:"excluded_geo_locations,omitempty"`
	FlexibleSpec            []FlexibleSpec `json:"flexible_spec,omitempty"`
	Exclusions              *FlexibleSpec  `json:"exclusions,omitempty"`

	DevicePlatforms             []string                 `json:"device_platforms,omitempty"`
	ExcludedPublisherCategories []string                 `json:"excluded_publisher_categories,omitempty"`
	Locales                     []int                    `json:"locales,omitempty"`
	TargetingOptimization       string                   `json:"targeting_optimization,omitempty"`
	UserDevice                  []string                 `json:"user_device,omitempty"`
	UserOs                      []string                 `json:"user_os,omitempty"`
	WirelessCarrier             []string                 `json:"wireless_carrier,omitempty"`
	TargetingRelaxationTypes    TargetingRelaxationTypes `json:"targeting_relaxation_types,omitempty"`
}

// IDContainer contains an ID and a name.
type IDContainer struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// GeoLocations is a set of countries, cities, and regions that can be targeted.
type GeoLocations struct {
	Countries     []string `json:"countries,omitempty"`
	LocationTypes []string `json:"location_types,omitempty"`
	Cities        []City   `json:"cities,omitempty"`
	Regions       []Region `json:"regions,omitempty"`
	Zips          []Zip    `json:"zips,omitempty"`
}

type City struct {
	Country      string `json:"country"`
	DistanceUnit string `json:"distance_unit"`
	Key          string `json:"key"`
	Name         string `json:"name"`
	Radius       int    `json:"radius"`
	Region       string `json:"region"`
	RegionID     string `json:"region_id"`
}

// Region can be targeted.
type Region struct {
	Key     string `json:"key"`
	Name    string `json:"name"`
	Country string `json:"country"`
}

// Zip can be targeted.
type Zip struct {
	Key           string `json:"key"`
	Name          string `json:"name"`
	PrimaryCityID int    `json:"primary_city_id"`
	RegionID      int    `json:"region_id"`
	Country       string `json:"country"`
}

// FlexibleSpec is used for targeting
type FlexibleSpec struct {
	Interests            []IDContainer `json:"interests,omitempty"`
	Behaviors            []IDContainer `json:"behaviors,omitempty"`
	LifeEvents           []IDContainer `json:"life_events,omitempty"`
	WorkEmployers        []IDContainer `json:"work_employers,omitempty"`
	FamilyStatuses       []IDContainer `json:"family_statuses,omitempty"`
	WorkPositions        []IDContainer `json:"work_positions,omitempty"`
	Politics             []IDContainer `json:"politics,omitempty"`
	EducationMajors      []IDContainer `json:"education_majors,omitempty"`
	EducationStatuses    []int         `json:"education_statuses,omitempty"`
	RelationshipStatuses []int         `json:"relationship_statuses,omitempty"`
}

// Advantage custom audience and Advantage lookalike can be enabled or disabled.
// if a value of 0 is passed, it will be disabled. If a value of 1 is passed, it will be enabled.
// If no key/value pair is passed, it will be considered as enabled.
// https://developers.facebook.com/docs/graph-api/changelog/version15.0/
type TargetingRelaxationTypes struct {
	CustomAudience int8 `json:"custom_audience"`
	Lookalike      int8 `json:"lookalike"`
}
