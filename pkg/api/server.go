package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/monsterxx03/gospy/pkg/proc"
)

type Server struct {
	port    int
	readers map[int]proc.ProcessMemReader // pid -> reader cache
	mu      sync.RWMutex
}

func NewServer(port int) *Server {
	return &Server{
		port:    port,
		readers: make(map[int]proc.ProcessMemReader),
	}
}

func (s *Server) getReader(pid int) (proc.ProcessMemReader, error) {
	s.mu.RLock()
	if reader, ok := s.readers[pid]; ok {
		s.mu.RUnlock()
		return reader, nil
	}
	s.mu.RUnlock()

	reader, err := proc.NewProcessMemReader(pid, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create reader: %w", err)
	}

	s.mu.Lock()
	s.readers[pid] = reader
	s.mu.Unlock()

	return reader, nil
}

func (s *Server) closeReader(pid int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if reader, ok := s.readers[pid]; ok {
		reader.Close()
		delete(s.readers, pid)
	}
}

func (s *Server) Start() error {
	http.HandleFunc("/runtime", s.handleRuntime)
	http.HandleFunc("/goroutines", s.handleGoroutines)
	http.HandleFunc("/memstats", s.handleMemStats)
	return http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)
}

func (s *Server) handleRuntime(w http.ResponseWriter, r *http.Request) {
	pid, err := getPID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reader, err := s.getReader(pid)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get reader: %v", err), http.StatusInternalServerError)
		return
	}

	rt, err := reader.RuntimeInfo()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get runtime info: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, rt)
}

func (s *Server) handleGoroutines(w http.ResponseWriter, r *http.Request) {
	pid, err := getPID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reader, err := s.getReader(pid)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create reader: %v", err), http.StatusInternalServerError)
		return
	}
	goroutines, err := reader.Goroutines()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get goroutines: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, goroutines)
}

func (s *Server) handleMemStats(w http.ResponseWriter, r *http.Request) {
	pid, err := getPID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	reader, err := s.getReader(pid)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create reader: %v", err), http.StatusInternalServerError)
		return
	}

	memStats, err := reader.MemStat()
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get memory stats: %v", err), http.StatusInternalServerError)
		return
	}

	writeJSON(w, memStats)
}

func getPID(r *http.Request) (int, error) {
	pidStr := r.URL.Query().Get("pid")
	if pidStr == "" {
		return 0, fmt.Errorf("pid parameter is required")
	}
	return strconv.Atoi(pidStr)
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, fmt.Sprintf("failed to encode response: %v", err), http.StatusInternalServerError)
	}
}
