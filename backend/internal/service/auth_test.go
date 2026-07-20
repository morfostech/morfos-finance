package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/morfostech/morfos-finance/internal/auth"
	"github.com/morfostech/morfos-finance/internal/domain"
)

// fakeUserRepo is an in-memory UserRepo for unit tests.
type fakeUserRepo struct {
	byID    map[int64]*domain.User
	byEmail map[string]*domain.User
	nextID  int64
}

func newFakeRepo() *fakeUserRepo {
	return &fakeUserRepo{
		byID:    map[int64]*domain.User{},
		byEmail: map[string]*domain.User{},
		nextID:  1,
	}
}

func (f *fakeUserRepo) GetByID(_ context.Context, id int64) (*domain.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUserRepo) Create(_ context.Context, u *domain.User) (*domain.User, error) {
	if _, exists := f.byEmail[u.Email]; exists {
		return nil, domain.ErrEmailTaken
	}
	u.ID = f.nextID
	f.nextID++
	cp := *u
	f.byID[u.ID] = &cp
	f.byEmail[u.Email] = &cp
	return u, nil
}

func (f *fakeUserRepo) UpdatePassword(_ context.Context, id int64, hash string) error {
	u, ok := f.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	u.SenhaHash = hash
	u.MustChangePassword = false
	return nil
}

func (f *fakeUserRepo) ResetPassword(_ context.Context, id int64, hash string) error {
	u, ok := f.byID[id]
	if !ok {
		return domain.ErrNotFound
	}
	u.SenhaHash = hash
	u.MustChangePassword = true
	return nil
}

func (f *fakeUserRepo) List(_ context.Context) ([]domain.User, error) {
	out := make([]domain.User, 0, len(f.byID))
	for _, u := range f.byID {
		out = append(out, *u)
	}
	return out, nil
}

// seedUser inserts a user with a real argon2 hash for the given password.
func seedUser(t *testing.T, f *fakeUserRepo, email, password string, role domain.Role, ativo bool) *domain.User {
	t.Helper()
	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatal(err)
	}
	u, err := f.Create(context.Background(), &domain.User{
		Nome: "Teste", Email: email, SenhaHash: hash, Role: role, Ativo: ativo,
	})
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func newAuthService(f *fakeUserRepo) *AuthService {
	return NewAuthService(f, auth.NewTokenManager("segredo-de-teste", time.Hour))
}

func TestLogin(t *testing.T) {
	f := newFakeRepo()
	seedUser(t, f, "user@morfos.com", "senha-correta", domain.RoleColaborador, true)
	seedUser(t, f, "inativo@morfos.com", "senha-correta", domain.RoleColaborador, false)
	svc := newAuthService(f)

	tests := []struct {
		name    string
		email   string
		senha   string
		wantErr error
	}{
		{"sucesso", "user@morfos.com", "senha-correta", nil},
		{"email case-insensitive", "USER@morfos.com", "senha-correta", nil},
		{"senha errada", "user@morfos.com", "errada", domain.ErrInvalidCredentials},
		{"email desconhecido", "ninguem@morfos.com", "x", domain.ErrInvalidCredentials},
		{"usuario inativo", "inativo@morfos.com", "senha-correta", domain.ErrUserInactive},
		{"campos vazios", "", "", domain.ErrInvalidCredentials},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := svc.Login(context.Background(), tc.email, tc.senha)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil {
				if res == nil || res.Token == "" {
					t.Fatal("expected a token on success")
				}
			}
		})
	}
}

func TestChangePassword(t *testing.T) {
	f := newFakeRepo()
	u := seedUser(t, f, "user@morfos.com", "senha-antiga", domain.RoleColaborador, true)
	u.MustChangePassword = true
	f.byID[u.ID].MustChangePassword = true
	svc := newAuthService(f)
	ctx := context.Background()

	// senha atual errada
	if err := svc.ChangePassword(ctx, u.ID, "errada", "nova-senha-123"); !errors.Is(err, domain.ErrWrongPassword) {
		t.Fatalf("err = %v, want ErrWrongPassword", err)
	}
	// nova senha curta
	if err := svc.ChangePassword(ctx, u.ID, "senha-antiga", "curta"); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("err = %v, want ErrValidation", err)
	}
	// sucesso
	if err := svc.ChangePassword(ctx, u.ID, "senha-antiga", "nova-senha-123"); err != nil {
		t.Fatalf("change: %v", err)
	}
	if f.byID[u.ID].MustChangePassword {
		t.Error("must_change_password should be cleared after change")
	}
	// login com a nova senha
	if _, err := svc.Login(ctx, "user@morfos.com", "nova-senha-123"); err != nil {
		t.Fatalf("login com nova senha: %v", err)
	}
}

func TestCreateUserValidation(t *testing.T) {
	f := newFakeRepo()
	svc := newAuthService(f)
	ctx := context.Background()

	tests := []struct {
		name    string
		in      CreateUserInput
		wantErr error
	}{
		{"ok", CreateUserInput{"Ana", "ana@morfos.com", "senha-inicial", domain.RoleColaborador}, nil},
		{"nome vazio", CreateUserInput{"", "b@morfos.com", "senha-inicial", domain.RoleAdmin}, domain.ErrValidation},
		{"email inválido", CreateUserInput{"B", "nao-email", "senha-inicial", domain.RoleAdmin}, domain.ErrValidation},
		{"role inválido", CreateUserInput{"C", "c@morfos.com", "senha-inicial", domain.Role("root")}, domain.ErrValidation},
		{"senha curta", CreateUserInput{"D", "d@morfos.com", "123", domain.RoleAdmin}, domain.ErrValidation},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u, err := svc.CreateUser(ctx, tc.in)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if tc.wantErr == nil {
				if !u.MustChangePassword {
					t.Error("new user must be forced to change password")
				}
				if u.SenhaHash == tc.in.SenhaInicial {
					t.Error("password stored in plaintext")
				}
			}
		})
	}

	// email duplicado
	if _, err := svc.CreateUser(ctx, CreateUserInput{"Ana2", "ana@morfos.com", "senha-inicial", domain.RoleColaborador}); !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("err = %v, want ErrEmailTaken", err)
	}
}
