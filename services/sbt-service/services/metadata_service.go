package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/pkg/points"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/sbt-service/models"
)

// MetadataService handles SBT metadata generation and management
type MetadataService struct {
	db            *sql.DB
	pinataService *PinataService
	baseURL       string         // Base API URL for external_url
	pointsClient  *points.Client // Points service client
}

// NewMetadataService creates a new metadata service
func NewMetadataService(db *sql.DB, pinataService *PinataService, baseURL string, pointsServiceURL string) *MetadataService {
	var pointsClient *points.Client
	if pointsServiceURL != "" {
		pointsClient = points.NewClient(pointsServiceURL)
	}

	return &MetadataService{
		db:            db,
		pinataService: pinataService,
		baseURL:       baseURL,
		pointsClient:  pointsClient,
	}
}

// GenerateSBT generates SBT metadata and uploads to IPFS
func (ms *MetadataService) GenerateSBT(ctx context.Context, req *models.UserRegistrationRequest) (*models.SBTGenerationResponse, error) {
	// 1. Check if user already has an SBT
	exists, err := ms.userExists(ctx, req.WalletAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to check user existence: %v", err)
	}
	if exists {
		return &models.SBTGenerationResponse{
			Status:  "error",
			Message: "User already has SBT",
		}, nil
	}

	// 2. Upload avatar to IPFS (if provided)
	var imageURI string
	if req.AvatarBase64 != "" {
		filename := fmt.Sprintf("avatar_%s_%d.png", req.WalletAddress, time.Now().Unix())
		imageResp, err := ms.pinataService.UploadBase64Image(ctx, req.AvatarBase64, filename)
		if err != nil {
			return nil, fmt.Errorf("failed to upload avatar: %v", err)
		}
		imageURI = FormatIPFSURI(imageResp.IpfsHash)
	} else if req.ImageURL != "" {
		imageURI = req.ImageURL
	} else {
		// Use default avatar
		imageURI = "https://plum-added-rat-858.mypinata.cloud/ipfs/bafkreib3ik5fn42mk3v2774ja4s3ar64oymxa4k73tsoqdbrqvkldpwtcu"
	}

	// 3. Generate static metadata
	metadata := ms.generateStaticMetadata(req, imageURI)

	// 4. Upload metadata to IPFS
	metadataName := fmt.Sprintf("sbt_metadata_%s_%d", req.WalletAddress, time.Now().Unix())
	metadataResp, err := ms.pinataService.UploadJSON(ctx, metadata, metadataName)
	if err != nil {
		return nil, fmt.Errorf("failed to upload metadata: %v", err)
	}

	tokenURI := FormatIPFSURI(metadataResp.IpfsHash)

	// 5. Save user profile to database
	profile := &models.UserProfile{
		WalletAddress:     req.WalletAddress,
		DisplayName:       req.DisplayName,
		RegistrationDate:  time.Now(),
		Inviter:           req.InviteFrom,
		TotalPoints:       0,
		TodayContribution: 0,
		TokenURI:          tokenURI,
		ImageURI:          imageURI,
		IPFSHash:          metadataResp.IpfsHash,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := ms.saveUserProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to save user profile: %v", err)
	}

	// 6. Process invitation relationship
	if req.InviteFrom != "" {
		if err := ms.addInviteRelation(ctx, req.InviteFrom, req.WalletAddress); err != nil {
			// Don't block main flow, just log error
			fmt.Printf("Warning: failed to add invite relation: %v\n", err)
		}
	}

	return &models.SBTGenerationResponse{
		Status:   "ok",
		TokenURI: tokenURI,
		ImageURI: imageURI,
		Message:  "SBT metadata generated successfully",
	}, nil
}

// generateStaticMetadata generates static metadata for SBT
func (ms *MetadataService) generateStaticMetadata(req *models.UserRegistrationRequest, imageURI string) *models.SBTMetadata {
	// Build external URL for dynamic data
	externalURL := fmt.Sprintf("%s/api/v1/sbt/dynamic/%s", ms.baseURL, req.WalletAddress)

	// Static attributes
	attributes := []models.Attribute{
		{
			TraitType: "Wallet",
			Value:     req.WalletAddress,
		},
		{
			TraitType: "Display Name",
			Value:     req.DisplayName,
		},
		{
			TraitType:   "Registration Date",
			Value:       time.Now().Format(time.RFC3339),
			DisplayType: "date",
		},
	}

	// Add inviter information (if any)
	if req.InviteFrom != "" {
		// Hash inviter address for privacy
		inviterHash := hashAddress(req.InviteFrom)
		attributes = append(attributes, models.Attribute{
			TraitType: "Inviter",
			Value:     inviterHash,
		})
	}

	// Add initial attributes
	for key, value := range req.InitialAttrs {
		attributes = append(attributes, models.Attribute{
			TraitType: key,
			Value:     value,
		})
	}

	return &models.SBTMetadata{
		Name:            fmt.Sprintf("SBT - KEY Identity #%s", req.WalletAddress[:8]+"..."),
		Description:     fmt.Sprintf("Hetu KEY SBT for %s - A Soulbound Token representing verified identity and achievements in the Hetu ecosystem.", req.WalletAddress),
		Image:           imageURI,
		ExternalURL:     externalURL,
		Attributes:      attributes,
		BackgroundColor: "ffffff", // Optional, white background
	}
}

