package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func GetRegisterUser(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/index.html")
	}
}

func RegisterUser(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var email, password string

		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			var body struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			email, password = body.Email, body.Password
		} else {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			email, password = r.FormValue("email"), r.FormValue("password")
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		user, err := s.CreateUser(email, string(hash))
		if err != nil {
			log.Printf("CreateUser failed: %v", err)
			http.Error(w, "could not create user", http.StatusConflict)
			return
		}
		jsonWrite(w, http.StatusCreated, user)
	}
}

func LoginUser(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var email, password string

		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			var body struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			email, password = body.Email, body.Password
		} else {
			if err := r.ParseForm(); err != nil {
				http.Error(w, "bad request", http.StatusBadRequest)
				return
			}
			email, password = r.FormValue("email"), r.FormValue("password")
		}

		user, found := s.GetUserByEmail(email)
		if !found {
			http.Error(w, "Invalid Cridentials", http.StatusBadRequest)
			return
		}

		res := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
		if res != nil {
			http.Error(w, "Invalid Cridentials", http.StatusBadRequest)
			return
		}

		// User is authed
		sesh, err := s.CreateSession(user.ID)
		if err != nil {
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		// Settin session cookie yummy
		cookie := http.Cookie{
			Name:     "session_token",
			Value:    sesh.ID,
			Path:     "/",
			Expires:  sesh.ExpiresAt,
			HttpOnly: true,                 // XSS protection
			Secure:   true,                 // Only sent over HTTPS
			SameSite: http.SameSiteLaxMode, // CSRF protection
		}

		http.SetCookie(w, &cookie)
		jsonWrite(w, http.StatusOK, "User authentificated")

	}
}

func GetLoginUser(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/login.html")
	}
}
