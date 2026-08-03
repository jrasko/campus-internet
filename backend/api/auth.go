package api

import (
	"backend/model"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/golang-jwt/jwt/v5"
)

type AuthHandler struct {
	config model.Configuration
}

type Claims struct {
	jwt.RegisteredClaims
	Username string
	Role     model.Role
}

type Permission func(role model.Role) bool

// Middleware wraps a http.HandlerFunc and checks if the caller is authenticated with a valid token
// if the caller has a valid token, the request will be forwarded to the wrapped http.HandlerFunc
func (a AuthHandler) Middleware(next http.HandlerFunc, permission Permission) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// read http authorization header
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			log.Println("[INFO] Unauthenticated access")
			w.WriteHeader(http.StatusForbidden)
			return
		}

		// parse and check jwt in header
		var claims Claims
		token, err := jwt.ParseWithClaims(
			header[7:],
			&claims,
			func(token *jwt.Token) (interface{}, error) { return getHMACSecretKey(a.config.HMACSecret), nil },
			jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Name}),
		)
		if err != nil {
			log.Printf("[WARNING] when parsing jwt: %v", err)
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if !token.Valid {
			log.Println("[DEBUG] got invalid token")
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if !permission(claims.Role) {
			log.Printf("[DEBUG] insufficient permission for role %s", claims.Role)
			w.WriteHeader(http.StatusForbidden)
			return
		}

		ctx := context.WithValue(r.Context(), model.FieldUsername, claims.Username)
		ctx = context.WithValue(ctx, model.FieldRole, claims.Role)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

var (
	expirationTime = 2 * time.Hour
)

type loginResponse struct {
	Token    string     `json:"token"`
	Role     model.Role `json:"role"`
	Username string     `json:"username"`
}

func (a AuthHandler) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// decode creadentials
		c := Credentials{}
		err := json.NewDecoder(r.Body).Decode(&c)
		if err != nil {
			log.Printf("[WARNING] when decoding login credentials: %v", err)
			w.WriteHeader(http.StatusForbidden)
			return
		}

		loginUser, err := a.checkCredentials(c)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusForbidden)
			return
		}

		// create a json web token
		token, err := CreateJWT(loginUser, a.config.HMACSecret)
		if err != nil {
			log.Printf("[ERROR] when creating jwt: %v", err)
			sendHttpError(w, err)
			return
		}

		// send token back to user
		err = json.NewEncoder(w).Encode(
			loginResponse{
				Token:    token,
				Role:     loginUser.Role,
				Username: loginUser.Username,
			},
		)
		if err != nil {
			log.Printf("[ERROR] when writing login response: %v", err)
			sendHttpError(w, err)
			return
		}
		log.Printf("[DEBUG] new login for user %s", c.Username)
	}
}

func (a AuthHandler) checkCredentials(credentials Credentials) (model.LoginUser, error) {
	// search username in configured users
	var loginUser model.LoginUser
	for _, user := range a.config.Users {
		if user.Username == credentials.Username {
			loginUser = user
		}
	}

	// check if username and password are equal
	pwCheck, err := argon2id.ComparePasswordAndHash(credentials.Password, loginUser.PasswordHash)
	// check if user is configured
	if err != nil {
		return model.LoginUser{}, fmt.Errorf("[ERROR] when checking hash: %v", err)
	}

	if loginUser == (model.LoginUser{}) {
		return model.LoginUser{}, fmt.Errorf("[DEBUG] login for invalid username %s", credentials.Username)
	}
	if !pwCheck {
		return model.LoginUser{}, fmt.Errorf("[DEBUG] wrong password for user %s", credentials.Username)
	}

	return loginUser, nil
}

func CreateJWT(user model.LoginUser, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, Claims{
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expirationTime)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	})

	return token.SignedString(getHMACSecretKey(secret))
}

func getHMACSecretKey(secret string) []byte {
	bytes, err := base64.StdEncoding.DecodeString(secret)
	if err == nil && len(bytes) >= 64 {
		return bytes
	}
	return []byte(secret)
}
