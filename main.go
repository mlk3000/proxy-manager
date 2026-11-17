package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Node struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Region  string `json:"region"`
}

type Server struct {
	db *sql.DB
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

	cmd := `INSERT INTO nodes (name,address,region) VALUES (?,?,?)`

	result, err := s.db.ExecContext(r.Context(), cmd, input.Name, input.Address, input.Region)
	if err != nil {
		log.Println("insert db error", err)
		http.Error(w, "database error", http.StatusInternalServerError)
	}

	id, err := result.LastInsertId()
	if err != nil {
		log.Println("insert db error", err)
		http.Error(w, "database error", http.StatusInternalServerError)
	}
	input.ID = int(id)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(input)
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
	//	snapshot := append([]Node{}, s.nodes...)

	rows, err := s.db.QueryContext(r.Context(), "SELECT id,name,address,region FROM nodes ORDER BY id")

	if err != nil {
		log.Println("query error:", err)
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	nodes := []Node{}
	// 初始化为空但非 nil 的 slice，JSON 编码成 []；
	// var nodes []node 初始化，如果没有 append 任何元素，会编码成 null
	// 请求失败时返回[] 更符合API设计
	for rows.Next() {
		temp := Node{}
		err = rows.Scan(&temp.ID, &temp.Name, &temp.Address, &temp.Region)
		if err != nil {
			log.Println("scan error:", err)
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		nodes = append(nodes, temp)
	}

	if err = rows.Err(); err != nil {
		log.Println("rows error:", err)
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	//w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(nodes)
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

	var result Node

	cmd := "SELECT * FROM nodes WHERE ID= ?"

	err = s.db.QueryRowContext(r.Context(), cmd, id).Scan(&result.ID, &result.Name, &result.Address, &result.Region)

	//two errs here:
	if err == sql.ErrNoRows {
		http.Error(w, "node not found", http.StatusNotFound)
		return
	}

	if err != nil {
		log.Println("query error:", err)
		http.Error(w, "err", http.StatusInternalServerError)
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
	//Previous two methods
	//s.nodes = slices.Delete(s.nodes, index, index+1)
	//s.nodes = append(s.nodes[:index], s.nodes[index+1:]...)

	cmd := "DELETE FROM nodes WHERE id = ?"

	res, err := s.db.ExecContext(r.Context(), cmd, id)
	//if line was not found, cmd will still work while deleting 0 lines

	if err != nil {
		log.Println("delete error", err)
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	count, err := res.RowsAffected()
	if err != nil {
		log.Println("rows affected error")
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	if count == 0 {
		log.Println("No res")
		http.Error(w, "no item", http.StatusNotFound)
	}

	w.WriteHeader(http.StatusNoContent)
}

func main() {

	db, err := sql.Open("sqlite", "nodes.db")
	if err != nil {
		log.Fatal(err)
	}

	defer db.Close()
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS nodes(
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	address TEXT NOT NULL,
	region TEXT NOT NULL
	)`)

	if err != nil {
		log.Fatal(err)
	}

	server := &Server{db: db}

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
