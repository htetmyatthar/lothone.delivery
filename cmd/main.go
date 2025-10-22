package main

import (
	"context"
	"log"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	// "github.com/htetmyatthar/lothone.delivery/handler"
	"github.com/htetmyatthar/lothone.delivery/internal/config"
	// "github.com/htetmyatthar/lothone.delivery/internal/utils"
	"github.com/htetmyatthar/lothone.delivery/middleware/auth"
	"github.com/htetmyatthar/lothone.delivery/middleware/csrf"
	"github.com/htetmyatthar/lothone.delivery/middleware/session"
)

// HACK: SERVER UUID is always unique on each server and should not be the same on one server.

// BUG: add server status feature
// BUG: add auto installation feature

var (
	staticHandler http.Handler = utils.InitStaticServer()
)

func main() {
	// The HTTP Server
	server := &http.Server{Addr: *config.WebHostIP+*config.WebPort, Handler: service()}

	// Server run context
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig

		// Shutdown signal with grace period of 10 seconds
		shutdownCtx, _ := context.WithTimeout(serverCtx, 10*time.Second)

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Fatal("graceful shutdown timed out.. forcing exit.")
			}
		}()

		// Trigger graceful shutdown
		err := server.Shutdown(shutdownCtx)
		if err != nil {
			log.Fatal(err)
		}
		serverStopCtx()
	}()

	// Run the server
	err := server.ListenAndServeTLS(*config.WebCert, *config.WebKey)
	// err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}

	// Wait for server context to be stopped
	<-serverCtx.Done()
}

func service() http.Handler {
	r := chi.NewRouter()
	// NOTE: logger should always be the first.
	r.Use(middleware.Logger)
	r.Use(session.GetSessionMgr().LoadAndSave)
	r.Use(middleware.CleanPath)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.AllowContentType("application/json", "text/css", "text/javascript", "text/plain", "text/xml", "text/html", "application/x-www-form-urlencoded"))
	r.Use(middleware.Heartbeat("/ping"))
	r.Use(csrf.CSRFMiddleware)

	r.Handle("/static/*", http.StripPrefix("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract extension
		ext := strings.ToLower(filepath.Ext(r.URL.Path))
		mimeType := mime.TypeByExtension(ext)
		if mimeType != "" {
			w.Header().Set("Content-Type", mimeType)
		}

		// Disable caching for debugging
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
		w.Header().Set("Expires", "0")
		w.Header().Set("Pragma", "no-cache")

		staticHandler.ServeHTTP(w, r)
	})))

	// public routes.
	r.Group(func(r chi.Router) {
		r.Use(httprate.LimitByIP(20, 1*time.Minute))
		r.Get("/login", handler.Login())
		r.Post("/login", handler.LoginPOST())
	})

	// private routes.
	r.Group(func(r chi.Router) {
		r.Use(auth.AuthMiddleware)

		r.Post("/logout", handler.LogoutPOST())

		r.Get("/dashboard", handler.Dashboard())
		r.Get("/dashboard/{type}", handler.DashboardSpecific())
		r.Get("/dashboard/{type}/refresh", handler.DashboardSpecificRefresh())

		// server related.
		r.Get("/server/ip", handler.ServerIPRefresh)
		r.Get("/server/status", handler.ServerStatus)
		r.Post("/server/restart", handler.ServerRestart)

		// account related.
		r.Get("/account-form", handler.AccountFormGet())

		r.Get("/accounts/{id}/qr", handler.AccountQRGet())
		r.Get("/accounts/{id}/textkey", handler.AccountTextGet())

		r.Post("/accounts", handler.AccountCreate())
		r.Delete("/accounts", handler.AccountDelete())
		r.Put("/accounts", handler.AccountEdit())

		r.Get("/accounts/edit", handler.AccountEditGet())
	})

	return r
}
