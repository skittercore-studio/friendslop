// Package session provides cookie-backed session middleware for friendslop.
//
// Tokens are 32 bytes of crypto-rand encoded as URL-safe base64, set in an
// httpOnly+SameSite=Lax cookie named `slop_session`. The DB-backed lookup
// resolves the cookie back to the (player, room) it belongs to.
//
// Tokens are issued once per join and never reused across rooms. Logout / leave
// invalidates the row in `players` (left_at != NULL).
package session

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/http"
)

// CookieName is the name of the session cookie set on every successful join.
const CookieName = "slop_session"

const tokenBytes = 32

// ctxKey is unexported so other packages must use the helpers below to read
// session state instead of poking context directly.
type ctxKey struct{}

// Session is the resolved (player, room) tuple derived from the cookie.
type Session struct {
	Token    string
	PlayerID string
	RoomID   string
	RoomCode string
	IsHost   bool
}

// ErrNoSession is returned by FromContext when the request has no resolved
// session attached.
var ErrNoSession = errors.New("no session on request")

// NewToken returns a fresh URL-safe base64 token (43 chars, ~256 bits).
func NewToken() (string, error) {
	buf := make([]byte, tokenBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// SetCookie writes the canonical session cookie. We deliberately omit Domain
// (host-only), use SameSite=Lax (allows top-level GET nav), and mark httpOnly.
// Secure is set when the request was served over TLS.
func SetCookie(w http.ResponseWriter, r *http.Request, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
	})
}

// ClearCookie expires the session cookie on the client.
func ClearCookie(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		MaxAge:   -1,
	})
}

// Resolver looks up a session by token in the DB. The query joins players +
// rooms so middleware does it in one round trip.
type Resolver struct {
	DB *sql.DB
}

// NewResolver returns a Resolver bound to the given DB handle.
func NewResolver(d *sql.DB) *Resolver { return &Resolver{DB: d} }

// Resolve fetches the session for the given token. Players that have left
// (left_at IS NOT NULL) are treated as no-session: their cookie is stale.
func (rs *Resolver) Resolve(ctx context.Context, token string) (*Session, error) {
	if token == "" {
		return nil, ErrNoSession
	}
	row := rs.DB.QueryRowContext(ctx, `
		SELECT p.id, p.room_id, p.is_host, r.code
		FROM players p
		JOIN rooms r ON r.id = p.room_id
		WHERE p.session_token = ? AND p.left_at IS NULL
	`, token)
	s := &Session{Token: token}
	var isHost int
	if err := row.Scan(&s.PlayerID, &s.RoomID, &isHost, &s.RoomCode); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoSession
		}
		return nil, err
	}
	s.IsHost = isHost != 0
	return s, nil
}

// OptionalSession resolves the cookie and stashes the Session on the request
// context if present, but never blocks the request — handlers behind it must
// call FromContext themselves and decide what to do with a missing session.
func (rs *Resolver) OptionalSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(CookieName)
		if err != nil || c.Value == "" {
			next.ServeHTTP(w, r)
			return
		}
		s, err := rs.Resolve(r.Context(), c.Value)
		if err == nil {
			r = r.WithContext(WithSession(r.Context(), s))
		}
		next.ServeHTTP(w, r)
	})
}

// RequireSession resolves the cookie and rejects with 401 if absent/invalid.
// Use this for endpoints that need to know the player.
func (rs *Resolver) RequireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(CookieName)
		if err != nil || c.Value == "" {
			http.Error(w, "missing session cookie", http.StatusUnauthorized)
			return
		}
		s, err := rs.Resolve(r.Context(), c.Value)
		if err != nil {
			http.Error(w, "invalid session", http.StatusUnauthorized)
			return
		}
		r = r.WithContext(WithSession(r.Context(), s))
		next.ServeHTTP(w, r)
	})
}

// WithSession attaches a Session to ctx for downstream handlers.
func WithSession(ctx context.Context, s *Session) context.Context {
	return context.WithValue(ctx, ctxKey{}, s)
}

// FromContext returns the Session attached to ctx by the middleware, or
// ErrNoSession if none was attached.
func FromContext(ctx context.Context) (*Session, error) {
	s, ok := ctx.Value(ctxKey{}).(*Session)
	if !ok || s == nil {
		return nil, ErrNoSession
	}
	return s, nil
}
