package main

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"flag"	
	"fmt"
	"github.com/lib/pq"
	"io"
	"log"
	"net/http"
	"text/template" // Faster than "html/template", and less of a pain for safeHTML
)

var (
	db			*sql.DB
	templates   map[string]*template.Template
	err		 	error
	
	debug	 	string
	
	dbSalt		= "SALT" // Database salt, for storing passwords safe from database leaks.
)

///////////////////////////////////////////////////////////////////////////////
//
// utility functions
//
///////////////////////////////////////////////////////////////////////////////
func check(err error) {
    if err != nil {
        log.Fatal(err)
    }
}

func print(text string) {
	log.Println(text)
}

func printVal(label string, v interface{}) {
	log.Printf("%s: %v", label, v)
}

func printValX(label string, v interface{}) {
	log.Printf("%s: %x", label, v)
}


func parseCommandLineFlags() (string, string, string, string, string) {
	// Grab command line flags
	f1 := flag.String("dbname",		"votezilla", "Database to connect to")	  ; 
	f2 := flag.String("dbuser",		"",		  "Database user")			   ; 
	f3 := flag.String("dbpassword", "",		  "Database password")		   ; 
	f4 := flag.String("dbsalt",		"",		  "Database salt (for security)"); 
	f5 := flag.String("debug",		"",		  "debug=true for development")  ; 
	flag.Parse()
	
	return *f1, *f2, *f3, *f4, *f5
}

func openDatabase(dbName, dbUser, dbPassword string) {
	print("openDatabase")
	
	// Connect to database
	dbInfo := fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable",
		dbUser, dbPassword, dbName)  

	fmt.Printf("dbInfo: %s", dbInfo)

	db, err = sql.Open("postgres", dbInfo)
	fmt.Println("err:", err)
	check(err)
	
	printVal("db", db)
}

func closeDatabase() {
	print("closeDatabase")
	
	if db != nil {
		defer db.Close()
	}
}


///////////////////////////////////////////////////////////////////////////////
//
// render template files
//
///////////////////////////////////////////////////////////////////////////////
func parseTemplateFiles() {
	log.Println(1)

	T := func(page string) string {
		return "templates/" + page + ".html"
	}

	templates = make(map[string]*template.Template)
	
	// HTML templates
	templates["test_index"]		= template.Must(template.ParseFiles(T("test_base"), T("test_index")))
	templates["frontPage"]		= template.Must(template.ParseFiles(T("base"), T("frontPage")))
	templates["form"]			= template.Must(template.ParseFiles(T("base"), T("form")))
	
	// Javascript snippets
	templates["registerDetailsScript"]	= template.Must(template.ParseFiles(T("registerDetailsScript")))
}

func executeTemplate(w http.ResponseWriter, templateName string, data interface{}) {
	log.Printf("executeTemplate: " + templateName)
	
	if debug != "" {
		parseTemplateFiles()
	}

	err := templates[templateName].Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// writes to io.Writer instead of http.ResponseWriter
func renderTemplate(w io.Writer, templateName string, data interface{}) {
	log.Printf("renderTemplate: " + templateName)
	
	if debug != "" {
		parseTemplateFiles()
	}

	err := templates[templateName].Execute(w, data)
	check(err)
}

// Render the table form, return the HTML string
func getFormHtml(tableForm TableForm) string {
	var formHTML bytes.Buffer
	renderTemplate(&formHTML, "tableForm", tableForm)
	return formHTML.String()
}


///////////////////////////////////////////////////////////////////////////////
//
// frontPage
//
///////////////////////////////////////////////////////////////////////////////
func frontPageHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("frontPageHandler")
	
	var args struct{
		Title string
	}
	args.Title = "votezilla"
	executeTemplate(w, "frontPage", args)
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("testHandler")
	
	var args struct{
		Title string
	}
	args.Title = "votezilla"
	executeTemplate(w, "test_index", args)
}


