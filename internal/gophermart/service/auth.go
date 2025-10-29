package service

import (
	"context"
	"fmt"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/handler/config"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/models"
	"github.com/alex-storchak/go-musthave-group-diploma/internal/gophermart/repository/pg"
	"gorm.io/gorm"
)

type Auth struct {
	user  *AuthUser
	token *AuthToken
}

func NewAuth(cfg *config.Config, conn *gorm.DB) *Auth {
	userRepo := pg.NewPgUser(conn)
	userService := NewAuthUser(userRepo)
	tokenService := NewAuthToken(cfg)

	return &Auth{
		user:  userService,
		token: tokenService,
	}
}

//nolint:dupl // register and login are different business processes with possible same structure
func (s *Auth) Register(ctx context.Context, login, password string) (*models.User, string, error) {
	user, err := s.user.Create(ctx, login, password)
	if err != nil {
		return nil, "", fmt.Errorf("create user: %w", err)
	}

	token, err := s.token.Generate(user.ID)
	if err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}

	return user, token, nil
}

//nolint:dupl // register and login are different business processes with possible same structure
func (s *Auth) Login(ctx context.Context, login, password string) (*models.User, string, error) {
	user, err := s.user.Authenticate(ctx, login, password)
	if err != nil {
		return nil, "", fmt.Errorf("authenticate user: %w", err)
	}

	token, err := s.token.Generate(user.ID)
	if err != nil {
		return nil, "", fmt.Errorf("generate token: %w", err)
	}

	return user, token, nil
}

func (s *Auth) ValidateUser(ctx context.Context, userID models.UserID) (*models.User, error) {
	user, err := s.user.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get user by ID: %w", err)
	}
	return user, nil
}

func (s *Auth) ValidateToken(tokenString string) (models.UserID, error) {
	userID, err := s.token.Parse(tokenString)
	if err != nil {
		return 0, fmt.Errorf("parse token: %w", err)
	}
	return userID, nil
}
