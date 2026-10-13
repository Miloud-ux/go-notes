package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"time"
	"uuid"

	"golang.org/x/crypto/bcrypt"
)

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "Everything works fine")
}

func createNotes(s Store) http.HandlerFunc {
	// Decode the request and extract the note text
	// create a new note

	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "bad json", http.StatusBadRequest)
			return
		}

		if body.Text == "" {
			http.Error(w, "Empty note", http.StatusBadRequest)
			return
		}

		n := s.Add(body.Text)
		jsonWrite(w, http.StatusCreated, n)
	}
}

func getNote(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		strid := r.PathValue("id")

		n, ok := s.Get(strid)
		if !ok {
			jsonWrite(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}

		jsonWrite(w, http.StatusOK, n)

	}
}

func listNotes(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// fetch all notes to a notes array
		// json encode the array & send

		notes := s.List()
		if len(notes) == 0 {
			jsonWrite(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}

		jsonWrite(w, http.StatusOK, notes)
	}
}

func deleteNote(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !s.Del(id) {
			jsonWrite(w, http.StatusNotFound, map[string]string{"error": "not found"})
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusNoContent)
		fmt.Fprintln(w, "Deleted note with id", id)
	}
}

func getRegisterUser(s Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "public/index.html")
	}
}

func registerUser(s Store) http.HandlerFunc {
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

// NewMux small wrapper for testing
func NewMux(s Store) *http.ServeMux {
	mux := http.NewServeMux()
	// server ping
	mux.HandleFunc("GET /health", handleHealth)

	// Notes API
	mux.HandleFunc("POST /notes", createNotes(s))
	mux.HandleFunc("GET /notes/{id}", getNote(s))
	mux.HandleFunc("GET /notes", listNotes(s))
	mux.HandleFunc("DELETE /notes/{id}", deleteNote(s))

	// User API
	mux.HandleFunc("GET /register", getRegisterUser(s))
	mux.HandleFunc("POST /register", registerUser(s))
	return mux
}

// Whoa adding middleware

type ctxKey string

const requestIDKey ctxKey = "requestID"

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New().String()
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		w.Header().Set("x-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		id, _ := r.Context().Value(requestIDKey).(string)
		next.ServeHTTP(w, r)
		log.Printf("ID: %s, METHOD: %s,URL PATH: %s,Duration: %v", id, r.Method, r.URL.Path, time.Since(start))
	})
}

func recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			// recover from err
			if err := recover(); err != nil {
				log.Printf("Panic: %v\n%s", err, debug.Stack())
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func main() {
	addr := flag.String("addr", "8000", "listen addrees")
	flag.Parse()

	parsedAddr := ":" + *addr

	store, err := NewSQLStore("notes.db")

	if err != nil {
		log.Fatal(err)
	}

	mux := NewMux(store)

	srv := &http.Server{
		Addr:         parsedAddr,
		Handler:      requestID(logging(recovery(mux))),
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		fmt.Println("Listening on addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("ListenAndServe() err: %s\n", err)
		}
	}()

	// graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt)

	<-stopChan
	log.Printf("Interrupted! Shutting down server ")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	srv.Shutdown(ctx)
	log.Println("server gracefully stopped")
}

/* In-memory db to store notes */

type Note struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

/* Creates a new memory store (for testing only) */
func newStore() *MemoryStore {
	return &MemoryStore{notes: map[string]Note{}, next: 0}
}

func jsonWrite(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
