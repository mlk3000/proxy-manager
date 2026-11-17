package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

type node struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Address string `json:"address"`
	Region  string `json:"region"`
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
	fmt.Println("database connected")

	cmd := `CREATE TABLE IF NOT EXISTS nodes(
		ID INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		address TEXT NOT NULL,
		region TEXT NOT NULL
	)`

	_, err = db.Exec(cmd)
	if err != nil {
		log.Fatal(err)
	}
	//don't simply use fmt.Sprintf to form a cmd as it may cause sql injection
	insertCmd := "INSERT INTO nodes(name,address,region) VALUES(?,?,?)"
	_, err = db.Exec(insertCmd, "KR-01", "kr.example.com:443", "KR")
	if err != nil {
		log.Fatal(err)
	}

	rows, err := db.Query("SELECT id,name,address,region FROM nodes ORDER BY id")

	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	nodes := []node{}

	for rows.Next() {
		node := node{}
		err = rows.Scan(&node.ID, &node.Name, &node.Address, &node.Region)
		//err here about conversion
		if err != nil {
			log.Fatal(err)
		}
		nodes = append(nodes, node)
	}
	//err here about fetching next line
	if rows.Err() != nil {
		log.Fatal(rows)
	}

	for _, node := range nodes {
		fmt.Println(node)
	}
}
