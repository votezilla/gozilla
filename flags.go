package main

import (
	"flag"	
	"fmt"
)

var (
	flags struct {
		dbName					string // Database name, currently 'votezilla'.
		dbUser					string // Database user.
		dbPassword				string // Database password. 
		dbSalt					string // Salt for encrypting secure information in database.
		debug	 				string // Reloads template files every time
		secureCookieHashKey		string // Secure key for encrypting secure cookies.
		secureCookieBlockKey	string // Even more secure key for encrypting secure cookies.
		newsAPIKey				string // News API key.
	}
)

///////////////////////////////////////////////////////////////////////////////
//
// flags
//
///////////////////////////////////////////////////////////////////////////////
func parseCommandLineFlags() {
	// Grab command line flags
	f1 := flag.String("dbname",			"votezilla",	"Database to connect to"); 
	f2 := flag.String("dbuser",			"",				"Database user"); 
	f3 := flag.String("dbpassword", 	"",				"Database password"); 
	f4 := flag.String("dbsalt",			"SALT",			"Database salt (for security)"); 
	f5 := flag.String("debug",		  	"",				"debug=true for development");
	f6 := flag.String("cookieHashKey",	"very-secret",	"secure cookie hash key");
	f7 := flag.String("cookieBlockKey",	"a-lot-secret", "secure cookie block key");
	f8 := flag.String("newsAPIKey",		"",				"news API key from https://newsapi.org");
	
	flag.Parse()
	
	flags.dbName				= *f1
	flags.dbUser				= *f2
	flags.dbPassword			= *f3
	flags.dbSalt				= *f4
	flags.debug					= *f5
	flags.secureCookieHashKey	= *f6
	flags.secureCookieBlockKey	= *f7
	flags.newsAPIKey			= *f8
	
	fmt.Printf("flags: %#v\n", flags)
}