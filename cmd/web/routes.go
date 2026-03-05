package main

import (
	"net/http"
)

// routes registra todos os endpoints públicos da API.
func (app *application) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", app.authHandler.Health)

	mux.HandleFunc("/api/auth/responsavel/register", app.handleMethod(http.MethodPost, app.authHandler.RegisterResponsavel))
	mux.HandleFunc("/api/auth/responsavel/login", app.handleMethod(http.MethodPost, app.authHandler.LoginResponsavel))
	mux.HandleFunc("/api/auth/professor/register", app.handleMethod(http.MethodPost, app.authHandler.RegisterProfessor))
	mux.HandleFunc("/api/auth/professor/login", app.handleMethod(http.MethodPost, app.authHandler.LoginProfessor))
	mux.HandleFunc("/api/auth/me", app.handleMethod(http.MethodGet, app.authHandler.Me))
	mux.HandleFunc("/api/auth/logout", app.handleMethod(http.MethodPost, app.authHandler.Logout))

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
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

		next.ServeHTTP(w, r)
	})
}
