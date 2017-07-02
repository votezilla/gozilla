package main


import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"flag"	
	"fmt"
	"github.com/bluele/gforms"
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
	
	dbSalt = "SALT" // Database salt, for storing passwords safe from database leaks.
)

type TableForm struct {
	Form			*gforms.FormInstance
	CallToAction	string
	AdditionalError string
}

type FormArgs struct{
	Forms			[]TableForm
	Title			string
	Introduction	string
	Footer			string
}

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
	templates["test_index"] = template.Must(template.ParseFiles(T("test_base"), T("test_index")))
	templates["frontPage"] = template.Must(template.ParseFiles(T("base"), T("frontPage")))
	templates["form"]	   = template.Must(template.ParseFiles(T("base"), T("form")))

//	printVal("templates", templates)
//	printVal(`templates["test_index"]`, templates["test_index"])
//	printVal(`templates["frontPage"]`, templates["frontPage"])
//	printVal(`templates["form"]`, templates["form"])
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
	
	if r.Method == "POST" && form.IsValid(){ // Handle POST, with valid data...
		// Non-matching passwords
		if !MatchingPasswords(form) {
			tableForm.AdditionalError = "Passwords must match"
		} else { 
			// Passwords match, everything is good - Register the user
			
			// Parse POST data into "data".
			data := RegisterData{}
			form.MapTo(&data)
			
			// Use a hashed password for security.
			
			passwordHash := sha256.Sum256([]byte(data.Password + dbSalt))
			
			var passwordHashInts[4]int64
			err:= binary.Read(bytes.NewBuffer(passwordHash[:]), binary.LittleEndian, &passwordHashInts)
			printVal("err", err)
			check(err)
			
			// TODO: GOTTA SEND VERIFICATION EMAIL... USER DOESN'T GET CREATED UNTIL EMAIL GETS VERIFIED
			// INSERT IT FOR NOW, TODO: VERIFY EMAIL AND SET emailverified=True when email is verified
			
			// Works: INSERT INTO votezilla.user(username,passwordhash) VALUES('asmith', '{798798,-8980,2323,6546}');
			printVal("db", db)
			
			printVal("data.Username", data.Username)
			printVal("passwordHashInts", passwordHashInts)
			
			var lastInsertId int
			err = db.QueryRow(
				"INSERT INTO votezilla.user(username,passwordhash) VALUES ($1, $2) returning id;", 
				data.Username, 
				pq.Array(passwordHashInts),
			).Scan(&lastInsertId)
			printVal("err", err)
			fmt.Println("lastInsertId =", lastInsertId)
			check(err)
			
			fmt.Println("next line")
			http.Redirect(w, r, "/registerDetails", http.StatusSeeOther)   
			
			return	
	
		}
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
	
	form := PersonalInfoForm(r)
	
	if r.Method == "POST" && form.IsValid(){ // Handle POST, with valid data...
	}
	
	// handle GET, or invalid form data from POST...	
	{
		args := FormArgs {
			Title: "Voter Information",
			Introduction: `Please answer a few questions so we can verify you are a real person.<br><br>
This also helps ensure that all citizens have a vote and a voice.`,
			Forms: []TableForm{{
				Form: form,
				CallToAction: "Submit",
		}}}
		executeTemplate(w, "form", args)
	}
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
	
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
		
	http.ListenAndServe(":8080", nil)
	
	log.Printf("Listening on http://localhost:8080...")
}

