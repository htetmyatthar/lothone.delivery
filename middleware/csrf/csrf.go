// Copyright 2012 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csrf

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/htetmyatthar/lothone.delivery/middleware/session"
)

const (
	// Name of the form field the csrf token will be.
	CSRFFieldName string = "token"
	// Name of the csrf cookie token will be.
	CSRFCookieName string = "lothone_token"
)

// csrfKey is the key to use for csrf middleware. This is generated at the start of the program.
var (
	csrfKey            string
	CSRFTrustedOrigins []string
)

func init() {
	// generate new csrf key at the start of the program.
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		panic(err) // NOTE: stop if the key generation gone wrong.
	}
	csrfKey = string(key)
	log.Println("New csrf token secret key is generated: ", csrfKey)
}

// csrfTimeout is the duration for which XSRF tokens are valid.
const csrfTimeout = 13 * time.Hour // one hour more than the session idle time.

// clean sanitizes a string for inclusion in a token by replacing all ":" with "::".
func clean(s string) string {
	return strings.Replace(s, `:`, `::`, -1)
}

// Generate returns a URL-safe secure XSRF token that expires in 1 hour.
// And also adds the CSRF cookie to the w to implement double submit cookie pattern.
//
// path is the cookie path.("Be careful of the trailing slashes('/')")
// key is a secret key for your application; it must be non-empty.
// userID is an optional unique identifier for the user.
func Generate(w http.ResponseWriter, path, userID string) string {
	token := generateTokenAtTime(csrfKey, userID, "", time.Now())
	// Create a new cookie with the CSRF token
	cookie := &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     path,
		HttpOnly: true, // Allow JavaScript to read the cookie for double submit
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   int(csrfTimeout), // Cookie expires in 1 hour
	}
	http.SetCookie(w, cookie)
	return token
}

// generateTokenAtTime is like Generate, but returns a token that expires 24 hours from now.
func generateTokenAtTime(key, userID, actionID string, now time.Time) string {
	if len(key) == 0 {
		panic("zero length xsrf secret key")
	}
	// Round time up and convert to milliseconds.
	milliTime := (now.UnixNano() + 1e6 - 1) / 1e6

	h := hmac.New(sha1.New, []byte(key))
	fmt.Fprintf(h, "%s:%s:%d", clean(userID), clean(actionID), milliTime)

	// Get the no padding base64 string.
	tok := string(h.Sum(nil))
	tok = base64.RawURLEncoding.EncodeToString([]byte(tok))

	return fmt.Sprintf("%s:%d", tok, milliTime)
}

// Valid reports whether a token is a valid, unexpired token returned by Generate.
// The token is considered to be expired and invalid if it is older than the default Timeout.
func Valid(token, userID, actionID string) bool {
	return validTokenAtTime(token, csrfKey, userID, actionID, time.Now(), csrfTimeout)
}

// ValidFor reports whether a token is a valid, unexpired token returned by Generate.
// The token is considered to be expired and invalid if it is older than the timeout duration.
func ValidFor(token, key, userID, actionID string, timeout time.Duration) bool {
	return validTokenAtTime(token, csrfKey, userID, actionID, time.Now(), timeout)
}

// validTokenAtTime reports whether a token is valid at the given time.
func validTokenAtTime(token, key, userID, actionID string, now time.Time, timeout time.Duration) bool {
	if len(key) == 0 {
		panic("zero length xsrf secret key")
	}
	// Extract the issue time of the token.
	sep := strings.LastIndex(token, ":")
	if sep < 0 {
		return false
	}
	millis, err := strconv.ParseInt(token[sep+1:], 10, 64)
	if err != nil {
		return false
	}
	issueTime := time.Unix(0, millis*1e6)

	// Check that the token is not expired.
	if now.Sub(issueTime) >= timeout {
		return false
	}

	// Check that the token is not from the future.
	// Allow 1 minute grace period in case the token is being verified on a
	// machine whose clock is behind the machine that issued the token.
	if issueTime.After(now.Add(1 * time.Minute)) {
		return false
	}

	expected := generateTokenAtTime(key, userID, actionID, issueTime)
	log.Println("\n\n\n\n\n")
	log.Println("csrf token is generated for expectation:\n key: ", key, "\nuser:", userID, "\naction: ", actionID, "\ntime: ", issueTime)
	log.Println("\n\n\n\n\n")
	log.Println("\nexpected: ", expected, "\nresult: ", token)
	log.Println("\n\n\n\n\n")

	// Check that the token matches the expected value.
	// Use constant time comparison to avoid timing attacks.
	return subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}

