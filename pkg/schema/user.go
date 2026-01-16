// Package schema defines universal data structures used across the Celerix platform.
package schema

import "time"

// UserRecord represents a standardized user identity within the Celerix ecosystem.
// It is typically stored in the '_system' persona under the 'users' app.
type UserRecord struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	DisplayName  string    `json:"display_name"`
	RecoveryCode string    `json:"recovery_code,omitempty"`
	LastActive   time.Time `json:"last_active"`
	CreatedAt    time.Time `json:"created_at"`
}

// AuditLog represents a standardized event log entry.
type AuditLog struct {
	Timestamp time.Time `json:"timestamp"`
	Actor     string    `json:"actor"`
	Action    string    `json:"action"`
	AppID     string    `json:"app_id"`
	PersonaID string    `json:"persona_id"`
	Details   string    `json:"details"`
}
