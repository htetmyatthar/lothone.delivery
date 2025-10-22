package auth

import (
	"log"
	"net/http"

	"github.com/htetmyatthar/lothone.delivery/internal/utils"
	"github.com/htetmyatthar/lothone.delivery/middleware/session"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("auth middleware is started.")
		// Proceed to next handler
		authenticated := session.GetSessionMgr().GetBool(r.Context(), utils.AuthenticatedField)

		if !authenticated {
			log.Println("Unauthenticated user redirecting to user login form")
			session.GetSessionMgr().Put(r.Context(), utils.URLAfterLogin, r.URL.String())

			// for htmx requests.
			if r.Header.Get("HX-Request") == "true" {
				log.Println("trying to handle htmx request.")
				w.Header().Set("HX-Redirect", "/login")
				w.Header().Set("HX-Push-Url", "/login")
				w.WriteHeader(http.StatusOK)
				return
			}

			// for normal html requests.
			log.Println("trying to handle html request.")
			http.Redirect(w, r, "/login", http.StatusFound)
			return

		}
		log.Println("auth middleware successfully executed.")
		next.ServeHTTP(w, r)
	})
}
