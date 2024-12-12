package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// Constants
const (
	defaultPort = "3000"
	apiPrefix   = "/api"
)

type SystemStats struct {
	CPUUsage   float64       `json:"cpuUsage"`
	MemUsage   float64       `json:"memUsage"`
	DiskUsage  float64       `json:"diskUsage"`
	NetTraffic int64         `json:"netTraffic"`
	Processes  []ProcessInfo `json:"processes"`
}

type ProcessInfo struct {
	PID         int32   `json:"pid"`
	Name        string  `json:"name"`
	CPUPercent  float64 `json:"cpuPercent"`
	MemoryUsage float32 `json:"memoryUsage"` // in MB
}

// Server represents our HTTP server
type Server struct {
	router *http.ServeMux
	port   string
}

// NewServer creates a new server instance
func NewServer(port string) *Server {
	if port == "" {
		port = defaultPort
	}
	return &Server{
		router: http.NewServeMux(),
		port:   port,
	}
}

// setupRoutes configures all the routes for the server
func (s *Server) setupRoutes() {
	// Serve frontend build files
	fs := http.FileServer(http.Dir("../frontend/dist"))
	s.router.Handle("/", fs)

	// API endpoints
	s.router.HandleFunc(apiPrefix+"/stats", s.statsHandler)
	s.router.HandleFunc(apiPrefix+"/events", s.sseHandler)
}

// Start starts the server and handles graceful shutdown
func (s *Server) Start() error {
	server := &http.Server{
		Addr:         ":" + s.port,
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel for shutdown signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Channel for server errors
	errChan := make(chan error, 1)

	go func() {
		log.Printf("Server running at http://localhost:%s\n", s.port)
		errChan <- server.ListenAndServe()
	}()

	// Wait for shutdown signal or error
	select {
	case <-stop:
		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(ctx)
	case err := <-errChan:
		return fmt.Errorf("server error: %w", err)
	}
}

// Fetch system and process stats
func getStats() (*SystemStats, error) {
	// Get CPU stats
	cpuPercentages, err := cpu.Percent(0, false)
	if err != nil {
		return nil, fmt.Errorf("error getting CPU stats: %w", err)
	}
	if len(cpuPercentages) == 0 {
		return nil, fmt.Errorf("no CPU statistics available")
	}

	// Get memory stats
	memStats, err := mem.VirtualMemory()
	if err != nil {
		return nil, fmt.Errorf("error getting memory stats: %w", err)
	}

	// Get disk stats
	diskStats, err := disk.Usage("/")
	if err != nil {
		return nil, fmt.Errorf("error getting disk stats: %w", err)
	}

	// Get network stats
	netStats, err := net.IOCounters(false)
	if err != nil {
		return nil, fmt.Errorf("error getting network stats: %w", err)
	}
	if len(netStats) == 0 {
		return nil, fmt.Errorf("no network statistics available")
	}

	// Get process stats
	procs, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("error getting process list: %w", err)
	}

	processInfo := []ProcessInfo{}
	for _, proc := range procs {
		name, err := proc.Name()
		if err != nil {
			continue // Skip this process if we can't get its name
		}

		cpuPercent, err := proc.CPUPercent()
		if err != nil {
			continue // Skip this process if we can't get CPU usage
		}

		memInfo, err := proc.MemoryInfo()
		if err != nil {
			continue // Skip this process if we can't get memory info
		}

		processInfo = append(processInfo, ProcessInfo{
			PID:         proc.Pid,
			Name:        name,
			CPUPercent:  cpuPercent,
				MemoryUsage: float32(memInfo.RSS) / (1024 * 1024),
		})
	}

	stats := &SystemStats{
		CPUUsage:   cpuPercentages[0],
			MemUsage:   memStats.UsedPercent,
			DiskUsage:  diskStats.UsedPercent,
			NetTraffic: int64(netStats[0].BytesRecv + netStats[0].BytesSent),
			Processes:  processInfo,
	}
	return stats, nil
}

// Convert the handlers to methods on Server
func (s *Server) statsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := getStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func (s *Server) sseHandler(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Create encoder for JSON
	encoder := json.NewEncoder(w)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			stats, err := getStats()
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: %v\n\n", err)
				w.(http.Flusher).Flush()
				continue
			}

			fmt.Fprintf(w, "event: stats\ndata: ")
			encoder.Encode(stats)
			fmt.Fprintf(w, "\n\n")
			w.(http.Flusher).Flush()
		}
	}
}

func main() {
	// Create and start server
	server := NewServer(os.Getenv("PORT"))
	server.setupRoutes()
	
	if err := server.Start(); err != nil {
		log.Fatal(err)
	}
}
	