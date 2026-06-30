package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func init() { gin.SetMode(gin.TestMode) }

func tokenForTest(t *testing.T, subject string) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   subject,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	})
	signed, err := tok.SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func serve(r http.Handler, method, path string, body string, headers map[string]string) *httptest.ResponseRecorder {
	var req *http.Request
	if body != "" {
		req, _ = http.NewRequest(method, path, bytes.NewBufferString(body))
	} else {
		req, _ = http.NewRequest(method, path, nil)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestLivez(t *testing.T) {
	w := serve(NewRouter(), "GET", "/livez", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHealth(t *testing.T) {
	w := serve(NewRouter(), "GET", "/health", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if w.Body.String() != `{"status":"ok"}` {
		t.Fatalf("unexpected body: %s", w.Body.String())
	}
}

func TestReadyz(t *testing.T) {
	w := serve(NewRouter(), "GET", "/readyz", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestVersion(t *testing.T) {
	r := NewRouterWithInfo(BuildInfo{Version: "1.2.3", Commit: "abc123", BuildTime: "2026-06-02"})
	w := serve(r, "GET", "/version", "", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	expected := `{"version":"1.2.3","commit":"abc123","build_time":"2026-06-02"}`
	if w.Body.String() != expected {
		t.Fatalf("unexpected body: got %q, want %q", w.Body.String(), expected)
	}
}

func TestNoRoute(t *testing.T) {
	w := serve(NewRouter(), "GET", "/does-not-exist", "", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
	expected := `{"error":"not found","path":"/does-not-exist"}`
	if w.Body.String() != expected {
		t.Fatalf("unexpected body: got %q, want %q", w.Body.String(), expected)
	}
}

func TestItemsRequiresAuth(t *testing.T) {
	w := serve(NewRouter(), "GET", "/v1/items", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthToken(t *testing.T) {
	w := serve(NewRouter(), "POST", "/v1/auth/token",
		`{"client_id":"demo","client_secret":"demo-secret"}`,
		map[string]string{"Content-Type": "application/json"})
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if _, ok := resp["token"]; !ok {
		t.Fatal("expected token field in response")
	}
}

func TestAuthTokenBadCredentials(t *testing.T) {
	w := serve(NewRouter(), "POST", "/v1/auth/token",
		`{"client_id":"demo","client_secret":"wrong"}`,
		map[string]string{"Content-Type": "application/json"})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestItemsCRUD(t *testing.T) {
	r := NewRouter()
	auth := map[string]string{
		"Authorization": "Bearer " + tokenForTest(t, "user1"),
		"Content-Type":  "application/json",
	}

	// Create
	w := serve(r, "POST", "/v1/items", `{"title":"buy milk","body":"2% from the corner store"}`, auth)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: expected 201, got %d; body: %s", w.Code, w.Body.String())
	}
	var created map[string]any
	json.NewDecoder(w.Body).Decode(&created)
	id := int(created["id"].(float64))

	// List
	w = serve(r, "GET", "/v1/items", "", map[string]string{"Authorization": auth["Authorization"]})
	if w.Code != http.StatusOK {
		t.Fatalf("list: expected 200, got %d", w.Code)
	}

	// Get
	w = serve(r, "GET", fmt.Sprintf("/v1/items/%d", id), "", map[string]string{"Authorization": auth["Authorization"]})
	if w.Code != http.StatusOK {
		t.Fatalf("get: expected 200, got %d", w.Code)
	}

	// Update
	w = serve(r, "PUT", fmt.Sprintf("/v1/items/%d", id), `{"title":"buy oat milk"}`, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("update: expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	// Delete
	w = serve(r, "DELETE", fmt.Sprintf("/v1/items/%d", id), "", map[string]string{"Authorization": auth["Authorization"]})
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: expected 204, got %d", w.Code)
	}

	// Confirm gone
	w = serve(r, "GET", fmt.Sprintf("/v1/items/%d", id), "", map[string]string{"Authorization": auth["Authorization"]})
	if w.Code != http.StatusNotFound {
		t.Fatalf("get after delete: expected 404, got %d", w.Code)
	}
}

func TestItemOwnershipIsolation(t *testing.T) {
	r := NewRouter()
	user1 := map[string]string{
		"Authorization": "Bearer " + tokenForTest(t, "user1"),
		"Content-Type":  "application/json",
	}
	user2 := map[string]string{
		"Authorization": "Bearer " + tokenForTest(t, "user2"),
		"Content-Type":  "application/json",
	}

	// user1 creates an item
	w := serve(r, "POST", "/v1/items", `{"title":"secret"}`, user1)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var created map[string]any
	json.NewDecoder(w.Body).Decode(&created)
	id := int(created["id"].(float64))

	// user2 cannot access it
	w = serve(r, "GET", fmt.Sprintf("/v1/items/%d", id), "", map[string]string{"Authorization": user2["Authorization"]})
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}
