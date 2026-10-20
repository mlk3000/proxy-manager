package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Node struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Region  string `json:"region"`
}

type Server struct {
	mu     sync.Mutex
	nodes  []Node
	nextID int
}

func (s *Server) createNode(w http.ResponseWriter, r *http.Request) {
	var input Node
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Address) == "" {
		http.Error(w, "name and address required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	input.ID = s.nextID
	s.nextID++

	s.nodes = append(s.nodes, input)
	s.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(input)
	if err != nil {
		log.Println("encode error:", err)
	}
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "ok")
}

func debugWait(w http.ResponseWriter, r *http.Request) {
	ms, err := strconv.Atoi(r.URL.Query().Get("ms"))
	if err != nil || ms < 0 || ms > 5000 {
		http.Error(w, "invalid ms", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)

	defer cancel()

	timer := time.NewTimer(time.Duration(ms) * time.Millisecond)
	defer timer.Stop()

	select {
	case <-timer.C:
		fmt.Fprintf(w, "work finished")
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			//to distinct timeout error from client subjectly closing connection
			//if so then no extra action is needed
			http.Error(w, "work timed out", http.StatusServiceUnavailable)
			return
		}
		log.Println("request canceled:", ctx.Err())
	}
}

func (s *Server) listNodes(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	snapshot := append([]Node{}, s.nodes...)
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	//w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(snapshot)
	if err != nil {
		log.Println("encode error", err)
	}
}

func (s *Server) getNode(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		http.Error(w, "invalid node ID!", http.StatusBadRequest)
		return
	}
	s.mu.Lock()

	var result Node
	found := false

	for _, node := range s.nodes {
		if node.ID == id {
			result = node
			found = true
			break
		}
	}
	s.mu.Unlock()
	if !found {
		http.Error(w, "id not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(result)
	if err != nil {
		log.Println("encode error:", err)
	}
}

func (s *Server) deleteNode(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		http.Error(w, "invalid node ID!", http.StatusBadRequest)
		return
	}
	s.mu.Lock()

	found := false

	for index, node := range s.nodes {
		if node.ID == id {
			s.nodes = slices.Delete(s.nodes, index, index+1)
			//s.nodes = append(s.nodes[:index], s.nodes[index+1:]...)
			found = true
			break
		}
	}
	s.mu.Unlock()
	if !found {
		http.Error(w, "id not found", http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func main() {

	server := &Server{nodes: []Node{
		{ID: 1, Name: "KR-01", Address: "kr.example.net:443", Region: "KR"},
		{ID: 2, Name: "US-01", Address: "us.example.net:443", Region: "US"},
	}, nextID: 3}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("GET /nodes", server.listNodes)
	mux.HandleFunc("POST /nodes", server.createNode)
	mux.HandleFunc("GET /nodes/{id}", server.getNode)
	mux.HandleFunc("DELETE /nodes/{id}", server.deleteNode)
	mux.HandleFunc("GET /debug/wait", debugWait)
	//no need to add {} since ?arg can be parsed straightly
	customServer := &http.Server{
		Addr:              "127.0.0.1:8080",
		Handler:           loggingMiddleware(mux),
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}
	log.Fatal(customServer.ListenAndServe())
	//http.ListenAndServe("127.0.0.1:8080", loggingMiddleware(mux))
}