///////////////////////////////////////////////////////////////////////////////
//
// login
//
///////////////////////////////////////////////////////////////////////////////
func loginHandler(w http.ResponseWriter, r *http.Request) {	
	form := LoginForm(r)

	if r.Method == "POST" && form.IsValid(){ // Handle POST, with valid data...
	}
	
	// handle GET, or invalid form data from POST...	
	{	
		args := FormArgs {
			Title: "Login",
			Footer: `<a href="/forgotPassword">Forgot your password?</a>`,
			Forms: []TableForm{{
				Form: form,
				CallToAction: "Login",
		}}}
		executeTemplate(w, "form", args)
	}
}

///////////////////////////////////////////////////////////////////////////////
//
// forgotPassword
//
///////////////////////////////////////////////////////////////////////////////
func forgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
}

///////////////////////////////////////////////////////////////////////////////
//
// register
//
///////////////////////////////////////////////////////////////////////////////
func registerHandler(w http.ResponseWriter, r *http.Request) {
	
	form := RegisterForm(r)
	tableForm := TableForm{
		Form: form,
		CallToAction: "Register",
	}

	if r.Method == "POST" && form.IsValid() { // Handle POST, with valid data...
		// Parse POST data into "data".
		data := RegisterData{}
		form.MapTo(&data)

		// Use a hashed password for security.
		passwordHash := sha256.Sum256([]byte(data.Password + dbSalt))
		var passwordHashInts[4]int64
		err:= binary.Read(bytes.NewBuffer(passwordHash[:]), binary.LittleEndian, &passwordHashInts)

		// TODO: CHECK FOR DUPLICATE USERNAME OR EMAIL
		// TODO: GOTTA SEND VERIFICATION EMAIL... USER DOESN'T GET CREATED UNTIL EMAIL GETS VERIFIED
		// INSERT IT FOR NOW, TODO: VERIFY EMAIL AND SET emailverified=True when email is verified

		// Works: INSERT INTO votezilla.user(username,passwordhash) VALUES('asmith', '{798798,-8980,2323,6546}');
		printVal("db", db)

		var lastInsertId int
		err = db.QueryRow(
			"INSERT INTO votezilla.User(Email,PasswordHash) VALUES ($1, $2) returning id;", 
			data.Email,
			pq.Array(passwordHashInts),
		).Scan(&lastInsertId)
		printVal("err", err)
		fmt.Println("lastInsertId =", lastInsertId)
		check(err)

		fmt.Println("next line")
		http.Redirect(w, r, "/registerDetails", http.StatusSeeOther)   

		return	
	}  

	// handle GET, or invalid form data from POST...	
	{		
		args := FormArgs {
			Title: "Register",
			Forms: []TableForm{
				tableForm,
		}}
		executeTemplate(w, "form", args)
	}
}

