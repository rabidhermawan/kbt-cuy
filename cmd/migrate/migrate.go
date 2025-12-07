package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

func main() {
	// Load .env file from the parent directory
	err := godotenv.Load("../.env")
	if err != nil {
		log.Println("Warning: Can't find .env file, using environment variables from system")
	}

	// 1. Connect to local SQLite
	localDB, err := sql.Open("sqlite3", "powerbank.db")
	if err != nil {
		log.Fatal("Failed to open local DB:", err)
	}
	defer localDB.Close()

	// 2. Dump schema and data
	tables := []string{"users", "powerbank_stations", "powerbanks", "transactions"}
	var sqlDump strings.Builder

	for _, table := range tables {
		// Get CREATE TABLE statement
		rows, err := localDB.Query(fmt.Sprintf("SELECT sql FROM sqlite_master WHERE type='table' AND name='%s'", table))
		if err != nil {
			log.Printf("Error getting schema for %s: %v", table, err)
			continue
		}
		for rows.Next() {
			var createSQL sql.NullString
			rows.Scan(&createSQL)
			if createSQL.Valid {
				sqlDump.WriteString(createSQL.String + ";\n")
			}
		}
		rows.Close()

		// Get INSERT statements
		rows, err = localDB.Query(fmt.Sprintf("SELECT * FROM %s", table))
		if err != nil {
			log.Printf("Error querying %s: %v", table, err)
			continue
		}
		columns, err := rows.Columns()
		if err != nil {
			log.Printf("Error getting columns for %s: %v", table, err)
			rows.Close()
			continue
		}
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		for rows.Next() {
			rows.Scan(valuePtrs...)
			var insertValues []string
			for _, val := range values {
				if val == nil {
					insertValues = append(insertValues, "NULL")
				} else {
					insertValues = append(insertValues, fmt.Sprintf("'%v'", strings.ReplaceAll(fmt.Sprintf("%v", val), "'", "''")))
				}
			}
			insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);\n",
				table, strings.Join(columns, ","), strings.Join(insertValues, ","))
			sqlDump.WriteString(insertSQL)
		}
		rows.Close()
	}

	// 3. Connect to Turso
	tursoURL := os.Getenv("TURSO_URL")
	tursoToken := os.Getenv("TURSO_AUTH_TOKEN")
	tursoDB, err := sql.Open("libsql", tursoURL+"?authToken="+tursoToken)
	if err != nil {
		log.Fatal("Failed to connect to Turso:", err)
	}
	defer tursoDB.Close()

	// 4. Execute the SQL dump on Turso
	sqlStatements := strings.Split(sqlDump.String(), ";")
	for _, stmt := range sqlStatements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		_, err := tursoDB.Exec(stmt)
		if err != nil {
			log.Printf("Error executing: %s\nError: %v", stmt, err)
		}
	}

	fmt.Println("Migration completed!")
}
