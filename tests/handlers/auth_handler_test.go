package handlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"plataforma/cmd/web/handlers"
	"plataforma/internal/auth"
)

// fakeAuthService permite simular cenários de sucesso e erro dos handlers.
type fakeAuthService struct {
	registerFn    func(input auth.RegisterInput) (auth.User, error)
	loginFn       func(input auth.LoginInput) (string, auth.User, error)
	userByTokenFn func(token string) (auth.User, error)
	logoutFn      func(token string)
}

// Register executa o comportamento configurado para o cenário de teste.
func (f *fakeAuthService) Register(input auth.RegisterInput) (auth.User, error) {
	return f.registerFn(input)
}

// Login executa o comportamento configurado para o cenário de teste.
func (f *fakeAuthService) Login(input auth.LoginInput) (string, auth.User, error) {
	return f.loginFn(input)
}

// UserByToken executa o comportamento configurado para o cenário de teste.
func (f *fakeAuthService) UserByToken(token string) (auth.User, error) {
	return f.userByTokenFn(token)
}

// Logout executa o comportamento configurado para o cenário de teste.
func (f *fakeAuthService) Logout(token string) {
	f.logoutFn(token)
}

// TestHealth valida resposta do endpoint de health-check.
func TestHealth(t *testing.T) {
	h := handlers.NewAuthHandler(&fakeAuthService{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	h.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var payload map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload["status"] != "ok" {
		t.Fatalf("expected status ok, got %q", payload["status"])
	}
}

// TestRegisterProfessorSuccess valida cadastro de professor com payload válido.
func TestRegisterProfessorSuccess(t *testing.T) {
	var captured auth.RegisterInput

	h := handlers.NewAuthHandler(&fakeAuthService{
		registerFn: func(input auth.RegisterInput) (auth.User, error) {
			captured = input
			return auth.User{
				ID:        "usr_1",
				Name:      "Ana",
				Email:     "ana@teste.com",
				Role:      auth.RoleProfessor,
				CreatedAt: time.Now().UTC(),
			}, nil
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/professor/register", strings.NewReader(`{"nome":"Ana","email":"ana@teste.com","senha":"123456"}`))
	h.RegisterProfessor(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", rec.Code)
	}

	if captured.Role != auth.RoleProfessor || captured.Name != "Ana" || captured.Password != "123456" {
		t.Fatalf("unexpected register input: %+v", captured)
	}
}

// TestLoginResponsavelInvalidCredentials valida retorno 401 para credenciais inválidas.
func TestLoginResponsavelInvalidCredentials(t *testing.T) {
	h := handlers.NewAuthHandler(&fakeAuthService{
		loginFn: func(input auth.LoginInput) (string, auth.User, error) {
			return "", auth.User{}, auth.ErrInvalidCredentials
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/responsavel/login", strings.NewReader(`{"email":"resp@teste.com","senha":"errada"}`))
	h.LoginResponsavel(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

// TestMeSuccess valida acesso ao endpoint /me com token válido.
func TestMeSuccess(t *testing.T) {
	h := handlers.NewAuthHandler(&fakeAuthService{
		userByTokenFn: func(token string) (auth.User, error) {
			if token != "token123" {
				t.Fatalf("unexpected token: %s", token)
			}
			return auth.User{ID: "usr_1", Role: auth.RoleResponsavel}, nil
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer token123")
	h.Me(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

// TestMeUnauthorized valida retorno 401 quando o token não é aceito.
func TestMeUnauthorized(t *testing.T) {
	h := handlers.NewAuthHandler(&fakeAuthService{
		userByTokenFn: func(token string) (auth.User, error) {
			return auth.User{}, errors.New("boom")
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer invalid")
	h.Me(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

// TestLogout valida invalidação de sessão com token Bearer.
func TestLogout(t *testing.T) {
	calledWith := ""
	h := handlers.NewAuthHandler(&fakeAuthService{
		logoutFn: func(token string) { calledWith = token },
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.Header.Set("Authorization", "Bearer token123")
	h.Logout(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", rec.Code)
	}
	if calledWith != "token123" {
		t.Fatalf("expected logout token token123, got %q", calledWith)
	}
}

// TestLogoutUnauthorizedWithoutToken valida proteção de logout sem autorização.
func TestLogoutUnauthorizedWithoutToken(t *testing.T) {
	h := handlers.NewAuthHandler(&fakeAuthService{})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	h.Logout(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}
