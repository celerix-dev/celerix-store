package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/celerix-dev/celerix-store/pkg/engine"
	"github.com/gin-gonic/gin"
)

func setupTestRouter() (*gin.Engine, *Handler) {
	gin.SetMode(gin.TestMode)
	store := engine.NewMemStore(nil, nil)
	h := &Handler{Store: store}
	r := gin.Default()

	r.GET("/personas", h.GetPersonas)
	r.GET("/personas/:persona/apps", h.GetApps)
	r.GET("/personas/:persona/apps/:app", h.GetAppStore)
	r.POST("/personas/:persona/apps/:app/keys/:key", h.Set)
	r.DELETE("/personas/:persona/apps/:app/keys/:key", h.Delete)
	r.POST("/move", h.Move)

	return r, h
}

func TestGetPersonas(t *testing.T) {
	r, h := setupTestRouter()
	h.Store.Set("p1", "a1", "k1", "v1")

	req, _ := http.NewRequest("GET", "/personas", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var personas []string
	json.Unmarshal(w.Body.Bytes(), &personas)
	if len(personas) != 1 || personas[0] != "p1" {
		t.Errorf("Expected [p1], got %v", personas)
	}
}

func TestSetAndGetAppStore(t *testing.T) {
	r, _ := setupTestRouter()

	// Set value
	val := map[string]any{"name": "test"}
	body, _ := json.Marshal(val)
	req, _ := http.NewRequest("POST", "/personas/p1/apps/a1/keys/k1", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Get App Store
	req, _ = http.NewRequest("GET", "/personas/p1/apps/a1", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var data map[string]any
	json.Unmarshal(w.Body.Bytes(), &data)
	if data["k1"].(map[string]any)["name"] != "test" {
		t.Errorf("Expected test, got %v", data["k1"])
	}
}

func TestMove(t *testing.T) {
	r, h := setupTestRouter()
	h.Store.Set("p1", "a1", "k1", "v1")

	moveReq := struct {
		SrcPersona string `json:"src_persona"`
		DstPersona string `json:"dst_persona"`
		AppID      string `json:"app_id"`
		Key        string `json:"key"`
	}{
		SrcPersona: "p1",
		DstPersona: "p2",
		AppID:      "a1",
		Key:        "k1",
	}

	body, _ := json.Marshal(moveReq)
	req, _ := http.NewRequest("POST", "/move", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Verify move
	val, err := h.Store.Get("p2", "a1", "k1")
	if err != nil || val != "v1" {
		t.Errorf("Move failed, val: %v, err: %v", val, err)
	}
	_, err = h.Store.Get("p1", "a1", "k1")
	if err == nil {
		t.Error("Key should have been deleted from source persona")
	}
}

func TestGetGlobalAPI(t *testing.T) {
	r, h := setupTestRouter()
	r.GET("/global/:app/:key", h.GetGlobal)
	h.Store.Set("p1", "a1", "k1", "v1")

	req, _ := http.NewRequest("GET", "/global/a1/k1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var res map[string]any
	json.Unmarshal(w.Body.Bytes(), &res)
	if res["persona"] != "p1" || res["value"] != "v1" {
		t.Errorf("Unexpected response: %v", res)
	}
}

func TestDeleteAPI(t *testing.T) {
	r, h := setupTestRouter()
	h.Store.Set("p1", "a1", "k1", "v1")

	req, _ := http.NewRequest("DELETE", "/personas/p1/apps/a1/keys/k1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	_, err := h.Store.Get("p1", "a1", "k1")
	if err == nil {
		t.Error("Key should have been deleted")
	}
}

func TestInvalidJSONSet(t *testing.T) {
	r, _ := setupTestRouter()

	req, _ := http.NewRequest("POST", "/personas/p1/apps/a1/keys/k1", bytes.NewBufferString("invalid"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}
}