///////////////////////////////////////////////////////////////////////////////
//
// register details about the user
//
///////////////////////////////////////////////////////////////////////////////
func registerDetailsHandler(w http.ResponseWriter, r *http.Request) {
	
	form := RegisterDetailsForm(r)
	
	if r.Method == "POST" && form.IsValid(){ // Handle POST, with valid data...

		// Passwords match, everything is good - Register the user

		// Parse POST data into "data".
		data := RegisterDetailsData{}
		form.MapTo(&data)
		
		fmt.Fprintf(w, "<br><p>country: %+v</p>", data.Country)
		
		fmt.Fprintf(w, "<br>races: %T - %+v", data.Races, data.Races)
		for k, v := range data.Races {
			fmt.Fprintf(w, "<br>%v -> %v", k, v)
		}
		
		fmt.Fprintf(w, "<br><p>data: %+v</p>", data)

	/*
		var lastInsertId int
		err = db.QueryRow(
			"UPDATE votezilla.user(username,passwordhash,email) VALUES ($1, $2, $3) returning id;", 
			data.Username, 
			pq.Array(passwordHashInts),
			data.Email,
		).Scan(&lastInsertId)
		printVal("err", err)
		fmt.Println("lastInsertId =", lastInsertId)
		check(err)

		fmt.Println("next line")
		http.Redirect(w, r, "/registerDone", http.StatusSeeOther)   
	*/
	
		return	
	} 
	
	// handle GET, or invalid form data from POST...	
	{
		// render registerDetailsScript template
		var scriptString string
		{
			scriptData := struct {
				CountriesWithStates			map[string]bool
				CountriesWithPostalCodes	map[string]bool
			}{
			    CountriesWithStates,
			    CountriesWithPostalCodes,
			}
			
			var scriptHTML bytes.Buffer
			renderTemplate(&scriptHTML, "registerDetailsScript", scriptData)
			scriptString = scriptHTML.String()
		}
		
		args := FormArgs {
			Title: "Voter Information",
			Introduction: "A good voting system ensures everyone is represented.<br>" +
			              "Your information is confidential.",
			Script: scriptString,
			Forms: []TableForm{{
				Form: form,
				CallToAction: "Submit",
		}}}
		executeTemplate(w, "form", args)
	}
	
	// Debug info:
	form.IsValid()
	data := RegisterDetailsData{}
	form.MapTo(&data)
	
	fmt.Fprintf(w, "<br>races: %T - %+v", data.Races, data.Races)
	for k, v := range data.Races {
	    fmt.Fprintf(w, "<br>%v -> %v", k, v)
	}	
	
	fmt.Fprintf(w, "<br>data: %T - %+v", data, data)

	fmt.Fprintf(w, "<br>r: %+v", r)
}

///////////////////////////////////////////////////////////////////////////////
//
// TODO: get user's ip address
//       1) To log in the database when user is first created.
//		 2) To set their location in registerDetails and save them time.
// USING: https://play.golang.org/p/Z6ATIhL_IM
//        https://stackoverflow.com/questions/27234861/correct-way-of-getting-clients-ip-addresses-from-http-request-golang
//
// (WAIT TIL TESTING FROM AWS, OTHERWISE IT'S LOCALHOST, BASICALLY MEANINGLESS)
//
///////////////////////////////////////////////////////////////////////////////
func ipHandler(w http.ResponseWriter, r *http.Request) {
	remoteAddr	 := r.RemoteAddr
	forwardedFor := r.Header.Get("X-Forwarded-For")
	
	fmt.Fprintf(w, "<p>remote addr: %s</p>", remoteAddr)
	fmt.Fprintf(w, "<p>forwarded for: %s</p>", forwardedFor)
	fmt.Fprintf(w, "<br><p>r: %+v</p>", r)
}

///////////////////////////////////////////////////////////////////////////////
//
// program entry
//
///////////////////////////////////////////////////////////////////////////////
func init() {
	log.Printf("init")
	
	parseTemplateFiles()
}

func main() {
	log.Printf("main")
	
	var dbName, dbUser, dbPassword string
	dbName, dbUser, dbPassword, dbSalt, debug = parseCommandLineFlags()
  
	fmt.Println("dbName", dbName)
	fmt.Println("dbUser", dbUser)
	fmt.Println("dbPassword", dbPassword)
	fmt.Println("dbSalt", dbSalt)
	fmt.Println("debug", debug)
   
	openDatabase(dbName, dbUser, dbPassword)
	defer closeDatabase()

	http.HandleFunc("/",				frontPageHandler)
	http.HandleFunc("/test/",			testHandler)
	http.HandleFunc("/login/",			loginHandler)
	http.HandleFunc("/forgotPassword/", forgotPasswordHandler)
	http.HandleFunc("/register/",		registerHandler)
	http.HandleFunc("/registerDetails/",registerDetailsHandler)
	http.HandleFunc("/ip/",				ipHandler)
	
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
		
	http.ListenAndServe(":8080", nil)
	
	log.Printf("Listening on http://localhost:8080...")
}

