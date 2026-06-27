package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/saintedlama/degubase/internal/identity"
	httplib "github.com/saintedlama/degubase/internal/infrastructure/http"
	"github.com/saintedlama/degubase/internal/models"
	"golang.org/x/crypto/bcrypt"
)

const authCookie = "auth"

type jwtClaims struct {
	UserID  int64 `json:"uid"`
	IsAdmin bool  `json:"adm"`
	jwt.RegisteredClaims
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func parseJWT(tokenStr string, secret []byte) (*jwtClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &jwtClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return secret, nil
	})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	claims, ok := token.Claims.(*jwtClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}
	return claims, nil
}

// ── Middleware ────────────────────────────────────────────────────────────────

func requireAuth(s identity.Store, jwtSecret []byte) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
				raw := strings.TrimPrefix(auth, "Bearer ")
				if strings.HasPrefix(raw, "degu_") {
					t, err := s.GetWorkspaceTokenByHash(r.Context(), hashToken(raw))
					if err != nil || t == nil {
						httplib.Unauthorized(w)
						return
					}
					ctx := context.WithValue(r.Context(), httplib.CtxTokenWsID, t.WorkspaceID)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
				claims, err := parseJWT(raw, jwtSecret)
				if err != nil {
					httplib.Unauthorized(w)
					return
				}
				ctx := context.WithValue(r.Context(), httplib.CtxUser, &models.User{ID: claims.UserID, IsAdmin: claims.IsAdmin})
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			cookie, err := r.Cookie(authCookie)
			if err != nil {
				httplib.Unauthorized(w)
				return
			}
			claims, err := parseJWT(cookie.Value, jwtSecret)
			if err != nil {
				httplib.Unauthorized(w)
				return
			}
			ctx := context.WithValue(r.Context(), httplib.CtxUser, &models.User{ID: claims.UserID, IsAdmin: claims.IsAdmin})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := httplib.UserFromCtx(r)
		if u == nil || !u.IsAdmin {
			httplib.Forbidden(w)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func ensureDefaultUser(s identity.Store) *models.User {
	ctx := context.Background()
	u, err := s.GetUserByUsername(ctx, "system")
	if err == nil && u != nil {
		return u
	}
	n, err := s.CountUsers(ctx)
	if err != nil || n > 0 {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("system"), bcrypt.DefaultCost)
	if err != nil {
		panic("generate bcrypt hash: " + err.Error())
	}
	u, err = s.CreateUser(ctx, "system", "System", string(hash), false)
	if err != nil {
		panic("create default user: " + err.Error())
	}
	return u
}

func withDefaultUser(user *models.User) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), httplib.CtxUser, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
