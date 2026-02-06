package users

import (
	"context"

	er "routePlanner/internal/errors"
	"routePlanner/internal/models"
	repo "routePlanner/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo repo.UserRepository
}

func NewUserService(userRepo repo.UserRepository) *UserService {
	return &UserService{
		repo: userRepo,
	}
}

func (s *UserService) GetById(ctx context.Context, id string) (*models.User, error) {
	if id == "" {
		return nil, er.ErrInvalidUserData
	}
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	if email == "" {
		return nil, er.ErrInvalidUserData
	}
	return s.repo.GetByEmail(ctx, email)
}

func (s *UserService) Create(ctx context.Context, u *models.UserCreateRequest) (*models.User, error) {
	if u == nil || u.Name == "" || u.Email == "" || u.Password == "" {
		return nil, er.ErrInvalidUserData
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, er.ErrInternalServer
	}

	user := models.User{
		Name:         u.Name,
		Email:        u.Email,
		PasswordHash: string(hashedPassword),
	}

	if err := s.repo.Create(ctx, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserService) ValidatePassword(user *models.User, password string) bool {
	if user == nil || password == "" {
		return false
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	return err == nil
}

func (s *UserService) Update(ctx context.Context, id string, u *models.UserUpdateRequest) error {
	if u == nil || id == "" {
		return er.ErrInvalidUserData
	}
	return s.repo.Update(ctx, id, u)
}

func (s *UserService) DeleteById(ctx context.Context, id string) error {
	if id == "" {
		return er.ErrInvalidUserData
	}
	return s.repo.Delete(ctx, id)
}
