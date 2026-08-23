package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"plataforma/internal/auth"
)

const maxRequestBodyBytes = 1 << 20

// AuthHandler concentra os endpoints HTTP de autenticação.
type AuthHandler struct {
	authService AuthService
}

// AuthService retorna a interface AuthService do handler
func (h *AuthHandler) AuthService() AuthService {
	return h.authService
}

// AuthService define o contrato mínimo de autenticação usado pelos handlers.
// A interface permite testar handlers com mocks sem depender de banco real.
type AuthService interface {
	Register(input auth.RegisterInput) (auth.User, error)
	Login(input auth.LoginInput) (string, auth.User, error)
	UserByToken(token string) (auth.User, error)
	Logout(token string)
}

type authResponsavelReader interface {
	GetResponsavelChildren(userID string) ([]auth.ResponsavelChildInfo, error)
}

type authResponsavelWriter interface {
	AddResponsavelChildren(userID string, links []auth.ResponsavelChildLinkInput) ([]auth.ResponsavelChildInfo, error)
}

// authRequest representa o payload aceito nos endpoints de autenticação.
// Campos em português e inglês são suportados para compatibilidade do frontend.
type authRequest struct {
	Nome     string `json:"nome"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Senha    string `json:"senha"`
	Password string `json:"password"`

	CodigoSala          string                 `json:"codigo_sala"`
	ClassroomCode       string                 `json:"classroom_code"`
	ClassroomAccessCode string                 `json:"classroomAccessCode"`
	CodigosFilhos       []string               `json:"codigos_filhos"`
	ChildrenCodes       []string               `json:"children_codes"`
	ChildrenCodesAlt    []string               `json:"childrenCodes"`
	VinculosFilhos      []authChildLinkRequest `json:"vinculos_filhos"`
	ChildrenLinks       []authChildLinkRequest `json:"children_links"`
	ChildrenLinksAlt    []authChildLinkRequest `json:"childrenLinks"`
}

type authChildLinkRequest struct {
	CodigoSala    string `json:"codigo_sala"`
	ClassroomCode string `json:"classroom_code"`
	CodigoFilho   string `json:"codigo_filho"`
	ChildCode     string `json:"child_code"`
}

// NewAuthHandler cria um handler autenticado com o serviço injetado.
func NewAuthHandler(authService AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// resolvedName prioriza o campo "nome" quando presente.
func (r authRequest) resolvedName() string {
	if strings.TrimSpace(r.Nome) != "" {
		return r.Nome
	}
	return r.Name
}

// resolvedPassword prioriza o campo "senha" quando presente.
func (r authRequest) resolvedPassword() string {
	if strings.TrimSpace(r.Senha) != "" {
		return r.Senha
	}
	return r.Password
}

func (r authRequest) resolvedClassroomCode() string {
	if strings.TrimSpace(r.CodigoSala) != "" {
		return r.CodigoSala
	}
	if strings.TrimSpace(r.ClassroomCode) != "" {
		return r.ClassroomCode
	}
	return r.ClassroomAccessCode
}

func (r authRequest) resolvedChildrenCodes() []string {
	if len(r.CodigosFilhos) > 0 {
		return r.CodigosFilhos
	}
	if len(r.ChildrenCodes) > 0 {
		return r.ChildrenCodes
	}
	return r.ChildrenCodesAlt
}

func (r authRequest) resolvedChildrenLinks() []auth.ResponsavelChildLinkInput {
	links := r.VinculosFilhos
	if len(links) == 0 {
		links = r.ChildrenLinks
	}
	if len(links) == 0 {
		links = r.ChildrenLinksAlt
	}

	resolved := make([]auth.ResponsavelChildLinkInput, 0, len(links))
	for _, link := range links {
		classroomCode := strings.TrimSpace(link.CodigoSala)
		if classroomCode == "" {
			classroomCode = strings.TrimSpace(link.ClassroomCode)
		}

		childCode := strings.TrimSpace(link.CodigoFilho)
		if childCode == "" {
			childCode = strings.TrimSpace(link.ChildCode)
		}

		resolved = append(resolved, auth.ResponsavelChildLinkInput{
			ClassroomCode: classroomCode,
			ChildCode:     childCode,
		})
	}

	return resolved
}

// Health expõe endpoint simples para verificação de disponibilidade da API.
func (h *AuthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// RegisterResponsavel cadastra um usuário com perfil de responsável.
func (h *AuthHandler) RegisterResponsavel(w http.ResponseWriter, r *http.Request) {
	h.registerByRole(w, r, auth.RoleResponsavel)
}

// RegisterProfessor cadastra um usuário com perfil de professor.
func (h *AuthHandler) RegisterProfessor(w http.ResponseWriter, r *http.Request) {
	h.registerByRole(w, r, auth.RoleProfessor)
}

// LoginResponsavel autentica um usuário com perfil de responsável.
func (h *AuthHandler) LoginResponsavel(w http.ResponseWriter, r *http.Request) {
	h.loginByRole(w, r, auth.RoleResponsavel)
}

// LoginProfessor autentica um usuário com perfil de professor.
func (h *AuthHandler) LoginProfessor(w http.ResponseWriter, r *http.Request) {
	h.loginByRole(w, r, auth.RoleProfessor)
}

// registerByRole centraliza o fluxo de cadastro com validações de entrada e resposta HTTP.
func (h *AuthHandler) registerByRole(w http.ResponseWriter, r *http.Request, role string) {
	req, err := parseAuthRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.authService.Register(auth.RegisterInput{
		Name:          req.resolvedName(),
		Email:         req.Email,
		Password:      req.resolvedPassword(),
		Role:          role,
		ClassroomCode: req.resolvedClassroomCode(),
		ChildrenCodes: req.resolvedChildrenCodes(),
		ChildrenLinks: req.resolvedChildrenLinks(),
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrEmailAlreadyExists):
			writeError(w, http.StatusConflict, "email already exists")
		case errors.Is(err, auth.ErrClassroomNotFound):
			writeError(w, http.StatusBadRequest, "classroom code not found")
		case errors.Is(err, auth.ErrInvalidChildrenCodes):
			writeError(w, http.StatusBadRequest, "invalid children codes for classroom")
		case errors.Is(err, auth.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid input")
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"user": user})
}

// loginByRole centraliza o fluxo de login para ambos os perfis.
func (h *AuthHandler) loginByRole(w http.ResponseWriter, r *http.Request, role string) {
	req, err := parseAuthRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, user, err := h.authService.Login(auth.LoginInput{
		Email:    req.Email,
		Password: req.resolvedPassword(),
		Role:     role,
	})
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrInvalidCredentials):
			writeError(w, http.StatusUnauthorized, "invalid credentials")
		case errors.Is(err, auth.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid input")
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": user})
}

// Me retorna os dados do usuário autenticado via token Bearer.
func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r.Header.Get("Authorization"))
	user, err := h.authService.UserByToken(token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"user": user})
}

// Logout invalida o token de sessão enviado no header Authorization.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	h.authService.Logout(token)
	w.WriteHeader(http.StatusNoContent)
}

// ResponsavelChildren retorna os filhos vinculados ao responsavel autenticado.
func (h *AuthHandler) ResponsavelChildren(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value("userID").(string)
	if strings.TrimSpace(userID) == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	reader, ok := h.authService.(authResponsavelReader)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	children, err := reader.GetResponsavelChildren(userID)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrUnauthorized):
			writeError(w, http.StatusUnauthorized, "unauthorized")
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"children": children})
}

// AddResponsavelChildren adiciona novos filhos ao responsavel autenticado.
func (h *AuthHandler) AddResponsavelChildren(w http.ResponseWriter, r *http.Request) {
	userID, _ := r.Context().Value("userID").(string)
	if strings.TrimSpace(userID) == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	req, err := parseAuthRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	links := req.resolvedChildrenLinks()
	if len(links) == 0 {
		classroomCode := strings.TrimSpace(req.resolvedClassroomCode())
		childrenCodes := req.resolvedChildrenCodes()
		for _, childCode := range childrenCodes {
			links = append(links, auth.ResponsavelChildLinkInput{ClassroomCode: classroomCode, ChildCode: childCode})
		}
	}

	writer, ok := h.authService.(authResponsavelWriter)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	children, err := writer.AddResponsavelChildren(userID, links)
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrUnauthorized):
			writeError(w, http.StatusUnauthorized, "unauthorized")
		case errors.Is(err, auth.ErrClassroomNotFound):
			writeError(w, http.StatusBadRequest, "classroom code not found")
		case errors.Is(err, auth.ErrInvalidChildrenCodes):
			writeError(w, http.StatusBadRequest, "invalid children codes for classroom")
		case errors.Is(err, auth.ErrInvalidInput):
			writeError(w, http.StatusBadRequest, "invalid input")
		default:
			writeError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"children": children})
}

// bearerToken extrai o token a partir do header "Authorization: Bearer <token>".
func bearerToken(authorizationHeader string) string {
	parts := strings.SplitN(strings.TrimSpace(authorizationHeader), " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

// writeJSON escreve uma resposta JSON padronizada com status code explícito.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError simplifica respostas de erro em formato JSON consistente.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// parseAuthRequest valida o corpo JSON e rejeita payloads grandes ou com campos desconhecidos.
func parseAuthRequest(r *http.Request) (authRequest, error) {
	defer r.Body.Close()

	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodyBytes+1))
	if err != nil {
		return authRequest{}, err
	}

	if int64(len(body)) > maxRequestBodyBytes {
		return authRequest{}, errors.New("request body too large")
	}

	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()

	var req authRequest
	if err := decoder.Decode(&req); err != nil {
		return authRequest{}, err
	}

	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return authRequest{}, errors.New("invalid request body")
	}

	return req, nil
}
