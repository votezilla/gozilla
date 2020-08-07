package main

import (
	"flag"
	//"strconv"
	"os"
)

var (
	flags struct {
		dbName					string 		// Database name, currently 'votezilla'.
		dbUser					string 		// Database user.
		dbPassword				string 		// Database password.
		dbSalt					string 		// Salt for encrypting secure information in database.
		debug	 				string 		// Reloads template files every time
		secureCookieHashKey		string 		// Secure key for encrypting secure cookies.
		secureCookieBlockKey	string		// Even more secure key for encrypting secure cookies.
		newsAPIKey				string		// News API key.
		//printMask				PrintMask	// For selective logging.
		port					string		// Which port to serve webpages from.
		offlineNews				string		// Whether to use offline cached news, when working without Internet connection.
		newsService				string		// Whether to be the news service.
		imageService			string		// Whether to be the image service.
		test					string		// Whether to run a simple test, instead of the actual service.
		mode					string		// A special mode to run a service in.
		testUserId				string		// UserId to test being loggin in as.
		isNewsAccelerated		string		// Whether News API queries should be accelerated
		randomizeTime			string

		skipWhitelist			bool
		inProduction			bool		// If true, we start HTTPS server
		redirectHTTPToHTTPS		bool		// If true, we redirect HTTP to HTTPS
		domain				string
	}
)

///////////////////////////////////////////////////////////////////////////////
//
// flags
//
///////////////////////////////////////////////////////////////////////////////
func parseCommandLineFlags() {
	// Grab command line flags
	f1 := flag.String("dbname",				"vz",			"Database to connect to")
	f2 := flag.String("dbuser",				"",				"Database user")
	f3 := flag.String("dbpassword", 		"",				"Database password")
	f4 := flag.String("dbsalt",				"SALT",			"Database salt (for security)")
	f5 := flag.String("debug",		  		"",				"debug=true for development")
	f6 := flag.String("cookieHashKey",		"very-secret",	"secure cookie hash key")
	f7 := flag.String("cookieBlockKey",		"a-lot-secret", "secure cookie block key")
	f8 := flag.String("newsAPIKey",			"",				"news API key from https://newsapi.org")
	//f9 := flag.String("printMask",		"65535",		"log output mask")
	fa := flag.String("port",				"8080",			"which port to serve webpages from")
	fb := flag.String("offlineNews",    	"",				"whether to use offline news")
	fc := flag.String("newsService",		"",				"whether to be the news service")
	fd := flag.String("imageService",		"",				"whether to be the image service")
	fe := flag.String("test",				"",				"whether to run a simple test, instead of the actual server")
	ff := flag.String("mode",				"",				"a special mode to run a server in")
	fg := flag.String("testUserId",			"",				"UserId to test being loggin in as")
	fh := flag.String("isNewsAccelerated",	"",				"Whether News API queries should be accelerated")
	fi := flag.String("randomizeTime", 		"true",			"True to randomize article order a little, in /news.")

	flag.BoolVar(&flags.skipWhitelist, 		 "skipWhitelist", false, "skip reading the whitelist, which is slow")
	flag.BoolVar(&flags.inProduction, 		 "inProduction", false, "if true, we start HTTPS server")
	flag.BoolVar(&flags.redirectHTTPToHTTPS, "redirect-to-https", false, "if true, we redirect HTTP to HTTPS")
	flag.StringVar(&flags.domain, "domain", "", "domain name to request your certificate")

	prVal("Command Line Args", os.Args)

	flag.Parse()

	flags.dbName				= *f1
	flags.dbUser				= *f2
	flags.dbPassword			= *f3
	flags.dbSalt				= *f4
	flags.debug					= *f5
	flags.secureCookieHashKey	= *f6
	flags.secureCookieBlockKey	= *f7
	flags.newsAPIKey			= *f8
/*
	printMask, err        		:= strconv.Atoi(*f9)
	flags.printMask = PrintMask(printMask)
	if err != nil {
		flags.printMask = PrintMask(all_)
	}
*/
	flags.port					= *fa
	flags.offlineNews			= *fb
	flags.newsService			= *fc
	flags.imageService			= *fd
	flags.test					= *fe
	flags.mode					= *ff
	flags.mode					= *ff
	flags.testUserId			= *fg
	flags.isNewsAccelerated		= *fh
	flags.randomizeTime			= *fi

	prf("flags: %#v\n", flags)
}
