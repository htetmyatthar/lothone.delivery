package session

import (
	"log"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/htetmyatthar/lothone.delivery/internal/config"
)

var (
	sessionMgr *scs.SessionManager
)

const (
	sessionName string = "lothone_id"
)

func init() {
	sessionMgr = scs.New()

	sessionMgr.Lifetime = 36 * time.Hour
	sessionMgr.IdleTimeout = 12 * time.Hour

	// Cookie configuration
	sessionMgr.Cookie.Name = config.SessionCookieName
	sessionMgr.Cookie.Domain = *config.WebHost

	sessionMgr.Cookie.HttpOnly = true
	sessionMgr.Cookie.Persist = false
	sessionMgr.Cookie.Secure = true
	sessionMgr.HashTokenInStore = true

	sessionMgr.Cookie.SameSite = http.SameSiteStrictMode
	sessionMgr.Cookie.Path = "/"

	// Log for debugging
	log.Printf("Session cookie configured: Secure=%v, SameSite=%v, Path=%s, Hostname:%s", sessionMgr.Cookie.Secure, sessionMgr.Cookie.SameSite, sessionMgr.Cookie.Path, sessionMgr.Cookie.Domain)
}

// GetSessionMgr returns the singleton session manager for the application.
func GetSessionMgr() *scs.SessionManager {
	return sessionMgr
}
