// gozilla.go
package main

import (
	"bytes"
	"fmt"
	"github.com/lib/pq"
	"log"
	"net/http"
	"text/template" // Faster than "html/template", and less of a pain for safeHTML
)

var (
	templates   map[string]*template.Template
	err		 	error
)

// Template arguments for webpage template.
type PageArgs struct {
	Title			string
	Script			string
}

///////////////////////////////////////////////////////////////////////////////
//
// frontPage
//
///////////////////////////////////////////////////////////////////////////////
func frontPageHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("frontPageHandler")
	
	args := PageArgs {
		Title: "votezilla",
	}
	args.Title = "votezilla"
	executeTemplate(w, "frontPage", args)
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
			PageArgs: PageArgs{Title: "Login"},
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
		passwordHashInts := GetPasswordHash256(data.Password)

		// TODO: Gotta send verification email... user doesn't get created until email gets verified.
		// TODO: Verify email and set emailverified=True when email is verified

		// Works: INSERT INTO votezilla.user(username,passwordhash) VALUES('asmith', '{798798,-8980,2323,6546}');
		printVal("db", db)
		
		// Check for duplicate email
		if !DbUnique("SELECT * FROM votezilla.User WHERE Email = $1;", data.Email) {
			fmt.Println("That email is taken... have you registered already?")
			
			field, err := form.GetField("email")
			assert(err)
			field.SetErrors([]string{"That email is taken... have you registered already?"})
        } else { 
        	// Check for duplicate username
			if !DbUnique("SELECT * FROM votezilla.User WHERE Username = $1;", data.Username) {
				fmt.Println("That username is taken... try another one.  Or, have you registered already?")
				field, err := form.GetField("username")
				assert(err)
				field.SetErrors([]string{"That username is taken... try another one.  Or, have you registered already?"})
			} else {
				// Add new user to the database        
				userId := DbInsert(
					"INSERT INTO votezilla.User(Email, Username, PasswordHash) VALUES ($1, $2, $3) returning id;", 
					data.Email,
					data.Username,
					pq.Array(passwordHashInts))
				
				CreateSession(w, r, userId, data.RememberMe)
				
				http.Redirect(w, r, "/registerDetails", http.StatusSeeOther)
				return	
			}
		}
	}  

	// handle GET, or invalid form data from POST...	
	{		
		args := FormArgs {
			PageArgs: PageArgs{Title: "Register"},
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
	
	userId, ok := GetSessionUserId(r)			
	if !ok { // Secure cookie not found.  Either session expired, or someone is hacking.
		// So go to the register page.
		log.Printf("secure cookie not found")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}
	
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
		
		printVal("userId", userId)
		
		log.Println(`UPDATE votezilla.User
				SET (Name, Country, Location, BirthYear, Gender, Party, Race, Marital, Schooling)
				= ($2, $3, $4, $5, $6, $7, $8, $9, $10)
				WHERE Id = $1`, 
			userId,
			data.Name,
			data.Country,
			data.Location, // TODO: remove ZipCode and City, add Location
			data.BirthYear,
			data.Gender,
			data.Party,
			pq.Array(data.Races), // TODO: change Race to Races[]
			data.Marital,
			data.Schooling)
	
		// Update the user record with registration details.
		DbQuery(
			`UPDATE votezilla.User
				SET (Name, Country, Location, BirthYear, Gender, Party, Race, Marital, Schooling)
				= ($2, $3, $4, $5, $6, $7, $8, $9, $10)
				WHERE Id = $1`, 
			userId,
			data.Name,
			data.Country,
			data.Location, // TODO: remove ZipCode and City, add Location
			data.BirthYear,
			data.Gender,
			data.Party,
			pq.Array(data.Races), // TODO: change Race to Races[]
			data.Marital,
			data.Schooling)
		
		http.Redirect(w, r, "/registerDone", http.StatusSeeOther)   
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
		
		congrats := ""
		if r.Method == "GET" {
			congrats = "Congrats for registering"
		}
		
		args := FormArgs {
			PageArgs: PageArgs{
				Title: "Voter Information",
				Script: scriptString},
			Congrats: congrats,
			Introduction: "A good voting system ensures everyone is represented.<br>" +
			              "Your information is confidential.",
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

func registerDoneHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `<h2>Congrats, you just registered</h2>
					<script>alert('Congrats, you just registered')</script>`)
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
// handler wrapper - Each request should refresh the session.
//
///////////////////////////////////////////////////////////////////////////////
func hwrap(handler func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		RefreshSession(w, r)
		
		handler(w, r)
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
	
	parseCommandLineFlags()
   
	OpenDatabase()
	defer CloseDatabase()
	
	InitSecurity()

	http.HandleFunc("/",				hwrap(frontPageHandler))
	http.HandleFunc("/login/",			hwrap(loginHandler))
	http.HandleFunc("/forgotPassword/", hwrap(forgotPasswordHandler))
	http.HandleFunc("/register/",		hwrap(registerHandler))
	http.HandleFunc("/registerDetails/",hwrap(registerDetailsHandler))
	http.HandleFunc("/registerDone/",	hwrap(registerDoneHandler))
	http.HandleFunc("/ip/",				hwrap(ipHandler))
	http.HandleFunc("/news/",			hwrap(newsHandler))
	http.HandleFunc("/newsSources/",	hwrap(newsSourcesHandler))
	
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
		
	http.ListenAndServe(":8080", nil)
	
	log.Printf("Listening on http://localhost:8080...")
}

