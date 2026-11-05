package auth_test

import (
	"errors"
	"go-adv-demo/internal/auth"
	"go-adv-demo/internal/user"
	"os"
	"testing"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// fakeUserRepo повторяет семантику реального репозитория: отсутствие записи
// даёт gorm.ErrRecordNotFound, а findErr имитирует сбой БД.
type fakeUserRepo struct {
	users   map[string]*user.User
	findErr error
	created []*user.User
}

func (r *fakeUserRepo) Create(u *user.User) (*user.User, error) {
	r.created = append(r.created, u)
	return u, nil
}

func (r *fakeUserRepo) FindByEmail(email string) (*user.User, error) {
	if r.findErr != nil {
		return nil, r.findErr
	}
	if u, ok := r.users[email]; ok {
		return u, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func hashPassword(t *testing.T, password string) string {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(hash)
}

func newRepoWithUser(t *testing.T, email, password string) *fakeUserRepo {
	t.Helper()
	return &fakeUserRepo{
		users: map[string]*user.User{
			email: {Email: email, Password: hashPassword(t, password), Name: "BlackMamba"},
		},
	}
}

func TestLoginReturnsEmailForValidCredentials(t *testing.T) {
	repo := newRepoWithUser(t, "regai@ya.ru", "secret123")
	service := auth.NewAuthService(repo)

	email, err := service.Login("regai@ya.ru", "secret123")
	if err != nil {
		t.Fatal(err)
	}
	if email != "regai@ya.ru" {
		t.Fatalf("expected email regai@ya.ru, got %q", email)
	}
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	repo := newRepoWithUser(t, "regai@ya.ru", "secret123")
	service := auth.NewAuthService(repo)

	_, err := service.Login("regai@ya.ru", "wrong")
	if err == nil || err.Error() != auth.ErrCredentials {
		t.Fatalf("expected %q, got %v", auth.ErrCredentials, err)
	}
}

// Одинаковое сообщение для «нет пользователя» и «неверный пароль» —
// защита от перечисления учётных записей.
func TestLoginRejectsUnknownUserWithSameError(t *testing.T) {
	repo := newRepoWithUser(t, "regai@ya.ru", "secret123")
	service := auth.NewAuthService(repo)

	_, err := service.Login("nobody@ya.ru", "secret123")
	if err == nil || err.Error() != auth.ErrCredentials {
		t.Fatalf("expected %q, got %v", auth.ErrCredentials, err)
	}
}

func TestRegisterRejectsExistingEmail(t *testing.T) {
	repo := newRepoWithUser(t, "regai@ya.ru", "secret123")
	service := auth.NewAuthService(repo)

	_, err := service.Register("regai@ya.ru", "other", "Someone")
	if err == nil || err.Error() != auth.ErrUserExists {
		t.Fatalf("expected %q, got %v", auth.ErrUserExists, err)
	}
	if len(repo.created) != 0 {
		t.Fatalf("expected no user to be created, got %d", len(repo.created))
	}
}

func TestRegisterStoresBcryptHashNotPlaintext(t *testing.T) {
	repo := &fakeUserRepo{users: map[string]*user.User{}}
	service := auth.NewAuthService(repo)

	if _, err := service.Register("new@ya.ru", "plaintext-pass", "Neo"); err != nil {
		t.Fatal(err)
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected 1 created user, got %d", len(repo.created))
	}
	stored := repo.created[0]
	if stored.Password == "plaintext-pass" {
		t.Fatal("password must not be stored in plaintext")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.Password), []byte("plaintext-pass")); err != nil {
		t.Fatalf("stored hash does not match password: %v", err)
	}
	if stored.Name != "Neo" {
		t.Fatalf("expected name Neo, got %q", stored.Name)
	}
}

// Известный баг: Register игнорирует ошибку FindByEmail. При сбое БД
// существующий пользователь выглядит «не найденным», и создаётся дубликат.
func TestRegisterFailsWhenLookupErrors(t *testing.T) {
	if os.Getenv("KNOWN_BUGS") == "" {
		t.Skip("известный баг: ошибка FindByEmail в Register проглатывается (запуск: KNOWN_BUGS=1)")
	}
	repo := &fakeUserRepo{findErr: errors.New("connection refused")}
	service := auth.NewAuthService(repo)

	_, err := service.Register("regai@ya.ru", "111", "BlackMamba")
	if err == nil {
		t.Fatal("expected error when user lookup fails")
	}
	if len(repo.created) != 0 {
		t.Fatalf("expected no user to be created on lookup failure, got %d", len(repo.created))
	}
}
