package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/hetu-project/Intelligence-KEY-Mining/pkg/points"
	"github.com/hetu-project/Intelligence-KEY-Mining/services/sbt-service/models"
)

// MetadataService handles SBT metadata generation and management
type MetadataService struct {
	db              *sql.DB
	pinataService   *PinataService
	baseURL         string           // Base API URL for external_url
	pointsClient    *points.Client   // Points service client
	referralService *ReferralService // Third-party referral API client
}

// NewMetadataService creates a new metadata service
func NewMetadataService(db *sql.DB, pinataService *PinataService, baseURL string, pointsServiceURL string, referralAPIURL string) *MetadataService {
	var pointsClient *points.Client
	if pointsServiceURL != "" {
		pointsClient = points.NewClient(pointsServiceURL)
	}

	// Initialize referral service (can be nil if URL not provided)
	referralService := NewReferralService(referralAPIURL)

	return &MetadataService{
		db:              db,
		pinataService:   pinataService,
		baseURL:         baseURL,
		pointsClient:    pointsClient,
		referralService: referralService,
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

	// Extract Twitter ID from initial attributes
	var twitterID string
	if req.InitialAttrs != nil {
		if twitterIDValue, exists := req.InitialAttrs["twitter_id"]; exists {
			if twitterIDStr, ok := twitterIDValue.(string); ok {
				twitterID = twitterIDStr
			}
		}
	}

	// 5. Save user profile to database
	profile := &models.UserProfile{
		WalletAddress:     req.WalletAddress,
		DisplayName:       req.DisplayName,
		TwitterID:         twitterID,
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

	// 2. Get additional mining statistics
	miningTotalPoints, err := ms.getUserMiningTotalPoints(ctx, walletAddress)
	if err != nil {
		// Don't block, use 0 as fallback
		miningTotalPoints = 0
	}

	consecutiveDays, err := ms.getUserConsecutiveMiningDays(ctx, walletAddress)
	if err != nil {
		// Don't block, use 0 as fallback
		consecutiveDays = 0
	}

	// 3. Get today's contribution (dynamic calculation)
	todayContribution, err := ms.getUserTodayContribution(ctx, walletAddress)
	if err != nil {
		// Don't block, use 0 as fallback
		todayContribution = 0
	}

	// 3. Build dynamic attributes
	dynamicAttrs := []models.Attribute{
		{
			TraitType:   "Total Points",
			Value:       profile.TotalPoints,
			DisplayType: "number",
		},
		{
			TraitType:   "Today Contribution",
			Value:       todayContribution,
			DisplayType: "number",
		},
		{
			TraitType:   "Total Mining Points",
			Value:       miningTotalPoints,
			DisplayType: "number",
		},
		{
			TraitType:   "Consecutive Mining Days",
			Value:       consecutiveDays,
			DisplayType: "number",
		},
	}

	// 4. Get user subnets from points-service API
	userSubnets, err := ms.getUserSubnetsFromPointsService(ctx, walletAddress)
	if err != nil {
		// Fallback to profile subnets
		userSubnets = profile.Subnets
	}

	// Add subnet membership to dynamic attributes
	for _, subnet := range userSubnets {
		dynamicAttrs = append(dynamicAttrs, models.Attribute{
			TraitType: "Subnet Membership",
			Value:     subnet.Name,
		})
	}

	// 5. Get historical points records
	pointsRecords, err := ms.getPointsHistory(ctx, walletAddress)
	if err != nil {
		// Don't block, return empty records
		pointsRecords = []models.PointsRecord{}
	}

	// 6. Get invitation information from third-party API
	var invitationInfo *models.InvitationInfo
	if ms.referralService != nil {
		invitationInfo = ms.referralService.GetReferralInfoSafe(ctx, walletAddress)
	} else {
		// Fallback: empty invitation info if no referral service configured
		invitationInfo = &models.InvitationInfo{
			Invitees:     []string{},
			InviteeCount: 0,
		}
	}

	return &models.DynamicMetadata{
		DynamicAttributes:       dynamicAttrs,
		HistoricalPointsRecords: pointsRecords,
		Subnets:                 userSubnets, // Now using API data instead of profile.Subnets
		SubnetNFTs:              profile.SubnetNFTs,
		InvitationInfo:          invitationInfo,
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
			wallet_address, display_name, twitter_id, registration_date, inviter,
			total_points, today_contribution, token_uri, image_uri, ipfs_hash,
			created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := ms.db.ExecContext(ctx, query,
		profile.WalletAddress, profile.DisplayName, profile.TwitterID, profile.RegistrationDate, profile.Inviter,
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
	var imageURI sql.NullString

	err := ms.db.QueryRowContext(ctx, query, walletAddress).Scan(
		&profile.WalletAddress, &profile.DisplayName, &profile.RegistrationDate, &inviter,
		&profile.TotalPoints, &profile.TodayContribution, &profile.TokenURI, &profile.TokenID, &imageURI, &profile.IPFSHash,
		&subnetsJSON, &nftsJSON, &profile.CreatedAt, &profile.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if inviter.Valid {
		profile.Inviter = inviter.String
	}

	// Handle nullable image_uri
	if imageURI.Valid {
		profile.ImageURI = imageURI.String
	}

	// Handle TokenID conversion
	if profile.TokenID.Valid {
		profile.TokenIDValue = profile.TokenID.Int64
	} else {
		profile.TokenIDValue = 0
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

// getUserMiningTotalPoints calculates total points earned from task completion (mining)
func (ms *MetadataService) getUserMiningTotalPoints(ctx context.Context, walletAddress string) (int, error) {
	query := `
		SELECT COALESCE(SUM(points), 0) 
		FROM points_history 
		WHERE wallet_address = ? 
		AND (source LIKE '%Task%' OR source LIKE '%Twitter%' OR source LIKE '%Retweet%')
		AND source NOT LIKE '%NFT%' 
		AND source NOT LIKE '%Invitation%'
	`

	var miningPoints int
	err := ms.db.QueryRowContext(ctx, query, walletAddress).Scan(&miningPoints)
	if err != nil {
		return 0, fmt.Errorf("failed to get mining total points: %v", err)
	}

	return miningPoints, nil
}

// getUserConsecutiveMiningDays calculates consecutive days of mining (task completion)
func (ms *MetadataService) getUserConsecutiveMiningDays(ctx context.Context, walletAddress string) (int, error) {
	// Get all distinct completion dates for this user, ordered by date descending
	query := `
		SELECT DISTINCT DATE(completed_at) as completion_date
		FROM user_task_completions 
		WHERE user_wallet = ? 
		ORDER BY completion_date DESC
	`

	rows, err := ms.db.QueryContext(ctx, query, walletAddress)
	if err != nil {
		return 0, fmt.Errorf("failed to query completion dates: %v", err)
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var dateStr string
		if err := rows.Scan(&dateStr); err != nil {
			continue
		}

		// Parse date string (YYYY-MM-DD format)
		date, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		dates = append(dates, date)
	}

	if len(dates) == 0 {
		return 0, nil
	}

	// Calculate consecutive days from the most recent date
	consecutiveDays := 1
	today := time.Now().Truncate(24 * time.Hour)

	// Check if the most recent completion was today or yesterday
	mostRecent := dates[0]
	daysDiff := int(today.Sub(mostRecent).Hours() / 24)

	// If last completion was more than 1 day ago, consecutive streak is broken
	if daysDiff > 1 {
		return 0, nil
	}

	// If last completion was today, start counting from today
	// If last completion was yesterday, start counting from yesterday
	if daysDiff == 1 {
		consecutiveDays = 1
	} else {
		consecutiveDays = 1
	}

	// Count backward to find consecutive days
	for i := 1; i < len(dates); i++ {
		expectedDate := dates[i-1].AddDate(0, 0, -1)
		if dates[i].Equal(expectedDate) {
			consecutiveDays++
		} else {
			break
		}
	}

	return consecutiveDays, nil
}

// getUserTodayContribution calculates user's points earned today (dynamic calculation)
func (ms *MetadataService) getUserTodayContribution(ctx context.Context, walletAddress string) (int, error) {
	// Query points-service for today's points
	if ms.pointsClient == nil {
		// Fallback: return 0 if no points service configured
		return 0, nil
	}

	// Build API URL for points history
	pointsServiceURL := strings.TrimSuffix(os.Getenv("POINTS_SERVICE_URL"), "/")
	if pointsServiceURL == "" {
		log.Printf("Warning: POINTS_SERVICE_URL not set, cannot fetch today's contribution")
		return 0, nil
	}

	// Use points history API with today's date filter
	apiURL := fmt.Sprintf("%s/api/v1/points/history/%s", pointsServiceURL, walletAddress)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %v", err)
	}

	// Make HTTP request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Don't fail completely, return 0
		log.Printf("Warning: failed to get points history from points service: %v", err)
		return 0, nil
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		log.Printf("Warning: points service returned status %d for points history", resp.StatusCode)
		return 0, nil
	}

	// Parse response
	var apiResponse struct {
		Status string `json:"status"`
		Data   struct {
			History []struct {
				Date   string `json:"date"`
				Points int    `json:"points"`
			} `json:"history"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		log.Printf("Warning: failed to decode points history response: %v", err)
		return 0, nil
	}

	if apiResponse.Status != "success" {
		log.Printf("Warning: points service returned status=%s for points history", apiResponse.Status)
		return 0, nil
	}

	// Calculate today's total points
	today := time.Now().Format("2006-01-02")
	todayPoints := 0

	for _, record := range apiResponse.Data.History {
		// Parse the date from the record (handle both date formats)
		recordDate := record.Date
		if strings.Contains(recordDate, "T") {
			// Handle ISO format like "2025-10-09T00:00:00Z"
			if t, err := time.Parse(time.RFC3339, recordDate); err == nil {
				recordDate = t.Format("2006-01-02")
			}
		}

		if recordDate == today {
			todayPoints += record.Points
		}
	}

	return todayPoints, nil
}

// getUserSubnetsFromPointsService gets user subnets from points-service API
func (ms *MetadataService) getUserSubnetsFromPointsService(ctx context.Context, walletAddress string) ([]models.SubnetInfo, error) {
	if ms.pointsClient == nil {
		// Fallback: return empty subnets if no points service configured
		return []models.SubnetInfo{}, nil
	}

	// Build API URL - need to access baseURL through a method or store it separately
	// For now, we'll extract it from the pointsClient or use environment variable
	pointsServiceURL := strings.TrimSuffix(os.Getenv("POINTS_SERVICE_URL"), "/")
	if pointsServiceURL == "" {
		log.Printf("Warning: POINTS_SERVICE_URL not set, cannot fetch user subnets")
		return []models.SubnetInfo{}, nil
	}
	apiURL := fmt.Sprintf("%s/api/v1/stats/users/%s/subnets", pointsServiceURL, walletAddress)

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Make HTTP request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		// Don't fail completely, return empty subnets
		log.Printf("Warning: failed to get user subnets from points service: %v", err)
		return []models.SubnetInfo{}, nil
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		log.Printf("Warning: points service returned status %d for user subnets", resp.StatusCode)
		return []models.SubnetInfo{}, nil
	}

	// Parse response
	var apiResponse struct {
		Success bool `json:"success"`
		Data    struct {
			UserWallet string `json:"user_wallet"`
			Subnets    []struct {
				SubnetID   string `json:"subnet_id"`
				SubnetName string `json:"subnet_name"`
				SubnetIcon string `json:"subnet_icon"`
			} `json:"subnets"`
			Count int `json:"count"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		log.Printf("Warning: failed to decode user subnets response: %v", err)
		return []models.SubnetInfo{}, nil
	}

	if !apiResponse.Success {
		log.Printf("Warning: points service returned success=false for user subnets")
		return []models.SubnetInfo{}, nil
	}

	// Convert to SubnetInfo format
	var subnets []models.SubnetInfo
	for _, subnet := range apiResponse.Data.Subnets {
		subnets = append(subnets, models.SubnetInfo{
			Name: subnet.SubnetName,
			Icon: subnet.SubnetIcon,
		})
	}

	return subnets, nil
}

// UpdateTwitterID updates the Twitter ID for a user
func (ms *MetadataService) UpdateTwitterID(ctx context.Context, walletAddress, twitterID string) error {
	// Check if user exists
	var exists bool
	checkQuery := `SELECT EXISTS(SELECT 1 FROM user_profiles WHERE wallet_address = ?)`
	err := ms.db.QueryRowContext(ctx, checkQuery, walletAddress).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check user existence: %v", err)
	}

	if !exists {
		return fmt.Errorf("user not found")
	}

	// Update Twitter ID
	updateQuery := `UPDATE user_profiles SET twitter_id = ? WHERE wallet_address = ?`
	result, err := ms.db.ExecContext(ctx, updateQuery, twitterID, walletAddress)
	if err != nil {
		return fmt.Errorf("failed to update Twitter ID: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("no rows updated")
	}

	return nil
}

// FormatIPFSURI formats IPFS hash as URI (imported from pinata_service)
func FormatIPFSURI(ipfsHash string) string {
	return fmt.Sprintf("ipfs://%s", ipfsHash)
}
