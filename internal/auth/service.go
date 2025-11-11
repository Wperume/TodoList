package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"todolist-api/internal/models"
)

var (
	ErrUserAlreadyExists = errors.New("user with this email already exists")
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive      = errors.New("user account is inactive")
	ErrRefreshTokenInvalid = errors.New("refresh token is invalid or expired")
)

// Service provides authentication operations
type Service struct {
	db        *gorm.DB
	jwtConfig *JWTConfig
}

// NewService creates a new authentication service
func NewService(db *gorm.DB, jwtConfig *JWTConfig) *Service {
	return &Service{
		db:        db,
		jwtConfig: jwtConfig,
	}
}

// Register creates a new user account
func (s *Service) Register(req *models.RegisterRequest) (*models.User, error) {
	// Check if user already exists
	var existingUser models.User
	err := s.db.Where("email = ?", req.Email).First(&existingUser).Error
	if err == nil {
		return nil, ErrUserAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("failed to check existing user: %w", err)
	}

	// Hash password
	hashedPassword, err := HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		Email:        req.Email,
		PasswordHash: hashedPassword,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         models.RoleUser, // Default role
		IsActive:     true,
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// Login authenticates a user and returns tokens
func (s *Service) Login(req *models.LoginRequest) (*models.AuthResponse, error) {
	// Find user by email
	var user models.User
	err := s.db.Where("email = ?", req.Email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Check if user is active
	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// Verify password
	if err := VerifyPassword(req.Password, user.PasswordHash); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Update last login time
	now := time.Now()
	user.LastLoginAt = &now
	s.db.Model(&user).Update("last_login_at", now)

	// Generate tokens
	return s.generateAuthResponse(&user)
}

// RefreshAccessToken generates a new access token using a refresh token
func (s *Service) RefreshAccessToken(refreshTokenString string) (*models.AuthResponse, error) {
	// Hash the refresh token to compare with database
	hashedToken := hashToken(refreshTokenString)

	// Find refresh token in database
	var refreshToken models.RefreshToken
	err := s.db.Preload("User").Where("token = ?", hashedToken).First(&refreshToken).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRefreshTokenInvalid
		}
		return nil, fmt.Errorf("failed to find refresh token: %w", err)
	}

	// Validate refresh token
	if !refreshToken.IsValid() {
		return nil, ErrRefreshTokenInvalid
	}

	// Check if user is active
	if !refreshToken.User.IsActive {
		return nil, ErrUserInactive
	}

	// Generate new tokens
	return s.generateAuthResponse(&refreshToken.User)
}

// RevokeRefreshToken revokes a refresh token
func (s *Service) RevokeRefreshToken(refreshTokenString string) error {
	hashedToken := hashToken(refreshTokenString)
	now := time.Now()

	result := s.db.Model(&models.RefreshToken{}).
		Where("token = ?", hashedToken).
		Update("revoked_at", now)

	if result.Error != nil {
		return fmt.Errorf("failed to revoke token: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrRefreshTokenInvalid
	}

	return nil
}

// RevokeAllUserTokens revokes all refresh tokens for a user
func (s *Service) RevokeAllUserTokens(userID uuid.UUID) error {
	now := time.Now()

	err := s.db.Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Update("revoked_at", now).Error

	if err != nil {
		return fmt.Errorf("failed to revoke user tokens: %w", err)
	}

	return nil
}

// GetUserByID retrieves a user by ID
func (s *Service) GetUserByID(userID uuid.UUID) (*models.User, error) {
	var user models.User
	err := s.db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}

// UpdateProfile updates a user's profile information
func (s *Service) UpdateProfile(userID uuid.UUID, req *models.UpdateProfileRequest) (*models.User, error) {
	var user models.User
	err := s.db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Update fields if provided
	updates := make(map[string]interface{})
	if req.FirstName != nil {
		updates["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		updates["last_name"] = *req.LastName
	}

	if len(updates) > 0 {
		if err := s.db.Model(&user).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("failed to update profile: %w", err)
		}
	}

	return &user, nil
}

// ChangePassword changes a user's password
func (s *Service) ChangePassword(userID uuid.UUID, req *models.ChangePasswordRequest) error {
	var user models.User
	err := s.db.Where("id = ?", userID).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return fmt.Errorf("failed to find user: %w", err)
	}

	// Verify current password
	if err := VerifyPassword(req.CurrentPassword, user.PasswordHash); err != nil {
		return ErrInvalidCredentials
	}

	// Hash new password
	hashedPassword, err := HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	if err := s.db.Model(&user).Update("password_hash", hashedPassword).Error; err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Revoke all existing refresh tokens for security
	_ = s.RevokeAllUserTokens(userID)

	return nil
}

// CleanupExpiredTokens removes expired refresh tokens from the database
func (s *Service) CleanupExpiredTokens() error {
	err := s.db.Where("expires_at < ?", time.Now()).Delete(&models.RefreshToken{}).Error
	if err != nil {
		return fmt.Errorf("failed to cleanup expired tokens: %w", err)
	}
	return nil
}

// Private helper methods

// generateAuthResponse generates access and refresh tokens for a user
func (s *Service) generateAuthResponse(user *models.User) (*models.AuthResponse, error) {
	// Generate access token
	accessToken, err := GenerateAccessToken(user, s.jwtConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshTokenString, err := GenerateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token in database (hashed)
	refreshToken := &models.RefreshToken{
		UserID:    user.ID,
		Token:     hashToken(refreshTokenString),
		ExpiresAt: time.Now().Add(s.jwtConfig.RefreshTokenDuration),
	}

	if err := s.db.Create(refreshToken).Error; err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Build response
	return &models.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshTokenString,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.jwtConfig.AccessTokenDuration.Seconds()),
		User: &models.UserInfo{
			ID:        user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Role:      user.Role,
		},
	}, nil
}

// hashToken creates a SHA-256 hash of a token for storage
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
