// Package httpserver exposes the MyTurn JSON API and serves the embedded
// static frontend.
package httpserver

import (
	"crypto/subtle"
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"sync"

	"myturn/internal/config"
	"myturn/internal/rotation"
)

const upcomingCount = 7

// Server holds shared, mutex-protected state for the HTTP handlers.
type Server struct {
	mu          sync.RWMutex
	cfg         *config.Config
	configPath  string
	staticFS    fs.FS
	adminSecret string
}

// New constructs a Server. staticFS should contain index.html, settings.html,
// and the other static frontend assets at its root. adminSecret gates access
// to the settings/config API via the X-Admin-Secret header.
func New(cfg *config.Config, configPath string, staticFS fs.FS, adminSecret string) *Server {
	return &Server{cfg: cfg, configPath: configPath, staticFS: staticFS, adminSecret: adminSecret}
}

// Handler builds the HTTP route mux for the server.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/state", s.handleGetState)
	mux.HandleFunc("GET /api/config", s.requireAdminSecret(s.handleGetConfig))
	mux.HandleFunc("PUT /api/config", s.requireAdminSecret(s.handlePutConfig))
	mux.HandleFunc("POST /api/skip/today", s.requireAdminSecret(s.handleSkipToday))
	mux.Handle("/", http.FileServerFS(s.staticFS))
	return mux
}

// requireAdminSecret rejects requests whose X-Admin-Secret header does not
// match the configured admin secret.
func (s *Server) requireAdminSecret(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		given := r.Header.Get("X-Admin-Secret")
		if subtle.ConstantTimeCompare([]byte(given), []byte(s.adminSecret)) != 1 {
			writeError(w, http.StatusUnauthorized, "invalid admin secret")
			return
		}
		next(w, r)
	}
}

type stateResponse struct {
	GroupName    string           `json:"group_name"`
	Date         string           `json:"date"`
	Today        string           `json:"today"`
	TodaySkipped bool             `json:"today_skipped"`
	Tomorrow     string           `json:"tomorrow"`
	Members      []string         `json:"members"`
	Upcoming     []rotation.Entry `json:"upcoming"`
}

func (s *Server) handleGetState(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	cfg := s.cfg
	s.mu.RUnlock()

	today, err := rotation.TodayInTZ(cfg.Timezone)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	upcoming, err := rotation.Upcoming(cfg, today, upcomingCount)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	todayMember := upcoming[0].Member
	tomorrowMember := ""
	if len(upcoming) > 1 {
		tomorrowMember = upcoming[1].Member
	}

	writeJSON(w, http.StatusOK, stateResponse{
		GroupName:    cfg.GroupName,
		Date:         today,
		Today:        todayMember,
		TodaySkipped: todayMember == "",
		Tomorrow:     tomorrowMember,
		Members:      cfg.Members,
		Upcoming:     upcoming,
	})
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	writeJSON(w, http.StatusOK, s.cfg)
}

func (s *Server) handlePutConfig(w http.ResponseWriter, r *http.Request) {
	var request struct {
		config.Config
		SkipToday *bool `json:"skip_today"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	next := request.Config
	if request.SkipToday != nil {
		today, err := rotation.TodayInTZ(next.Timezone)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		next.SkipDates = setSkipDate(next.SkipDates, today, *request.SkipToday)
	}
	if err := config.Validate(&next); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if err := config.Save(s.configPath, &next); err != nil {
		writeError(w, http.StatusInternalServerError, "saving config: "+err.Error())
		return
	}
	s.cfg = &next
	writeJSON(w, http.StatusOK, s.cfg)
}

func setSkipDate(skipDates []string, date string, skipped bool) []string {
	result := make([]string, 0, len(skipDates)+1)
	found := false
	for _, existing := range skipDates {
		if existing == date {
			found = true
			if !skipped {
				continue
			}
		}
		result = append(result, existing)
	}
	if skipped && !found {
		result = append(result, date)
	}
	return result
}

type skipTodayResponse struct {
	Date    string         `json:"date"`
	Skipped bool           `json:"skipped"`
	Config  *config.Config `json:"config"`
}

func (s *Server) handleSkipToday(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	today, err := rotation.TodayInTZ(s.cfg.Timezone)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, date := range s.cfg.SkipDates {
		if date == today {
			writeJSON(w, http.StatusOK, skipTodayResponse{Date: today, Skipped: true, Config: s.cfg})
			return
		}
	}

	next := cloneConfig(s.cfg)
	next.SkipDates = append(next.SkipDates, today)
	if err := config.Save(s.configPath, &next); err != nil {
		writeError(w, http.StatusInternalServerError, "saving config: "+err.Error())
		return
	}
	s.cfg = &next
	writeJSON(w, http.StatusOK, skipTodayResponse{Date: today, Skipped: true, Config: s.cfg})
}

func cloneConfig(cfg *config.Config) config.Config {
	members := append([]string(nil), cfg.Members...)
	skipDates := append([]string(nil), cfg.SkipDates...)
	return config.Config{
		GroupName: cfg.GroupName,
		Timezone:  cfg.Timezone,
		StartDate: cfg.StartDate,
		Members:   members,
		SkipDates: skipDates,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("error encoding JSON response: %v", err)
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, errorResponse{Error: message})
}
