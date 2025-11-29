package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealth(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter()

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	expected := `{"status":"ok"}`
	if w.Body.String() != expected {
		t.Fatalf("unexpected body: got %q, want %q", w.Body.String(), expected)
	}
}

func TestRoot(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := NewRouter()

	req, _ := http.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	expected := `{"message":"hello from gin in vagrant VM"}`
	if w.Body.String() != expected {
		t.Fatalf("unexpected body: got %q, want %q", w.Body.String(), expected)
	}
}
