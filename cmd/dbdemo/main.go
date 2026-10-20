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
	//sql.DB 实际上是一个管理数据库连接的句柄，内部负责连接池，而不是单条物理连接。
	//而且 sql.Open() 不保证已经建立实际连接，所以我们再调用 Ping() 验证。

	fmt.Println("database connected")

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS nodes(
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	name TEXT NOT NULL,
	address TEXT NOT NULL,
	region TEXT NOT NULL
	)`)

	if err != nil {
		log.Fatal(err)
	}

	query := `
		INSERT INTO nodes (name, address, region)
		VALUES (?, ?, ?)
	`

	_, err = db.Exec(
		query,
		"JP-01",
		"jp.example.net:443",
		"JP",
	)
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
		var temp node
		err = rows.Scan(&temp.ID, &temp.Name, &temp.Address, &temp.Region)
		if err != nil {
			log.Fatal(err)
		}
		nodes = append(nodes, temp)
	}
	//in case connection halt,row.Next() wont throw error

	if err = rows.Err(); err != nil {
		log.Fatal(err)
	}
	for _, node := range nodes {
		fmt.Printf(
			"ID=%d Name=%s Address=%s Region=%s\n",
			node.ID,
			node.Address,
			node.Name,
			node.Region,
		)
	}
}
