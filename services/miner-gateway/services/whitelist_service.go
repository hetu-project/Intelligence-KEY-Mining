package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// WhitelistUser represents a whitelist user record
type WhitelistUser struct {
	WalletAddress string    `json:"wallet_address" db:"wallet_address"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	CreatedBy     string    `json:"created_by" db:"created_by"`
	Reason        *string   `json:"reason" db:"reason"`
	Status        string    `json:"status" db:"status"`
}

// WhitelistService manages whitelist operations
type WhitelistService struct {
	db *sql.DB
}

// NewWhitelistService creates a new whitelist service
func NewWhitelistService(db *sql.DB) *WhitelistService {
	return &WhitelistService{
		db: db,
	}
}

// IsUserWhitelisted checks if a user is in the active whitelist
func (ws *WhitelistService) IsUserWhitelisted(ctx context.Context, walletAddress string) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM whitelist_users 
		WHERE wallet_address = ? AND status = 'active'
	`

	var count int
	err := ws.db.QueryRowContext(ctx, query, walletAddress).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check whitelist status: %v", err)
	}

	return count > 0, nil
}

// AddUserToWhitelist adds a user to the whitelist
func (ws *WhitelistService) AddUserToWhitelist(ctx context.Context, walletAddress, createdBy, reason string) error {
	query := `
		INSERT INTO whitelist_users (wallet_address, created_by, reason, status)
		VALUES (?, ?, ?, 'active')
		ON DUPLICATE KEY UPDATE 
			status = 'active',
			created_by = VALUES(created_by),
			reason = VALUES(reason),
			created_at = CURRENT_TIMESTAMP
	`

	_, err := ws.db.ExecContext(ctx, query, walletAddress, createdBy, reason)
	if err != nil {
		return fmt.Errorf("failed to add user to whitelist: %v", err)
	}

	return nil
}

// RemoveUserFromWhitelist removes a user from the whitelist (sets status to inactive)
func (ws *WhitelistService) RemoveUserFromWhitelist(ctx context.Context, walletAddress string) error {
	query := `
		UPDATE whitelist_users 
		SET status = 'inactive' 
		WHERE wallet_address = ?
	`

	result, err := ws.db.ExecContext(ctx, query, walletAddress)
	if err != nil {
		return fmt.Errorf("failed to remove user from whitelist: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found in whitelist")
	}

	return nil
}

// GetWhitelistUsers returns all whitelist users with pagination
func (ws *WhitelistService) GetWhitelistUsers(ctx context.Context, status string, limit, offset int) ([]WhitelistUser, int, error) {
	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}

	if status != "" && status != "all" {
		whereClause += " AND status = ?"
		args = append(args, status)
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM whitelist_users %s", whereClause)
	var total int
	err := ws.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get total count: %v", err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT wallet_address, created_at, created_by, reason, status
		FROM whitelist_users 
		%s
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, whereClause)

	args = append(args, limit, offset)
	rows, err := ws.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query whitelist users: %v", err)
	}
	defer rows.Close()

	var users []WhitelistUser
	for rows.Next() {
		var user WhitelistUser
		var reason sql.NullString

		err := rows.Scan(
			&user.WalletAddress,
			&user.CreatedAt,
			&user.CreatedBy,
			&reason,
			&user.Status,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan whitelist user: %v", err)
		}

		if reason.Valid {
			user.Reason = &reason.String
		}

		users = append(users, user)
	}

	return users, total, nil
}
