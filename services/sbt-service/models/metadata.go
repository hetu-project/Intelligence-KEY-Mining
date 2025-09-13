package models

import (
	"database/sql"
	"time"
)

// SBTMetadata represents the complete SBT metadata structure
type SBTMetadata struct {
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	Image           string      `json:"image"`                      // IPFS avatar link
	ExternalURL     string      `json:"external_url"`               // Dynamic data API endpoint
	Attributes      []Attribute `json:"attributes"`                 // Static attributes
	BackgroundColor string      `json:"background_color,omitempty"` // Optional background color
}

// Attribute represents a metadata attribute
type Attribute struct {
	TraitType   string      `json:"trait_type"`
	Value       interface{} `json:"value"`
	DisplayType string      `json:"display_type,omitempty"` // "date", "number", etc.
}

// UserRegistrationRequest represents user registration request
type UserRegistrationRequest struct {
	WalletAddress string                 `json:"wallet_address" validate:"required"`
	DisplayName   string                 `json:"display_name" validate:"required"`
	AvatarBase64  string                 `json:"avatar_base64,omitempty"` // Base64 encoded avatar
	ImageURL      string                 `json:"image_url,omitempty"`     // Or directly provide image URL
	InviteFrom    string                 `json:"invite_from,omitempty"`   // Inviter
	InviteTo      []string               `json:"invite_to,omitempty"`     // Invitees
	InitialAttrs  map[string]interface{} `json:"initial_attributes,omitempty"`
}

// SBTGenerationResponse represents SBT generation response
type SBTGenerationResponse struct {
	Status          string `json:"status"`
	TokenURI        string `json:"token_uri"`        // IPFS URI
	TokenID         string `json:"token_id"`         // Token ID on blockchain
	ContractAddress string `json:"contract_address"` // Contract address
	ImageURI        string `json:"image_uri"`        // Avatar IPFS URI
	Message         string `json:"message"`
}

// DynamicMetadata represents dynamic metadata returned by external_url
type DynamicMetadata struct {
	DynamicAttributes       []Attribute    `json:"dynamic_attributes"`
	HistoricalPointsRecords []PointsRecord `json:"historical_points_records"`
	Subnets                 []SubnetInfo   `json:"subnets"`
	SubnetNFTs              []SubnetNFT    `json:"subnet_nfts"`
}

// PointsRecord represents a points record
type PointsRecord struct {
	Date   string `json:"date"`   // "2023-10-20"
	Source string `json:"source"` // "Daily Task", "Content Creation"
	Points int    `json:"points"`
	TxRef  string `json:"tx_ref,omitempty"` // Transaction reference
}

// SubnetInfo represents subnet information
type SubnetInfo struct {
	Name string `json:"name"` // "AI Training Net"
	Icon string `json:"icon"` // IPFS icon link
}

// SubnetNFT represents subnet NFT information
type SubnetNFT struct {
	Contract string `json:"contract"` // Contract address
	TokenID  int64  `json:"tokenId"`  // Token ID
}

// UserProfile represents complete user profile data
type UserProfile struct {
	// Basic information (static)
	WalletAddress    string    `json:"wallet_address" db:"wallet_address"`
	DisplayName      string    `json:"display_name" db:"display_name"`
	RegistrationDate time.Time `json:"registration_date" db:"registration_date"`

	// Invitation information
	Inviter  string   `json:"inviter,omitempty" db:"inviter"`
	Invitees []string `json:"invitees,omitempty"`

	// Points overview (dynamic)
	TotalPoints       int `json:"total_points" db:"total_points"`
	TodayContribution int `json:"today_contribution" db:"today_contribution"`

	// Subnet information (dynamic)
	Subnets    []SubnetInfo `json:"subnets"`
	SubnetNFTs []SubnetNFT  `json:"subnet_nfts"`

	// SBT information
	TokenURI     string        `json:"token_uri" db:"token_uri"`
	TokenID      sql.NullInt64 `json:"-" db:"token_id"`
	TokenIDValue int64         `json:"token_id"`
	ImageURI     string        `json:"image_uri" db:"image_uri"`
	IPFSHash     string        `json:"ipfs_hash" db:"ipfs_hash"`

	// Timestamps
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// PinataUploadRequest represents Pinata upload request
type PinataUploadRequest struct {
	PinataContent  interface{}     `json:"pinataContent"`
	PinataMetadata *PinataMetadata `json:"pinataMetadata,omitempty"`
	PinataOptions  *PinataOptions  `json:"pinataOptions,omitempty"`
}

// PinataMetadata represents Pinata metadata
type PinataMetadata struct {
	Name      string            `json:"name"`
	KeyValues map[string]string `json:"keyvalues,omitempty"`
}

// PinataOptions represents Pinata options
type PinataOptions struct {
	CidVersion int `json:"cidVersion,omitempty"`
}

// PinataResponse represents Pinata response
type PinataResponse struct {
	IpfsHash  string    `json:"IpfsHash"`
	PinSize   int       `json:"PinSize"`
	Timestamp time.Time `json:"Timestamp"`
}

// UpdateProfileRequest represents profile update request
type UpdateProfileRequest struct {
	WalletAddress     string       `json:"wallet_address" validate:"required"`
	TotalPoints       *int         `json:"total_points,omitempty"`
	TodayContribution *int         `json:"today_contribution,omitempty"`
	Subnets           []SubnetInfo `json:"subnets,omitempty"`
	SubnetNFTs        []SubnetNFT  `json:"subnet_nfts,omitempty"`
}

// UpdateInviteRequest represents invite relation update request
type UpdateInviteRequest struct {
	WalletAddress string   `json:"wallet_address" validate:"required"`
	InviteFrom    string   `json:"invite_from,omitempty"` // Inviter
	InviteTo      []string `json:"invite_to,omitempty"`   // Invitees list
}
