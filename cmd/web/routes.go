package main

import (
	"context"
	"net/http"
	"strings"
)

// routes registra todos os endpoints públicos da API.
func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	// Middleware de autenticação para rotas de turma
	requireAuth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			token := ""
			parts := strings.SplitN(strings.TrimSpace(authHeader), " ", 2)
			if len(parts) == 2 && strings.EqualFold(parts[0], "Bearer") {
				token = strings.TrimSpace(parts[1])
			}
			if token == "" {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"missing or invalid token"}`))
				return
			}
			user, err := app.authHandler.AuthService().UserByToken(token)
			if err != nil {
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"invalid or expired token"}`))
				return
			}
			ctx := context.WithValue(r.Context(), "userID", user.ID)
			next(w, r.WithContext(ctx))
		}
	}

	// Perfil do professor (GET/POST)
	mux.HandleFunc("/api/professor/profile", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			app.profileHandler.GetProfile(w, r)
		case http.MethodPost:
			app.profileHandler.SaveProfile(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/health", app.authHandler.Health)

	mux.HandleFunc("/api/auth/responsavel/register", app.handleMethod(http.MethodPost, app.authHandler.RegisterResponsavel))
	mux.HandleFunc("/api/auth/responsavel/login", app.handleMethod(http.MethodPost, app.authHandler.LoginResponsavel))
	mux.HandleFunc("/api/auth/professor/register", app.handleMethod(http.MethodPost, app.authHandler.RegisterProfessor))
	mux.HandleFunc("/api/auth/professor/login", app.handleMethod(http.MethodPost, app.authHandler.LoginProfessor))
	mux.HandleFunc("/api/auth/me", app.handleMethod(http.MethodGet, app.authHandler.Me))
	mux.HandleFunc("/api/auth/logout", app.handleMethod(http.MethodPost, app.authHandler.Logout))

	mux.HandleFunc("/api/classes", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			app.classroomHandler.Handler.Create(w, r)
		case http.MethodGet:
			app.classroomHandler.Handler.List(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	mux.HandleFunc("/api/classes/", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		// Roteamento manual para /api/classes/:id e /api/classes/:id/students
		id := ""
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/classes/"), "/")
		if len(parts) > 0 {
			id = parts[0]
		}

		// Suporta GET /api/classes/:id/students usando o mesmo handler de listagem por sala.
		if len(parts) >= 2 && strings.EqualFold(parts[1], "students") {
			if r.Method != http.MethodGet {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			query := r.URL.Query()
			query.Set("sala", id)
			r.URL.RawQuery = query.Encode()
			app.classroomHandler.StudentHandler.ListByClassroom(w, r)
			return
		}

		r = r.WithContext(context.WithValue(r.Context(), "id", id))
		switch r.Method {
		case http.MethodGet:
			app.classroomHandler.Handler.GetByID(w, r)
		case http.MethodPut:
			app.classroomHandler.Handler.Update(w, r)
		case http.MethodDelete:
			app.classroomHandler.Handler.Delete(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/students", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			app.classroomHandler.StudentHandler.Create(w, r)
		case http.MethodGet:
			app.classroomHandler.StudentHandler.ListByClassroom(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	mux.HandleFunc("/api/students/", requireAuth(func(w http.ResponseWriter, r *http.Request) {
		id := ""
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/students/"), "/")
		if len(parts) > 0 {
			id = parts[0]
		}
		r = r.WithContext(context.WithValue(r.Context(), "studentID", id))

		switch r.Method {
		case http.MethodGet:
			app.classroomHandler.StudentHandler.GetByID(w, r)
		case http.MethodPut:
			app.classroomHandler.StudentHandler.Update(w, r)
		case http.MethodDelete:
			app.classroomHandler.StudentHandler.Delete(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))

	return app.withCORS(mux)
}

// handleMethod garante que cada rota aceite apenas o verbo HTTP esperado.
func (app *application) handleMethod(method string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Suporta preflight CORS com resposta vazia.
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != method {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = w.Write([]byte(`{"error":"method not allowed"}`))
			return
		}

		next(w, r)
	}
}

// withCORS aplica cabeçalhos para integração com o frontend React.
func (app *application) withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")

		// Responde imediatamente a requisições OPTIONS (preflight)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
