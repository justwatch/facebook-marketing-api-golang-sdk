package types

import (
	"crypto/sha256"
	"encoding/hex"
)

// CutomerInfromation entity https://developers.facebook.com/docs/marketing-api/conversions-api/parameters/customer-information-parameters
type CustomerInformation struct {
	Email           []string `json:"em,omitempty"`
	PhoneNumber     []string `json:"ph,omitempty"`
	ClientIPAddress string   `json:"client_ip_address,omitempty"`
	ClientUserAgent string   `json:"client_user_agent,omitempty"`
	Fbc             string   `json:"fbc,omitempty"`
	Fbp             string   `json:"fbp,omitempty"`
}

func NewCustomerInformation() CustomerInformation {
	return CustomerInformation{}
}

// Adds already hashed email to the struct
func (ci CustomerInformation) WithHashedEmail(hashedEmail string) CustomerInformation {
	ci.Email = []string{hashedEmail}
	return ci
}

// Adds non-hashed email to the struct for it to be hashed.
func (ci CustomerInformation) WithEmail(email string) CustomerInformation {
	hashedEmailBytes := sha256.Sum256([]byte(email))
	hashedEmail := hex.EncodeToString(hashedEmailBytes[:])
	ci.Email = []string{hashedEmail}
	return ci
}

// Adds non-hashed phoneNumber to the struct for it to be hashed.
func (ci CustomerInformation) WithPhoneNumber(email string) CustomerInformation {
	hashedPhoneBytes := sha256.Sum256([]byte(email))
	hashedPhone := hex.EncodeToString(hashedPhoneBytes[:])
	ci.PhoneNumber = []string{hashedPhone}
	return ci
}

// Adds already hashed email to the struct
func (ci CustomerInformation) WithHashedPhoneNumber(hashedPhoneNumber string) CustomerInformation {
	ci.PhoneNumber = []string{hashedPhoneNumber}
	return ci
}

func (ci CustomerInformation) WithFbc(fbc string) CustomerInformation {
	ci.Fbc = fbc
	return ci
}

func (ci CustomerInformation) WithFbp(fbp string) CustomerInformation {
	ci.Fbp = fbp
	return ci
}

func (ci CustomerInformation) WithClientIPAddress(clientIPAddress string) CustomerInformation {
	ci.ClientIPAddress = clientIPAddress
	return ci
}

func (ci CustomerInformation) WithClientUserAgent(clientUserAgent string) CustomerInformation {
	ci.ClientUserAgent = clientUserAgent
	return ci
}
