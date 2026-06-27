package http

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	nethttp "net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const AuthCookie = "auth"
const JwtTTL = 30 * 24 * time.Hour

type jwtClaims struct {
	UserID  int64 `json:"uid"`
	IsAdmin bool  `json:"adm"`
	jwt.RegisteredClaims
}

func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func GenerateAPIToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return
	}
	token = "degu_" + hex.EncodeToString(b)
	hash = HashToken(token)
	return
}

func SignJWT(secret []byte, userID int64, isAdmin bool) (string, error) {
	claims := jwtClaims{
		UserID:  userID,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(JwtTTL)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret)
}

func SetAuthCookie(w nethttp.ResponseWriter, signed string) {
	nethttp.SetCookie(w, &nethttp.Cookie{
		Name:     AuthCookie,
		Value:    signed,
		Path:     "/",
		HttpOnly: true,
		SameSite: nethttp.SameSiteLaxMode,
		MaxAge:   int(JwtTTL.Seconds()),
	})
}

func ClearAuthCookie(w nethttp.ResponseWriter) {
	nethttp.SetCookie(w, &nethttp.Cookie{
		Name:    AuthCookie,
		Value:   "",
		Path:    "/",
		MaxAge:  -1,
		Expires: time.Unix(0, 0),
	})
}
