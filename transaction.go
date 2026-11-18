package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

type RenamePairRequest struct {
	FirstId    int    `json:"first_id"`
	FirstName  string `json:"first_name"`
	SecondtId  int    `json:"second_id"`
	SecondName string `json:"second_name"`
}

func (s *Server) renamePair(w http.ResponseWriter, r *http.Request) {
	temp := RenamePairRequest{}

	err := json.NewDecoder(r.Body).Decode(&temp)
	if err != nil {
		log.Println("invalid JSON", err)
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if temp.FirstId < 0 ||
		temp.SecondtId < 0 ||
		temp.FirstId == temp.SecondtId ||
		strings.TrimSpace(temp.FirstName) == "" ||
		strings.TrimSpace(temp.SecondName) == "" {
		log.Println("invalid rename request", err)
		http.Error(w, "invalid rename request", http.StatusBadRequest)
		return
	}

	cmd := "UPDATE nodes SET name = ? WHERE id = ?"
	tx, err := s.db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Println("begin tx error", err)
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	//instantly call defer rollback as an insurance whenever return later
	defer tx.Rollback()

	res, err := tx.ExecContext(r.Context(), cmd, temp.FirstName, temp.FirstId)

	if err != nil {
		log.Println("first update error", err)
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	affected, err := res.RowsAffected()

	if err != nil {
		log.Println("rows affected error", err)
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	if affected == 0 {
		log.Println("first node not found", err)
		http.Error(w, "first node not found", http.StatusNotFound)
		return
	}

	res, err = tx.ExecContext(r.Context(), cmd, temp.SecondName, temp.SecondtId)

	if err != nil {
		log.Println("second update error", err)
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	affected, err = res.RowsAffected()

	if err != nil {
		log.Println("rows affected error", err)
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	if affected == 0 {
		log.Println("second node not found", err)
		http.Error(w, "second node not found", http.StatusNotFound)
		return
	}

	err = tx.Commit()
	if err != nil {
		log.Println("commit error", err)
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)

}
