package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
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

// HTTP handler for GET /api/stats
func statsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := getStats()
	if err != nil {
		http.Error(w, "Error fetching stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// SSE handler for /events
func sseHandler(w http.ResponseWriter, r *http.Request) {
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
	router := http.NewServeMux()
	
	// Serve frontend build files
	fs := http.FileServer(http.Dir("../frontend/dist"))
	router.Handle("/", fs)
	
	// API endpoints
	router.HandleFunc("/api/stats", statsHandler)
	router.HandleFunc("/api/events", sseHandler)

	fmt.Println("Server running at http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", router))
}
	