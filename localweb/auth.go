package localweb

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

const (
	minimumPasswordCharacters = 12
	maximumPasswordBytes      = 1024
	argonMemoryKiB            = 19 * 1024
	argonIterations           = 2
	argonParallelism          = 1
	passwordHashBytes         = 32
	passwordSaltBytes         = 16
	webSessionTTL             = 12 * time.Hour
	loginFailureWindow        = time.Minute
	maxClientLoginFailures    = 5
	maxGlobalLoginFailures    = 50
	maxWebSessions            = 64
	sessionCookieName         = "anytty_web_session"
)

type loginFailureState struct {
	started time.Time
	count   int
}

type webAuthenticator struct {
	salt []byte
	hash []byte
	now  func() time.Time

	verifyMu sync.Mutex
	mu       sync.Mutex
	sessions map[string]time.Time
	failures map[string]loginFailureState
	global   loginFailureState
}

// ValidatePassword applies the Local Web passphrase policy without retaining
// the supplied bytes.
func ValidatePassword(password []byte) error {
	if !utf8.Valid(password) {
		return fmt.Errorf("Web access password must be valid UTF-8")
	}
	if utf8.RuneCount(password) < minimumPasswordCharacters {
		return fmt.Errorf("Web access password must contain at least %d characters", minimumPasswordCharacters)
	}
	if len(password) > maximumPasswordBytes {
		return fmt.Errorf("Web access password must not exceed %d bytes", maximumPasswordBytes)
	}
	if len(bytes.TrimSpace(password)) == 0 {
		return fmt.Errorf("Web access password must not be blank")
	}
	return nil
}

func newWebAuthenticator(password []byte) (*webAuthenticator, error) {
	if len(password) == 0 {
		return nil, nil
	}
	if err := ValidatePassword(password); err != nil {
		return nil, err
	}
	salt := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate Web password salt: %w", err)
	}
	candidate := append([]byte(nil), password...)
	hash := argon2.IDKey(candidate, salt, argonIterations, argonMemoryKiB, argonParallelism, passwordHashBytes)
	clear(candidate)
	return &webAuthenticator{
		salt: salt, hash: hash, now: time.Now,
		sessions: make(map[string]time.Time), failures: make(map[string]loginFailureState),
	}, nil
}

func (auth *webAuthenticator) authenticate(clientKey string, password []byte) (string, time.Duration, bool, error) {
	if auth == nil {
		return "", 0, false, fmt.Errorf("Web password protection is not enabled")
	}
	auth.verifyMu.Lock()
	defer auth.verifyMu.Unlock()

	now := auth.now().UTC()
	if retryAfter := auth.retryAfter(clientKey, now); retryAfter > 0 {
		return "", retryAfter, false, nil
	}
	candidate := argon2.IDKey(password, auth.salt, argonIterations, argonMemoryKiB, argonParallelism, passwordHashBytes)
	matched := subtle.ConstantTimeCompare(candidate, auth.hash) == 1
	clear(candidate)
	if !matched {
		auth.recordFailure(clientKey, now)
		return "", 0, false, nil
	}

	token, err := randomToken()
	if err != nil {
		return "", 0, false, err
	}
	auth.mu.Lock()
	delete(auth.failures, clientKey)
	auth.pruneSessionsLocked(now)
	auth.evictOldestSessionLocked()
	auth.sessions[token] = now.Add(webSessionTTL)
	auth.mu.Unlock()
	return token, 0, true, nil
}

func (auth *webAuthenticator) authenticated(request *http.Request) bool {
	if auth == nil {
		return true
	}
	cookie, err := request.Cookie(sessionCookieName)
	if err != nil || len(cookie.Value) != 43 {
		return false
	}
	now := auth.now().UTC()
	auth.mu.Lock()
	defer auth.mu.Unlock()
	expires, ok := auth.sessions[cookie.Value]
	if !ok || !now.Before(expires) {
		delete(auth.sessions, cookie.Value)
		return false
	}
	return true
}

func (auth *webAuthenticator) retryAfter(clientKey string, now time.Time) time.Duration {
	auth.mu.Lock()
	defer auth.mu.Unlock()
	if auth.global.started.IsZero() || !now.Before(auth.global.started.Add(loginFailureWindow)) {
		auth.global = loginFailureState{started: now}
		clear(auth.failures)
	}
	if retryAfter := failureRetryAfter(auth.global, maxGlobalLoginFailures, now); retryAfter > 0 {
		return retryAfter
	}
	client := currentFailureWindow(auth.failures[clientKey], now)
	return failureRetryAfter(client, maxClientLoginFailures, now)
}

func (auth *webAuthenticator) recordFailure(clientKey string, now time.Time) {
	auth.mu.Lock()
	defer auth.mu.Unlock()
	client := currentFailureWindow(auth.failures[clientKey], now)
	client.count++
	auth.failures[clientKey] = client
	auth.global = currentFailureWindow(auth.global, now)
	auth.global.count++
}

func currentFailureWindow(state loginFailureState, now time.Time) loginFailureState {
	if state.started.IsZero() || !now.Before(state.started.Add(loginFailureWindow)) {
		return loginFailureState{started: now}
	}
	return state
}

func failureRetryAfter(state loginFailureState, maximum int, now time.Time) time.Duration {
	if state.count < maximum {
		return 0
	}
	remaining := state.started.Add(loginFailureWindow).Sub(now)
	if remaining <= 0 {
		return 0
	}
	return remaining
}

func (auth *webAuthenticator) pruneSessionsLocked(now time.Time) {
	for token, expires := range auth.sessions {
		if !now.Before(expires) {
			delete(auth.sessions, token)
		}
	}
}

func (auth *webAuthenticator) evictOldestSessionLocked() {
	if len(auth.sessions) < maxWebSessions {
		return
	}
	var oldestToken string
	var oldestExpiry time.Time
	for token, expires := range auth.sessions {
		if oldestToken == "" || expires.Before(oldestExpiry) {
			oldestToken = token
			oldestExpiry = expires
		}
	}
	delete(auth.sessions, oldestToken)
}

func (auth *webAuthenticator) close() {
	if auth == nil {
		return
	}
	auth.verifyMu.Lock()
	defer auth.verifyMu.Unlock()
	auth.mu.Lock()
	defer auth.mu.Unlock()
	clear(auth.salt)
	clear(auth.hash)
	clear(auth.sessions)
	clear(auth.failures)
}

func loginClientKey(request *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(request.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		if ip := net.ParseIP(forwarded); ip != nil {
			return ip.String()
		}
	}
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil {
		if ip := net.ParseIP(host); ip != nil {
			return ip.String()
		}
		return host
	}
	return request.RemoteAddr
}