// GetDynamicMetadata returns dynamic metadata for external_url API
func (ms *MetadataService) GetDynamicMetadata(ctx context.Context, walletAddress string) (*models.DynamicMetadata, error) {
	// 1. Get user profile
	profile, err := ms.getUserProfile(ctx, walletAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %v", err)
	}

	// 2. Build dynamic attributes
	dynamicAttrs := []models.Attribute{
		{
			TraitType:   "Total Points",
			Value:       profile.TotalPoints,
			DisplayType: "number",
		},
		{
			TraitType:   "Today's Contribution",
			Value:       profile.TodayContribution,
			DisplayType: "number",
		},
	}

	// Add subnet membership
	for _, subnet := range profile.Subnets {
		dynamicAttrs = append(dynamicAttrs, models.Attribute{
			TraitType: "Subnet Membership",
			Value:     subnet.Name,
		})
	}

	// 3. Get historical points records
	pointsRecords, err := ms.getPointsHistory(ctx, walletAddress)
	if err != nil {
		// Don't block, return empty records
		pointsRecords = []models.PointsRecord{}
	}

	return &models.DynamicMetadata{
		DynamicAttributes:       dynamicAttrs,
		HistoricalPointsRecords: pointsRecords,
		Subnets:                 profile.Subnets,
		SubnetNFTs:              profile.SubnetNFTs,
	}, nil
}

// UpdateUserProfile updates user profile (dynamic data)
func (ms *MetadataService) UpdateUserProfile(ctx context.Context, req *models.UpdateProfileRequest) error {
	updates := []string{}
	args := []interface{}{}

	if req.TotalPoints != nil {
		updates = append(updates, "total_points = ?")
		args = append(args, *req.TotalPoints)
	}

	if req.TodayContribution != nil {
		updates = append(updates, "today_contribution = ?")
		args = append(args, *req.TodayContribution)
	}

	if req.Subnets != nil {
		subnetsJSON, _ := json.Marshal(req.Subnets)
		updates = append(updates, "subnets = ?")
		args = append(args, string(subnetsJSON))
	}

	if req.SubnetNFTs != nil {
		nftsJSON, _ := json.Marshal(req.SubnetNFTs)
		updates = append(updates, "subnet_nfts = ?")
		args = append(args, string(nftsJSON))
	}

	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	updates = append(updates, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, req.WalletAddress)

	query := fmt.Sprintf("UPDATE user_profiles SET %s WHERE wallet_address = ?",
		updates[0])
	for i := 1; i < len(updates); i++ {
		query = fmt.Sprintf("%s, %s", query, updates[i])
	}

	_, err := ms.db.ExecContext(ctx, query, args...)
	return err
}

// Database operations

func (ms *MetadataService) userExists(ctx context.Context, walletAddress string) (bool, error) {
	var count int
	query := "SELECT COUNT(*) FROM user_profiles WHERE wallet_address = ?"
	err := ms.db.QueryRowContext(ctx, query, walletAddress).Scan(&count)
	return count > 0, err
}

