// Package models provides role management for authorization.
package models

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jheysaaz/snippy-backend/app/database"
)

// Role represents a user role for authorization
type Role struct {
	CreatedAt   time.Time `json:"createdAt"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	ID          int       `json:"id"`
}

// UserRole represents the assignment of a role to a user
type UserRole struct {
	AssignedAt time.Time `json:"assignedAt"`
	AssignedBy *string   `json:"assignedBy,omitempty"`
	UserID     string    `json:"userId"`
	RoleName   string    `json:"roleName"`
	RoleID     int       `json:"roleId"`
}

// Predefined role names
const (
	RoleAdmin   = "admin"
	RoleUser    = "user"
	RoleTester  = "tester"
	RolePremium = "premium"
)

// GetUserRoles retrieves all roles for a specific user.
func GetUserRoles(ctx context.Context, userID string) ([]UserRole, error) {
	query := `
		SELECT ur.user_id, ur.role_id, r.name, ur.assigned_at, ur.assigned_by
		FROM user_roles ur
		JOIN roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1
		ORDER BY ur.assigned_at DESC
	`

	rows, err := database.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			fmt.Printf("error closing user roles rows: %v\n", closeErr)
		}
	}()

	var userRoles []UserRole
	for rows.Next() {
		var ur UserRole
		var assignedBy sql.NullString

		err := rows.Scan(&ur.UserID, &ur.RoleID, &ur.RoleName, &ur.AssignedAt, &assignedBy)
		if err != nil {
			return nil, err
		}

		if assignedBy.Valid {
			ur.AssignedBy = &assignedBy.String
		}

		userRoles = append(userRoles, ur)
	}

	return userRoles, rows.Err()
}

// GetUserRoleNames retrieves just the role names for a user (for JWT claims).
func GetUserRoleNames(ctx context.Context, userID string) ([]string, error) {
	query := `
		SELECT r.name
		FROM user_roles ur
		JOIN roles r ON ur.role_id = r.id
		WHERE ur.user_id = $1
	`

	rows, err := database.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			fmt.Printf("error closing role names rows: %v\n", closeErr)
		}
	}()

	var roles []string
	for rows.Next() {
		var roleName string
		if err := rows.Scan(&roleName); err != nil {
			return nil, err
		}
		roles = append(roles, roleName)
	}

	return roles, rows.Err()
}

// HasRole checks if a user has a specific role.
func HasRole(ctx context.Context, userID, roleName string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM user_roles ur
			JOIN roles r ON ur.role_id = r.id
			WHERE ur.user_id = $1 AND r.name = $2
		)
	`

	var exists bool
	err := database.DB.QueryRowContext(ctx, query, userID, roleName).Scan(&exists)
	return exists, err
}

// HasAnyRole checks if a user has any of the specified roles.
func HasAnyRole(ctx context.Context, userID string, roleNames []string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM user_roles ur
			JOIN roles r ON ur.role_id = r.id
			WHERE ur.user_id = $1 AND r.name = ANY($2)
		)
	`

	var exists bool
	err := database.DB.QueryRowContext(ctx, query, userID, roleNames).Scan(&exists)
	return exists, err
}

// AssignRole assigns a role to a user.
func AssignRole(ctx context.Context, userID, roleName string, assignedBy *string) error {
	// Get role ID by name
	var roleID int
	err := database.DB.QueryRowContext(ctx, `SELECT id FROM roles WHERE name = $1`, roleName).Scan(&roleID)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("role '%s' not found", roleName)
		}
		return err
	}

	// Insert user role assignment
	query := `
		INSERT INTO user_roles (user_id, role_id, assigned_by)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, role_id) DO NOTHING
	`

	_, err = database.DB.ExecContext(ctx, query, userID, roleID, assignedBy)
	return err
}

// RevokeRole removes a role from a user.
func RevokeRole(ctx context.Context, userID, roleName string) error {
	query := `
		DELETE FROM user_roles
		WHERE user_id = $1 AND role_id = (SELECT id FROM roles WHERE name = $2)
	`

	result, err := database.DB.ExecContext(ctx, query, userID, roleName)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user does not have role '%s'", roleName)
	}

	return nil
}

// GetAllRoles retrieves all available roles.
func GetAllRoles(ctx context.Context) ([]Role, error) {
	query := `
		SELECT id, name, description, created_at
		FROM roles
		ORDER BY name
	`

	rows, err := database.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			fmt.Printf("error closing roles rows: %v\n", closeErr)
		}
	}()

	var roles []Role
	for rows.Next() {
		var role Role
		err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}

	return roles, rows.Err()
}

// GetRoleByName retrieves a role by its name.
func GetRoleByName(ctx context.Context, name string) (*Role, error) {
	query := `
		SELECT id, name, description, created_at
		FROM roles
		WHERE name = $1
	`

	var role Role
	err := database.DB.QueryRowContext(ctx, query, name).Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("role '%s' not found", name)
		}
		return nil, err
	}

	return &role, nil
}

// HasPermission checks if a user has a specific permission based on their roles.
// Admin role has access to all permissions.
func HasPermission(ctx context.Context, userID, permission string) (bool, error) {
	// Check if user has admin role (admins have all permissions)
	hasAdmin, err := HasRole(ctx, userID, RoleAdmin)
	if err != nil {
		return false, err
	}
	if hasAdmin {
		return true, nil
	}

	// Check if user has the specific permission through any of their roles
	query := `
		SELECT EXISTS(
			SELECT 1
			FROM user_roles ur
			JOIN role_permissions rp ON ur.role_id = rp.role_id
			JOIN permissions p ON rp.permission_id = p.id
			WHERE ur.user_id = $1 AND p.name = $2
		)
	`

	var hasPermission bool
	err = database.DB.QueryRowContext(ctx, query, userID, permission).Scan(&hasPermission)
	return hasPermission, err
}
