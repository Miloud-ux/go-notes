package main

import (
	"database/sql"
	"errors"
	"log"
	"strconv"
	"sync"
)

type Store interface {
	Add(text string) Note
	Get(id string) (Note, bool)
	List() []Note
	Del(id string) bool
}

type MemoryStore struct {
	mu    sync.RWMutex
	notes map[string]Note
	next  int
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

// compile-time trick to check if all structs are implementing the Store interface
var _ Store = (*SQLStore)(nil)
var _ Store = (*MemoryStore)(nil)
