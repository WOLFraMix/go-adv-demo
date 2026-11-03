package auth_test

import (
	"go-adv-demo/internal/auth"
	"go-adv-demo/internal/user"
	"testing"
)

type MockUserRepository struct{}

func (repo *MockUserRepository) Create(u *user.User) (*user.User, error) {
	return &user.User{
		Email: "regai@ya.ru",
	}, nil
}

func (repo *MockUserRepository) FindByEmail(email string) (*user.User, error) {
	return nil, nil
}

func TestRegisterSuccess(t *testing.T) {
	authService := auth.NewAuthService(&MockUserRepository{})
	email, err := authService.Register("regai@ya.ru", "111", "BlackMamba")
	if err != nil {
		t.Fatal(err)
	}
	if email != "regai@ya.ru" {
		t.Fatal("Email do not match")
	}
}
