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
		isNewsAccelerated		string		// Whether News API queries should be accelerated.
		randomizeTime			string
		skipWhitelist			bool
		domain					string
		cachingService			string		// Whether to be the caching service.
		requirePassword			bool		// Whether to require passwords for logging in.
		checkForNotifications	bool		// Whether to do ajaxCheckForNotifications
		separateNewsAndPolls	bool		// Whether to separate news and polls in separate tabs
		skipFirewall			bool		// Whether to skip the firewall
		testEmail				bool		// Send a test email and exit
		dailyEmail				bool		// Send the daily email and exit
		smtpPassword			string		// Password for sending email to the SMTP server
		dryRun					bool		// If true, email message is generated but not sent
		featuredArticleId		int			// For the daily poll email, the main article (i.e. poll) to share.")
		emailTarget				string		// Target for the batch email, e.g. 'Daily', 'Test', one day... 'Weekly' and 'Monthly'
		testEmailAddress		string		// Test email address
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
	fj := flag.String("cachingService",		"",				"whether to be the caching service")

	flag.BoolVar(&flags.skipWhitelist, 		 "skipWhitelist", false, "skip reading the whitelist, which is slow")
	flag.StringVar(&flags.domain, "domain", "", "domain name to request your certificate")
	flag.BoolVar(&flags.requirePassword, 	"requirePassword", false, "Whether to require passwords for logging in.")
	flag.BoolVar(&flags.checkForNotifications, "checkForNotifications", true, "Whether to do ajaxCheckForNotifications.")
	flag.BoolVar(&flags.separateNewsAndPolls, "separateNewsAndPolls", false, "Whether to separate news and polls in separate tabs.")
	flag.BoolVar(&flags.skipFirewall, "skipFirewall", true, "Whether to skip the firewall")
	flag.BoolVar(&flags.testEmail, "testEmail", false, "Send a test email and exit")
	flag.BoolVar(&flags.dailyEmail, "dailyEmail", false, "Send the daily email and exit")


	flag.StringVar(&flags.smtpPassword, "smtpPassword", "", "Password for sending email to the SMTP server")
	flag.BoolVar(&flags.dryRun, "dryRun", true, "If true, email message is generated but not sent")
	flag.IntVar(&flags.featuredArticleId, "featuredArticleId", -1, "For the daily poll email, the main article (i.e. poll) to share.")
	flag.StringVar(&flags.emailTarget, "emailTarget", "", "Target for the batch email, e.g. 'Daily', 'Test', one day... 'Weekly' and 'Monthly'")
	flag.StringVar(&flags.testEmailAddress, "testEmailAddress", "", "Test email address")

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
	flags.cachingService		= *fj

	prf("flags: %#v\n", flags)
}
