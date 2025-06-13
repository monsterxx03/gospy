package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/monsterxx03/gospy/pkg/proc"
)

type Server struct {
	port     int
	readers  map[int]proc.ProcessMemReader // pid -> reader cache
	mu       sync.RWMutex
	showDead bool

	enableMCP bool
	mcpServer *server.StreamableHTTPServer
}

func NewServer(port int, showDead bool, enableMCP bool) *Server {
	s := &Server{
		port:      port,
		readers:   make(map[int]proc.ProcessMemReader),
		showDead:  showDead,
		enableMCP: enableMCP,
	}
	if enableMCP {
		s.mcpServer = s.getMCPSServer()
	}
	return s
}

func (s *Server) getMCPSServer() *server.StreamableHTTPServer {
	ms := server.NewMCPServer(
		"gospy mcp server",
		"1.0.0",
	)
	goroutineTool := mcp.NewTool("goroutines",
		mcp.WithDescription("dump golang process's goroutines"),
		mcp.WithNumber("pid", mcp.Required(), mcp.Description("process pid")))
	ms.AddTool(goroutineTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pid := int(request.GetArguments()["pid"].(float64))
		reader, err := s.getReader(pid)
		if err != nil {
			return nil, err
		}
		goroutines, err := reader.Goroutines(s.showDead)
		if err != nil {
			return nil, fmt.Errorf("failed to get goroutines: %w", err)
		}
		data, err := json.Marshal(goroutines)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(data)), nil
	})

	memstatsTool := mcp.NewTool("gomemstats",
		mcp.WithDescription("dump golang process's memory statistics"),
		mcp.WithNumber("pid", mcp.Required(), mcp.Description("process pid")))
	ms.AddTool(memstatsTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pid := int(request.GetArguments()["pid"].(float64))
		reader, err := s.getReader(pid)
		if err != nil {
			return nil, err
		}
		memStats, err := reader.MemStat()
		if err != nil {
			return nil, fmt.Errorf("failed to get memory stats: %w", err)
		}
		data, err := json.Marshal(memStats)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(data)), nil
	})

	runtimeTool := mcp.NewTool("goruntime",
		mcp.WithDescription("get golang process's runtime info"),
		mcp.WithNumber("pid", mcp.Required(), mcp.Description("process pid")))
	ms.AddTool(runtimeTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pid := int(request.GetArguments()["pid"].(float64))
		reader, err := s.getReader(pid)
		if err != nil {
			return nil, err
		}
		runtimeInfo, err := reader.RuntimeInfo()
		if err != nil {
			return nil, fmt.Errorf("failed to get runtime info: %w", err)
		}
		data, err := json.Marshal(runtimeInfo)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(data)), nil
	})

	pgrepTool := mcp.NewTool("pgrep",
		mcp.WithDescription("find process IDs by process name"),
		mcp.WithString("name", mcp.Required(), mcp.Description("process name to search for")))
	ms.AddTool(pgrepTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		name := request.GetArguments()["name"].(string)
		cmd := exec.Command("pgrep", name)
		output, err := cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("pgrep failed: %w", err)
		}
		pids := strings.Split(strings.TrimSpace(string(output)), "\n")
		data, err := json.Marshal(pids)
		if err != nil {
			return nil, err
		}
		return mcp.NewToolResultText(string(data)), nil
	})

	return server.NewStreamableHTTPServer(ms)
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
	if s.enableMCP {
		http.Handle("/mcp", s.mcpServer)
	}
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
	goroutines, err := reader.Goroutines(s.showDead)
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
