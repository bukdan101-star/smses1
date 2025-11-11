package services

import (
	"errors"
	"strings"
	"time"

	"event-management-backend/internal/config"
	"event-management-backend/internal/models"
	"event-management-backend/internal/repositories"
	"event-management-backend/internal/utils"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

type AuthService struct {
	repo *repositories.Repository
	cfg  *config.Config
}

func NewAuthService(repo *repositories.Repository, cfg *config.Config) *AuthService {
	return &AuthService{repo: repo, cfg: cfg}
}

type LoginResponse struct {
	Token string       `json:"token"`
	User  *models.User `json:"user"`
}

func (s *AuthService) Authenticate(email, password string) (*LoginResponse, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	if email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}

	user, err := s.repo.UserRepo.GetUserByEmail(email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	if err := utils.CheckPassword(password, user.Password); err != nil {
		return nil, errors.New("invalid credentials")
	}

	token, err := s.generateJWT(user)
	if err != nil {
		return nil, errors.New("failed to generate token")
	}

	return &LoginResponse{
		Token: token,
		User:  user,
	}, nil
}

func (s *AuthService) CreateUser(email, password, role string) (*models.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	role = strings.TrimSpace(strings.ToLower(role))

	// Validate role
	allowedRoles := map[string]bool{"admin": true, "organizer": true, "staff": true}
	if !allowedRoles[role] {
		return nil, errors.New("invalid role: must be admin, organizer, or staff")
	}

	// Check if user already exists
	if existing, _ := s.repo.UserRepo.GetUserByEmail(email); existing != nil {
		return nil, errors.New("email already registered")
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(password)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		ID:       uuid.New(),
		Email:    email,
		Password: hashedPassword,
		Role:     role,
	}

	if err := s.repo.UserRepo.CreateUser(user); err != nil {
		return nil, err
	}

	// Remove password from response
	user.Password = ""
	return user, nil
}

func (s *AuthService) generateJWT(user *models.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID.String(),
		"email":   user.Email,
		"role":    user.Role,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JWTSecret))
}

func (s *AuthService) GetUserProfile(userID string) (*models.User, error) {
	user, err := s.repo.UserRepo.GetUserByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	// Remove sensitive data
	user.Password = ""
	return user, nil
}
