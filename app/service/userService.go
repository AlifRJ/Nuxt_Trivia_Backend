package service

import (
	"context"
	"errors"

	"github.com/AlifRJ/Go_Auth/app/model"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrPasswordTooShort = errors.New("password must be 6 characters or longer")
	ErrEmailTaken       = errors.New("email is already registered")
	ErrUsernameTaken    = errors.New("username is already taken")
	ErrInvalidRequest    = errors.New("Invalid request payload")
)

type UserService struct {
	repo model.UserRepository
	authRepo model.AuthRepository
}

func NewUserService(repo model.UserRepository, authRepo model.AuthRepository) *UserService {
	return &UserService{repo: repo, authRepo: authRepo}
}

func (s *UserService) GetAll(ctx context.Context, limit, offset int) ([]*model.User, error) {
	if limit <= 0 {
		limit = 10 
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.GetAll(ctx, limit, offset)
}

func (s *UserService) GetByID(ctx context.Context, id uint) (*model.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) RegisterUser(ctx context.Context, name, username, email, password string) (*model.User, error) {
	// Validate empty request
	if name == "" || username == "" || email == ""{
		return nil, ErrInvalidRequest
	}

	// Validate name
	if len(name) < 3 {
		return nil, errors.New("Name is too short")
	}

	// Validate username
	if len(username) < 5 {
		return nil, errors.New("Username must be at least 6 character long")
	}
	
	// validate password
	if len(password) < 6 {
		return nil, ErrPasswordTooShort
	}

	if existingUser, _ := s.repo.GetByEmail(ctx, email); existingUser != nil {
		return nil, ErrEmailTaken
	}
	if existingUser, _ := s.repo.GetByUsername(ctx, username); existingUser != nil {
		return nil, ErrUsernameTaken
	}

	// Hash Password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	
	user := &model.User{Name: name, Username: username, Email: email, Password: string(hashedPassword)}
	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *UserService) UpdateUser(ctx context.Context, id uint, name, username, email, password string) (*model.User, error) {
	// Fetch existing user record
	existingUser, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	passwordChanged := false

	// Hash password if provided and non-empty
	if password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		existingUser.Password = string(hashedPassword)
		passwordChanged = true
	}

	if name != "" {
		existingUser.Name = name
	}
	if username != "" {
		existingUser.Username = username
	}
	if email != "" {
		existingUser.Email = email
	}

	// Persist updated user
	if err := s.repo.Update(ctx, existingUser); err != nil {
		return nil, err
	}

	// Security: If password was changed, revoke active refresh tokens across all devices
	if passwordChanged && s.authRepo != nil {
		_ = s.authRepo.DeleteRefreshToken(ctx, id)
	}

	existingUser.Password = "" // Strip hash before returning
	return existingUser, nil
}

func (s *UserService) DeleteUser(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

func (s *UserService) DeleteUserPermanently(ctx context.Context, id uint) error {
	return s.repo.PermanentDelete(ctx, id)
}
