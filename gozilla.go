// gozilla.go
package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"github.com/lib/pq"
	"log"
	"net/http"
	"strings"
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
// login
//
///////////////////////////////////////////////////////////////////////////////
func loginHandler(w http.ResponseWriter, r *http.Request) {	
	userId := GetSession(r)
	assert(userId == -1) // User must not be already logged in!
	
	form := LoginForm(r)
	
	if r.Method == "POST" && form.IsValid(){ // Handle POST, with valid data...
		// Parse POST data.
		data := LoginData{}
		form.MapTo(&data)
		
		var rows *sql.Rows
		if strings.Contains(data.EmailOrUsername, "@") {
			rows = DbQuery("SELECT Id, PasswordHash[1], PasswordHash[2], PasswordHash[3], PasswordHash[4] " + 
							"FROM votezilla.User WHERE Email = $1;", 
							data.EmailOrUsername)
		} else {
			rows = DbQuery("SELECT Id, PasswordHash[1], PasswordHash[2], PasswordHash[3], PasswordHash[4] " + 
							"FROM votezilla.User WHERE Username = $1;", 
							data.EmailOrUsername)
		}
		
		var userId int
		var passwordHashInts int256			

		defer rows.Close()
		if !rows.Next() {
			field, err := form.GetField("email or username")
			assert(err)
			field.SetErrors([]string{"That email does not exist. Do you need to register?"})
		} else {
			err := rows.Scan(&userId, &passwordHashInts[0], &passwordHashInts[1], &passwordHashInts[2], &passwordHashInts[3]);
			check(err)
			check(rows.Err())
			fmt.Printf("User found! - id: '%d' passwordHashInts: %#v\n", userId, passwordHashInts)	
		
			passwordHash := GetPasswordHash256(data.Password)		
			if  passwordHash[0] != passwordHashInts[0] ||
				passwordHash[1] != passwordHashInts[1] ||
				passwordHash[2] != passwordHashInts[2] ||
				passwordHash[3] != passwordHashInts[3] {

				field, err := form.GetField("password")
				assert(err)
				field.SetErrors([]string{"Invalid password.  Forgot password?"})	
			} else {				
				CreateSession(w, r, userId, data.RememberMe)

				serveHTML(w, `<h2>Successfully logged in</h2>
								  <script>alert('Successfully logged in')</script>`)
				return		// TODO: add redirect here!!!
			}
		}
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
// logout
//
///////////////////////////////////////////////////////////////////////////////
func logoutHandler(w http.ResponseWriter, r *http.Request) {	
	DestroySession(w, r)
	
	serveHTML(w, `<h2>Successfully logged out</h2>
				  <script>alert('Successfully logged out')</script>`)
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
		// Parse POST data.
		data := RegisterData{}
		form.MapTo(&data)

		// Use a hashed password for security.
		passwordHashInts := GetPasswordHash256(data.Password)

		// TODO: Gotta send verification email... user doesn't get created until email gets verified.
		// TODO: Verify email and set emailverified=True when email is verified
		
		// Check for duplicate email
		if DbExists("SELECT * FROM votezilla.User WHERE Email = $1;", data.Email) {
			fmt.Println("That email is taken... have you registered already?")
			
			field, err := form.GetField("email")
			assert(err)
			field.SetErrors([]string{"That email is taken... have you registered already?"})
        } else { 
        	// Check for duplicate username
			if DbExists("SELECT * FROM votezilla.User WHERE Username = $1;", data.Username) {
				fmt.Println("That username is taken... try another one.  Or, have you registered already?")
				field, err := form.GetField("username")
				assert(err)
				field.SetErrors([]string{"That username is taken... try another one.  Or, have you registered already?"})
			} else {
				// Add new user to the database   
				fmt.Printf("passwordHashInts[0]: %T %#v\n", passwordHashInts[0], passwordHashInts[0])
				userId := DbInsert(
					"INSERT INTO votezilla.User(Email, Username, PasswordHash) " +
						"VALUES ($1, $2, ARRAY[$3::bigint, $4::bigint, $5::bigint, $6::bigint]) returning id;", 
					data.Email,
					data.Username,
					passwordHashInts[0],
					passwordHashInts[1],
					passwordHashInts[2],
					passwordHashInts[3])
				
				// Create session (encrypted userId).
				CreateSession(w, r, userId, data.RememberMe)
				// Set "RememberMe" cookie
				if data.RememberMe {
					setCookie(w, r, "RememberMe", "true", longExpiration(), false)
				} else {
					setCookie(w, r, "RememberMe", "false", longExpiration(), false)
				}
				
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
	RefreshSession(w, r)
	
	form := RegisterDetailsForm(r)
	
	userId := GetSession(r)			
	if userId == -1 { // Secure cookie not found.  Either session expired, or someone is hacking.
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
		
		printVal("userId", userId)
	
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
			congrats = "Congrats for registering" // Congrats for registering... now enter more information.
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
}

func registerDoneHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)
	serveHTML(w, `<h2>Congrats, you just registered</h2>
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
		if flags.debug != "" {
			printVal("Handling request from: ", formatRequest(r))
		}
		
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
   
	InitNews()
	
	OpenDatabase()
	defer CloseDatabase()	
	
	InitSecurity()
	
	http.HandleFunc("/",                hwrap(newsHandler))
	http.HandleFunc("/login/",          hwrap(loginHandler))
	http.HandleFunc("/logout/",         hwrap(logoutHandler))
	http.HandleFunc("/forgotPassword/", hwrap(forgotPasswordHandler))
	http.HandleFunc("/register/",       hwrap(registerHandler))
	http.HandleFunc("/registerDetails/",hwrap(registerDetailsHandler))
	http.HandleFunc("/registerDone/",   hwrap(registerDoneHandler))
	http.HandleFunc("/ip/",             hwrap(ipHandler))
	http.HandleFunc("/newsSources/",    hwrap(newsSourcesHandler))
	
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
		
	http.ListenAndServe(":8080", nil)
	
	log.Printf("Listening on http://localhost:8080...")
}