// CSRFMiddleware prevents csrf attacks via checking it has the valid csrf using
// double submit cookie pattern (cookie + [body/header])
func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Skip CSRF check for GET, HEAD, OPTIONS, TRACE as they're typically safe
		if r.Method == http.MethodGet || r.Method == http.MethodHead ||
			r.Method == http.MethodOptions || r.Method == http.MethodTrace {
			next.ServeHTTP(w, r)
			return
		}

		// Check origin for HTTPS connections
		if r.URL.Scheme == "https" {
			referer, err := url.Parse(r.Referer())
			if err != nil || referer.String() == "" {
				log.Printf("Invalid or missing referer: %v", err)
				unauthorizedHandler(w, "No referer")
				return
			}

			valid := (r.URL.Scheme == referer.Scheme && r.URL.Host == referer.Host)
			if !valid {
				valid = slices.Contains(CSRFTrustedOrigins, referer.Host)
			}

			if !valid {
				log.Printf("Invalid referer: %s", referer.String())
				unauthorizedHandler(w, "Bad referer")
				return
			}
		}

		// Get CSRF token from cookie
		cookieToken, err := r.Cookie(CSRFCookieName)
		if err != nil {
			log.Printf("CSRF cookie not found: %v", err)
			// Redirect to GET endpoint to generate new token
			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
			return
		}

		// Get CSRF token from request (header/form)
		token, err := getToken(r)
		if err != nil || token == "" {
			log.Printf("CSRF token not present in request: %v", err)
			// Redirect to GET endpoint to generate new token
			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
			return
		}

		// Compare tokens using constant-time comparison
		if subtle.ConstantTimeCompare([]byte(cookieToken.Value), []byte(token)) != 1 {
			log.Printf("CSRF tokens don't match. Cookie: %s, Request: %s",
				cookieToken.Value, token)
			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
			return
		}

		sToken := session.GetSessionMgr().Token(r.Context())
		// Validate token signature and expiration
		/*
			actionID := r.Method + r.URL.String()
			if r.URL.String() == "/accounts" { // for accounts path.
				actionID = AccountsActionID
			}
		*/
		actionID := ""
		if !Valid(token, sToken, actionID) {
			http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// unauthorizedhandler sets a HTTP 403 Forbidden status and writes the
// CSRF failure reason to the response.
func unauthorizedHandler(w http.ResponseWriter, reason string) {
	http.Error(w, http.StatusText(http.StatusForbidden)+reason, http.StatusForbidden)
}

// getToken gets the csrf token from the http header or form to check with csrf cookie.
func getToken(r *http.Request) (string, error) {
	// 1. Check the form value first.
	issued := r.FormValue(CSRFFieldName)

	// 2. Fall back to get the token inside the header.
	if issued == "" {
		issued = r.Header.Get("X-CSRF-TOKEN")
	}

	// 3. Finally, fall back to the multipart form (if set).
	if issued == "" && r.MultipartForm != nil {
		vals := r.MultipartForm.Value[CSRFFieldName]

		if len(vals) > 0 {
			issued = vals[0]
		}
	}

	// Return nil (equivalent to empty byte slice) if no token was found
	if issued == "" {
		return "", nil
	}
	return issued, nil
}
