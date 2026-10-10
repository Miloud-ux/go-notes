package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func testCreateThenGet(t *testing.T, store Store) {
	// create a 'createNote' request (POST /notes)
	// grab the response's note (text + id) and make sure it's StatusCreated
	// create a 'getNote' request (GET /notes/{ID}) with the previous responses ID
	// grab the new response and make sure the status code is statusok
	// make sure the new response's fields = the previous response's

	mux := NewMux(store)

	body := bytes.NewBufferString(`{"text":"hello"}`)
	req := httptest.NewRequest("POST", "/notes", body)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST: expected status code 201 but got %d instead\nBody: %s", rec.Code, rec.Body)
	}

	var created Note
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("Decoding json failed %v", err)
	}

	if created.ID == "" || created.Text != "hello" {
		t.Fatalf("unexpected note: %+v", created)
	}

	req = httptest.NewRequest("GET", "/notes/"+created.ID, nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET: expected status code 200 but got %d instead\nBody: %s", rec.Code, rec.Body)
	}

	var fetched Note
	if err := json.NewDecoder(rec.Body).Decode(&fetched); err != nil {
		t.Fatalf("Decoding json failed %v", err)
	}

	if fetched != created {
		t.Fatalf("want %+v but got %v instead", created, fetched)
	}

}

func TestMemoryCreateThenGet(t *testing.T) {
	testCreateThenGet(t, newStore())
}

func TestSQLCreateThenGet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLStore(path)
	if err != nil {
		t.Fatal(err)
	}
	testCreateThenGet(t, store)
}

func testGetMissing(t *testing.T, store Store) {
	// fetch a non existing note (e.g. ID = 999)
	// check resopnse.statusCode == 404 not found

	mux := NewMux(store)

	req := httptest.NewRequest("GET", "/notes/999", nil)
	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET: expected 404 got %d", rec.Code)
	}
}

func TestMemoryGetMissing(t *testing.T) {
	testGetMissing(t, newStore())
}

func TestSQLGetMissing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLStore(path)
	if err != nil {
		t.Fatal(err)
	}
	testGetMissing(t, store)

}

func testDelete(t *testing.T, store Store) {
	// create a note and save the ID
	// delete(ID)
	// try and getnote(ID)
	mux := NewMux(store)

	body := bytes.NewBufferString(`{"text":"hello"}`)
	req := httptest.NewRequest("POST", "/notes", body)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("POST: want 201, got %d", rec.Code)
	}

	var created Note
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}

	// delete it
	req = httptest.NewRequest("DELETE", "/notes/"+created.ID, nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("DELETE: want 204, got %d", rec.Code)
	}

	// confirm it's gone
	req = httptest.NewRequest("GET", "/notes/"+created.ID, nil)
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("after delete: want 404, got %d", rec.Code)
	}
}

func TestMemoryDelete(t *testing.T) {
	testDelete(t, newStore())
}

func TestSQLDelete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLStore(path)
	if err != nil {
		t.Fatal(err)
	}
	testDelete(t, store)
}
