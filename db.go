// db.go
package main

import (
	"database/sql"
	"fmt"
)

var (
	db			*sql.DB
)
	
// Open database.
func OpenDatabase() {
	print("OpenDatabase")
	
	// Connect to database
	dbInfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable",
		flags.dbUser, flags.dbPassword, flags.dbName)  

	fmt.Printf("dbInfo: %s", dbInfo)

	db, err = sql.Open("postgres", dbInfo)
	check(err)
	
	printVal("db", db)
}

// Close database.
func CloseDatabase() {
	print("CloseDatabase")
	
	if db != nil {
		defer db.Close()
	}
}

// Inserts a new record into the database and returns the Id of the inserted record.
// Panics on error.
func DbInsert(query string, values ...interface{}) int {
	var lastInsertId int
	check(db.QueryRow(
		query,
		values...
	).Scan(&lastInsertId))	
	return lastInsertId
}

// Executes a database query, returns the sql.Rows.
// Panics on error.
func DbQuery(query string, values ...interface{}) *sql.Rows {
	rows, err := db.Query(query, values...)
	check(err)	
	return rows
}

// Executes a query, and TRUE if it returned any row.
// Panics on error
func DbExists(query string, values ...interface{}) bool {
	rows := DbQuery(query, values...)
	return rows.Next()
}
