// db.go
package main

import (
	"database/sql"
	"fmt"
	"time"
)

var (
	db			*sql.DB
)
	
// Open database.
func OpenDatabase() {
	pr(db_, "OpenDatabase")
	
	// Connect to database
	dbInfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable",
		flags.dbUser, flags.dbPassword, flags.dbName)  

	prf(db_, "dbInfo: %s", dbInfo)

	db, err = sql.Open("postgres", dbInfo)
	check(err)
	
	// Suggested defaults:
	db.SetMaxOpenConns(20) // Sane default
	db.SetMaxIdleConns(0)
    db.SetConnMaxLifetime(time.Nanosecond)
	
	prVal(db_, "db", db)
}

// Close database.
func CloseDatabase() {
	pr(db_, "CloseDatabase")
	
	open := db.Stats().OpenConnections
	if open > 0 {
		// This could also modify the return code...
		prf(db_, "failed to close %d connections!", open)
    }
	
	if db != nil {
		db.Close()
	}
}

func DbTrackOpenConnections() {
	// This could also modify the return code...
	prVal(db_, "Open connections", db.Stats().OpenConnections)
}

// Executes a query that does not return anything.  Necessary for not leaking connections.
func DbExec(query string, values ...interface{}) {
	prf(db_, "DbExec query:%#v values:%#v", query, values)
	
	// TODO: test this!!!
	
	//stmt, err := db.Prepare(query)
	//check(err)
	//
	//_, err = stmt.Exec(values...)
	//check(err)

	_, err = db.Exec(query, values...)

	//stmt.Close()
}

// Inserts a new record into the database and returns the Id of the inserted record.
// Panics on error.
func DbInsert(query string, values ...interface{}) int {
	prf(db_, "DbInsert query:%#v values:%#v", query, values)
	
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
	prf(db_, "DbQuery query:%#v values:%#v", query, values)
	
	rows, err := db.Query(query, values...)
	check(err)	
	return rows
}

// Executes a query, and TRUE if it returned any row.
// Panics on error
func DbExists(query string, values ...interface{}) bool {
	pr(db_, "DbExists")
	rows := DbQuery(query, values...)
	return rows.Next()
}

// If string is empty, convert to to NULL.
func ConvertNullString(s string) sql.NullString {
    if len(s) == 0 {
        return sql.NullString{}
    }
    return sql.NullString{
         String: s,
         Valid: true,
    }
}
