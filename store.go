package main

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"log"
	"net/mail"
	"strconv"
	"sync"
	"time"
)

type Store interface {
	Add(text string) Note
	Get(id string) (Note, bool)
	List() []Note
	Del(id string) bool
	CreateUser(email string, passwordHash string) (User, error)
	GetUserByEmail(email string) (User, bool)
	CreateSession(id string) (Session, error)
}

type MemoryStore struct {
	mu       sync.RWMutex
	notes    map[string]Note
	users    map[string]User
	sessions map[string]Session
	next     int
}

type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Session struct {
	ID        string
	UserID    string
	CreatedAt time.Time
	ExpiresAt time.Time
}

/* User related methods */

func (s *SQLStore) CreateUser(email string, passwordHash string) (User, error) {
	// some email validation
	_, err := mail.ParseAddress(email)
	if err != nil {
		log.Printf("Invalid email address err: %v", err)
		return User{}, err
	}

	res, err := s.db.Exec(`INSERT INTO users(email, password_hash) VALUES(?,?)`, email, passwordHash)

	if err != nil {
		log.Printf("Failed to insert user into the database")
		return User{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("LastInsertId() err: %v", err)
		return User{}, err
	}

	var createdAt time.Time
	err = s.db.QueryRow(`SELECT created_at FROM users WHERE id = ?`, id).Scan(&createdAt)
	if err != nil {
		log.Printf("Database query error: %v", err)
		return User{}, err
	}

	return User{
		ID:           strconv.FormatInt(id, 10),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    createdAt,
	}, nil

}

func (s *SQLStore) GetUserByEmail(email string) (User, bool) {
	var user struct {
		ID           int
		PasswordHash string
		Email        string
		CreatedAt    time.Time
	}
	user.Email = email

	err := s.db.QueryRow(`SELECT id, password_hash, created_at FROM users WHERE email = ?`, email).Scan(&user.ID, &user.PasswordHash, &user.CreatedAt)

	if errors.Is(err, sql.ErrNoRows) {
		return User{}, false
	}

	if err != nil {
		log.Printf("query error %v", err)
		return User{}, false
	}

	return User{
		ID:           strconv.FormatInt(int64(user.ID), 10),
		Email:        email,
		PasswordHash: user.PasswordHash,
		CreatedAt:    user.CreatedAt,
	}, true

}

func (s *SQLStore) CreateSession(id string) (Session, error) {
	sid := rand.Text()

	uuid, err := strconv.Atoi(id)
	if err != nil {
		log.Printf("string to num conversion err: %v", err)
	}

	expirationDate := time.Now().Add(time.Hour * 24)
	// ms difference but i don't want to query the db twice to get the time lol
	_, err = s.db.Exec(`INSERT INTO sessions (id, user_id, expires_at) VALUES (?,?,?)`, sid, uuid, expirationDate)

	if err != nil {
		log.Printf("Failed to insert into he database: %v", err)
		return Session{}, err
	}

	var createdAt time.Time

	err = s.db.QueryRow(`SELECT created_at FROM sessions WHERE id = ?`, sid).Scan(&createdAt)
	if err != nil {
		log.Printf("Database query error: %v", err)
		return Session{}, err
	}

	return Session{
		ID:        sid,
		UserID:    id,
		CreatedAt: createdAt,
		ExpiresAt: expirationDate,
	}, nil

}

func (s *SQLStore) Get(id string) (Note, bool) {
	var n Note
	var intID int64

	err := s.db.QueryRow("SELECT id, text FROM notes WHERE id = ?", id).Scan(&intID, &n.Text)

	if errors.Is(err, sql.ErrNoRows) {
		// no match found
		return Note{}, false
	}

	if err != nil {
		log.Printf("query error %v", err)
		return Note{}, false
	}

	n.ID = strconv.FormatInt(intID, 10)
	return n, true
}

func (s *MemoryStore) Del(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, exists := s.notes[id]

	if exists {
		delete(s.notes, id)
		return true
	} else {
		return false
	}
}

func (s *SQLStore) Del(id string) bool {
	res, err := s.db.Exec("DELETE FROM notes WHERE id = ?", id)

	if err != nil {
		log.Printf("Delete error: %v", err)
		return false
	}

	n, err := res.RowsAffected()

	if err != nil {
		log.Printf("rows affected error: %v", err)
		return false
	}
	return n > 0
}

func (s *MemoryStore) List() []Note {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Note, 0, len(s.notes))
	for _, n := range s.notes {
		out = append(out, n)
	}
	return out
}

func (s *SQLStore) List() []Note {
	var out []Note
	rows, err := s.db.Query("SELECT id, text FROM notes")

	if err != nil {
		log.Printf("Error fetching rows: %v", err)
		return []Note{}
	}
	defer rows.Close()

	for rows.Next() {
		var (
			intID int64
			text  string
		)

		if err := rows.Scan(&intID, &text); err != nil {
			log.Printf("Scan error %v", err)
			return []Note{}
		}

		out = append(out, Note{
			ID:   strconv.FormatInt(intID, 10),
			Text: text,
		})
	}

	if err := rows.Err(); err != nil {
		log.Printf("rows error %v", err)
	}
	return out
}

func (s *SQLStore) Add(text string) Note {
	res, err := s.db.Exec("INSERT INTO notes(text) VALUES(?)", text)

	if err != nil {
		log.Printf("Failed to insert into he database")
		return Note{}
	}

	id, err := res.LastInsertId()
	if err != nil {
		log.Printf("Last insert id err: %v", err)
		return Note{}
	}

	return Note{
		ID:   strconv.FormatInt(id, 10),
		Text: text,
	}
}
func (s *MemoryStore) Add(text string) Note {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.next++
	n := Note{ID: strconv.Itoa(s.next), Text: text}
	s.notes[n.ID] = n
	return n
}

func (s *MemoryStore) Get(id string) (Note, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n, ok := s.notes[id]
	return n, ok
}

func (s *MemoryStore) CreateUser(email string, passwordHash string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := mail.ParseAddress(email)
	if err != nil {
		return User{}, err
	}

	s.next++
	user := User{
		ID:           strconv.Itoa(s.next),
		Email:        email,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}
	s.users[user.ID] = user
	return user, nil
}

func (s *MemoryStore) GetUserByEmail(email string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.users == nil {
		return User{}, false
	}

	for _, u := range s.users {
		if u.Email == email {
			return u, true
		}
	}
	return User{}, false
}

func (s *MemoryStore) CreateSession(id string) (Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.sessions == nil {
		s.sessions = make(map[string]Session)
	}

	sid := rand.Text()
	createdAt := time.Now()
	expirationDate := createdAt.Add(time.Hour * 24)

	session := Session{
		ID:        sid,
		UserID:    id,
		CreatedAt: createdAt,
		ExpiresAt: expirationDate,
	}

	s.sessions[sid] = session
	return session, nil
}

// compile-time trick to check if all structs are implementing the Store interface
var _ Store = (*SQLStore)(nil)
var _ Store = (*MemoryStore)(nil)
