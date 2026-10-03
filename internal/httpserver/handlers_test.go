package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"testing/fstest"

	"myturn/internal/config"
	"myturn/internal/rotation"
)

func TestSkipTodayRequiresSecretAndPersistsOnce(t *testing.T) {
	today, err := rotation.TodayInTZ("UTC")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		GroupName: "Test",
		Timezone:  "UTC",
		StartDate: today,
		Members:   []string{"Ana", "Bruno"},
	}
	server := New(cfg, t.TempDir()+"/config.json", fstest.MapFS{}, "secret")
	handler := server.Handler()

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodPost, "/api/skip/today", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want %d", unauthorized.Code, http.StatusUnauthorized)
	}

	for i := 0; i < 2; i++ {
		request := httptest.NewRequest(http.MethodPost, "/api/skip/today", nil)
		request.Header.Set("X-Admin-Secret", "secret")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("skip request status = %d, want %d: %s", response.Code, http.StatusOK, response.Body)
		}
	}

	saved, err := config.Load(server.configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.SkipDates) != 1 || saved.SkipDates[0] != today {
		t.Fatalf("saved skip dates = %v, want [%s]", saved.SkipDates, today)
	}

	stateRecorder := httptest.NewRecorder()
	handler.ServeHTTP(stateRecorder, httptest.NewRequest(http.MethodGet, "/api/state", nil))
	if stateRecorder.Code != http.StatusOK {
		t.Fatalf("state status = %d, want %d", stateRecorder.Code, http.StatusOK)
	}
	var state stateResponse
	if err := json.Unmarshal(stateRecorder.Body.Bytes(), &state); err != nil {
		t.Fatal(err)
	}
	if !state.TodaySkipped || state.Today != "" || state.Tomorrow != "Ana" {
		t.Fatalf("state after skip = today_skipped:%v today:%q tomorrow:%q", state.TodaySkipped, state.Today, state.Tomorrow)
	}
}

func TestPutConfigAppliesSkipTodayOnlyWhenSaved(t *testing.T) {
	today, err := rotation.TodayInTZ("UTC")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		GroupName: "Test",
		Timezone:  "UTC",
		StartDate: today,
		Members:   []string{"Ana", "Bruno"},
	}
	server := New(cfg, t.TempDir()+"/config.json", fstest.MapFS{}, "secret")
	handler := server.Handler()

	request := httptest.NewRequest(http.MethodPut, "/api/config", strings.NewReader(`{"group_name":"Test","timezone":"UTC","start_date":"`+today+`","members":["Ana","Bruno"],"skip_today":true}`))
	request.Header.Set("X-Admin-Secret", "secret")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("save status = %d, want %d: %s", response.Code, http.StatusOK, response.Body)
	}

	saved, err := config.Load(server.configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.SkipDates) != 1 || saved.SkipDates[0] != today {
		t.Fatalf("saved skip dates = %v, want [%s]", saved.SkipDates, today)
	}

	request = httptest.NewRequest(http.MethodPut, "/api/config", strings.NewReader(`{"group_name":"Test","timezone":"UTC","start_date":"`+today+`","members":["Ana","Bruno"],"skip_dates":["`+today+`"],"skip_today":false}`))
	request.Header.Set("X-Admin-Secret", "secret")
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("save to unskip status = %d, want %d: %s", response.Code, http.StatusOK, response.Body)
	}
	saved, err = config.Load(server.configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.SkipDates) != 0 {
		t.Fatalf("saved skip dates after unchecking switch = %v, want none", saved.SkipDates)
	}
}
