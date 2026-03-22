package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"plataforma/internal/users"
	"strings"

	"go.mongodb.org/mongo-driver/mongo"
)

type ProfileHandler struct {
	Service   *users.ProfileService
	UsersRepo *users.Repository // para buscar nome/email
}

func NewProfileHandler(service *users.ProfileService, usersRepo *users.Repository) *ProfileHandler {
	return &ProfileHandler{Service: service, UsersRepo: usersRepo}
}

// POST /api/professor/profile
func (h *ProfileHandler) SaveProfile(w http.ResponseWriter, r *http.Request) {
	// Recebe o JSON genérico para extrair nome/email e os outros campos
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error":"invalid request body"}`))
		return
	}
	userID := getUserIDFromContext(r.Context())
	if strings.TrimSpace(userID) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
		return
	}
	// Atualiza nome/email na tabela users se enviados
	update := map[string]interface{}{}
	if v, ok := payload["nome"].(string); ok && strings.TrimSpace(v) != "" {
		update["name"] = v
	}
	if v, ok := payload["email"].(string); ok && strings.TrimSpace(v) != "" {
		update["email"] = v
	}
	if len(update) > 0 && h.UsersRepo != nil {
		_ = h.UsersRepo.UpdateFieldsByPublicID(r.Context(), userID, update)
	}
	// Monta struct Profile só com os campos do profile
	var req users.Profile
	req.PublicID = userID
	if v, ok := payload["photo"].(string); ok {
		req.Photo = v
	}
	if v, ok := payload["formacoes"].(string); ok {
		req.Formacoes = v
	}
	if v, ok := payload["areas_atuacao"].(string); ok {
		req.AreasAtuacao = v
	}
	if v, ok := payload["pesquisas"].(string); ok {
		req.Pesquisas = v
	}
	if v, ok := payload["descricao"].(string); ok {
		req.Descricao = v
	}
	if v, ok := payload["telefone"].(string); ok {
		req.Telefone = v
	}
	if err := h.Service.SaveProfile(r.Context(), req); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"failed to save profile"}`))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"profile saved"}`))
}

// GET /api/professor/profile
func (h *ProfileHandler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromContext(r.Context())
	if strings.TrimSpace(userID) == "" {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":"unauthorized"}`))
		return
	}
	profile, err := h.Service.GetProfile(r.Context(), userID)
	if err != nil && err != mongo.ErrNoDocuments {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"profile not found"}`))
		return
	}
	// Busca nome e email do usuário
	var name, email string
	if h.UsersRepo != nil {
		user, err := h.UsersRepo.FindByPublicID(r.Context(), userID)
		if err == nil {
			name = user.Name
			email = user.Email
		}
	}
	// Monta resposta combinando profile + nome/email
	resp := map[string]interface{}{
		"nome":          name,
		"email":         email,
		"photo":         profile.Photo,
		"formacoes":     profile.Formacoes,
		"areas_atuacao": profile.AreasAtuacao,
		"pesquisas":     profile.Pesquisas,
		"descricao":     profile.Descricao,
		"telefone":      profile.Telefone,
		"updated_at":    profile.UpdatedAt,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Função utilitária para pegar o userID do contexto
func getUserIDFromContext(ctx context.Context) string {
	userID, _ := ctx.Value("userID").(string)
	return userID
}
