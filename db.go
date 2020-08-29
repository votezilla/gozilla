// db.go
package main

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
    "github.com/jmoiron/sqlx"
)

var (
	//db			*sql.DB
	db			*sqlx.DB
)

// Open database.
func OpenDatabase() {
	pr("OpenDatabase")

	// Connect to database
	dbInfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable",
		flags.dbUser, flags.dbPassword, flags.dbName)

	prf("dbInfo: %s", dbInfo)

	db, err = sqlx.Connect("postgres", dbInfo)//sql.Open("postgres", dbInfo)
	check(err)

	// Suggested defaults:
	db.SetMaxOpenConns(25) // Sane default
	db.SetMaxIdleConns(25)
    	db.SetConnMaxLifetime(1 * time.Minute)

	prVal("db", db)
}

// Close database.
func CloseDatabase() {
	pr("CloseDatabase")

	open := db.Stats().OpenConnections
	if open > 0 {
		// This could also modify the return code...
		prf("failed to close %d connections!", open)
    }

	if db != nil {
		db.Close()
	}
}

func DbTrackOpenConnections() {
	// This could also modify the return code...
	prVal("Open connections", db.Stats().OpenConnections)
}

// Replace all instances of "$$" with "votezilla." or whatever the schema is, in the query.
func replaceSchema(query string) string {
	return strings.Replace(query, "$$", "vz.", -1)
}

// Executes a query that does not return anything.  Necessary for not leaking connections.
func DbExec(query string, values ...interface{}) {
	query = replaceSchema(query)
	prf("DbExec query:%s %v", query, values)

	_, err = db.Exec(query, values...)
	check(err)
}

// Inserts a new record into the database and returns the Id of the inserted record.
// Panics on error.
func DbInsert(query string, values ...interface{}) int64 {
	query = replaceSchema(query)
	prf("DbInsert query:%s %v", query, values)

	var lastInsertId int64

	check(db.QueryRow(
		query,
		values...
	).Scan(&lastInsertId))
	return lastInsertId
}

// Executes a database query, returns the sql.Rows.
// Panics on error.
func DbQuery(query string, values ...interface{}) *sql.Rows {
	query = replaceSchema(query)
	prf("DbQuery query:%s %v", query, values)

	rows, err := db.Query(query, values...)
	check(err)
	return rows
}

// Executes a database query which returns a single count,
// (usually by invoking COUNT(*)), and returns the int count.
func DbQueryCount(query string, values ...interface{}) int {
	var count int

	rows := DbQuery(query, values...)
	if rows.Next() {
		err := rows.Scan(&count)
		check(err)
	}
	check(rows.Err())
	rows.Close()

	return count
}

// Executes a query, and TRUE if it returned any row.
// Panics on error
func DbExists(query string, values ...interface{}) bool {
	query = replaceSchema(query)
	prf("DbExists query:%s %v", query, values)

	rows := DbQuery(query, values...)

	return rows.Next()
}


// For future reference, here's how to do a map scan, for handling columns abstractly:
//
// rows, err := db.Queryx("SELECT * FROM place")
// for rows.Next() {
//     results := make(map[string]interface{})
//     err = rows.MapScan(results)
// }
//
// Source: https://jmoiron.github.io/sqlx/
