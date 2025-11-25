package handlers

import (
	"net/http"

	"github.com/go-chi/chi"
	chimiddle "github.com/go-chi/chi/middleware"
)

// in this file, I need to setup the handler. While it typically uses middleware, and we re-route to that here, we don't have any middleware! there is no permissions-based
// things that we need to handle here.

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

	// setup route (no middleware from our end)
	// Need to implement the PostContent function still!

	r.Route("/post", func(router chi.Router) {
		// implementation for this endpoint will be done in FFI.go which parses the JSON, and implements the FFI bridge to send the information to our C++ engine.
		router.Post("/Trade", Trade)
		// router.Post("/Cancel", Cancel)
	})

}
