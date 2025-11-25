package handlers

import (
	"net/http"

	"github.com/go-chi/chi"
	chimiddle "github.com/go-chi/chi/middleware"
)

func Handler(r *chi.Mux) {
	// strip trailing slashes (from chi package)
	r.Use(chimiddle.StripSlashes)

	// Add CORS middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if req.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, req)
		})
	})

	// setup route (MUST be /order for your test URL)
	r.Route("/order", func(router chi.Router) {
		// We use lowercase "trade" here to match URL best practices
		router.Post("/trade", Trade)
		router.Post("/cancel", Cancel)
		router.Get("/status", Status)
	})
}