func (ms *MetadataService) saveUserProfile(ctx context.Context, profile *models.UserProfile) error {
	query := `
		INSERT INTO user_profiles (
			wallet_address, display_name, registration_date, inviter,
			total_points, today_contribution, token_uri, image_uri, ipfs_hash,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := ms.db.ExecContext(ctx, query,
		profile.WalletAddress, profile.DisplayName, profile.RegistrationDate, profile.Inviter,
		profile.TotalPoints, profile.TodayContribution, profile.TokenURI, profile.ImageURI, profile.IPFSHash,
		profile.CreatedAt, profile.UpdatedAt,
	)

	return err
}

func (ms *MetadataService) getUserProfile(ctx context.Context, walletAddress string) (*models.UserProfile, error) {
	query := `
		SELECT wallet_address, display_name, registration_date, inviter,
		       total_points, today_contribution, token_uri, token_id, image_uri, ipfs_hash,
		       subnets, subnet_nfts, created_at, updated_at
		FROM user_profiles WHERE wallet_address = ?
	`

	var profile models.UserProfile
	var subnetsJSON, nftsJSON sql.NullString
	var inviter sql.NullString

	err := ms.db.QueryRowContext(ctx, query, walletAddress).Scan(
		&profile.WalletAddress, &profile.DisplayName, &profile.RegistrationDate, &inviter,
		&profile.TotalPoints, &profile.TodayContribution, &profile.TokenURI, &profile.TokenID, &profile.ImageURI, &profile.IPFSHash,
		&subnetsJSON, &nftsJSON, &profile.CreatedAt, &profile.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if inviter.Valid {
		profile.Inviter = inviter.String
	}

	// Parse JSON field
	if subnetsJSON.Valid {
		json.Unmarshal([]byte(subnetsJSON.String), &profile.Subnets)
	}

	if nftsJSON.Valid {
		json.Unmarshal([]byte(nftsJSON.String), &profile.SubnetNFTs)
	}

	return &profile, nil
}

func (ms *MetadataService) getPointsHistory(ctx context.Context, walletAddress string) ([]models.PointsRecord, error) {
	query := `
		SELECT date, source, points, tx_ref 
		FROM points_history 
		WHERE wallet_address = ? 
		ORDER BY date DESC 
		LIMIT 50
	`

	rows, err := ms.db.QueryContext(ctx, query, walletAddress)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []models.PointsRecord
	for rows.Next() {
		var record models.PointsRecord
		var txRef sql.NullString

		err := rows.Scan(&record.Date, &record.Source, &record.Points, &txRef)
		if err != nil {
			continue
		}

		if txRef.Valid {
			record.TxRef = txRef.String
		}

		records = append(records, record)
	}

	return records, nil
}

func (ms *MetadataService) addInviteRelation(ctx context.Context, inviter, invitee string) error {
	query := "INSERT INTO invite_relations (inviter, invitee, created_at) VALUES (?, ?, ?)"
	_, err := ms.db.ExecContext(ctx, query, inviter, invitee, time.Now())
	return err
}

// Helper functions

func hashAddress(address string) string {
	// Simple hash processing, should use more secure hash algorithm in practice
	if len(address) > 10 {
		return address[:6] + "..." + address[len(address)-4:]
	}
	return address
}

// GetUserProfile gets user profile (public method)
func (ms *MetadataService) GetUserProfile(ctx context.Context, walletAddress string) (*models.UserProfile, error) {
	return ms.getUserProfile(ctx, walletAddress)
}

// UpdateInviteRelation updates invite relationship
func (ms *MetadataService) UpdateInviteRelation(ctx context.Context, req *models.UpdateInviteRequest) error {
	// Check if user exists
	var exists bool
	checkQuery := "SELECT EXISTS(SELECT 1 FROM user_profiles WHERE wallet_address = ?)"
	err := ms.db.QueryRowContext(ctx, checkQuery, req.WalletAddress).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %v", err)
	}
	if !exists {
		return fmt.Errorf("user with wallet address %s not found", req.WalletAddress)
	}

	// Build update SQL
	var updates []string
	var args []interface{}

	if req.InviteFrom != "" {
		updates = append(updates, "invite_from = ?")
		args = append(args, req.InviteFrom)
	}

	if req.InviteTo != nil {
		// Convert invitee list to JSON string
		inviteToJSON, err := json.Marshal(req.InviteTo)
		if err != nil {
			return fmt.Errorf("failed to marshal invite_to: %v", err)
		}
		updates = append(updates, "invite_to = ?")
		args = append(args, string(inviteToJSON))
	}

	if len(updates) == 0 {
		return fmt.Errorf("no fields to update")
	}

	// Add update time
	updates = append(updates, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, req.WalletAddress)

	// Execute update
	query := fmt.Sprintf("UPDATE user_profiles SET %s WHERE wallet_address = ?",
		strings.Join(updates, ", "))

	_, err = ms.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update invite relation: %v", err)
	}

	// Also update dynamic data on IPFS
	err = ms.updateDynamicDataOnIPFS(ctx, req.WalletAddress)
	if err != nil {
		// Log error but don't affect main flow
		log.Printf("Warning: Failed to update dynamic data on IPFS for %s: %v", req.WalletAddress, err)
	}

	log.Printf("Invite relation updated successfully for wallet: %s", req.WalletAddress)
	return nil
}

// updateDynamicDataOnIPFS updates dynamic data on IPFS
func (ms *MetadataService) updateDynamicDataOnIPFS(ctx context.Context, walletAddress string) error {
	// Get latest dynamic data
	dynamicData, err := ms.GetDynamicMetadata(ctx, walletAddress)
	if err != nil {
		return fmt.Errorf("failed to get dynamic metadata: %v", err)
	}

	// Upload dynamic data to IPFS
	dynamicJSON, err := json.Marshal(dynamicData)
	if err != nil {
		return fmt.Errorf("failed to marshal dynamic data: %v", err)
	}

	filename := fmt.Sprintf("dynamic_%s.json", walletAddress)
	_, err = ms.pinataService.UploadJSON(ctx, json.RawMessage(dynamicJSON), filename)
	if err != nil {
		return fmt.Errorf("failed to pin dynamic data to IPFS: %v", err)
	}

	return nil
}

// FormatIPFSURI formats IPFS hash as URI (imported from pinata_service)
func FormatIPFSURI(ipfsHash string) string {
	return fmt.Sprintf("ipfs://%s", ipfsHash)
}
