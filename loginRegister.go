// gozilla.go
package main

import (
	"bytes"
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// Common field names
	kEmailOrUsername = "email or username"
	kEmail		 = "email"
	kUsername	 = "username"
	kPassword        = "password"
	kRememberMe		 = "remember me"
)


///////////////////////////////////////////////////////////////////////////////
//
// login
//
///////////////////////////////////////////////////////////////////////////////
func loginHandler(w http.ResponseWriter, r *http.Request) {
	userId := GetSession(r)
	assert(userId == -1) // User must not be already logged in!

	bRememberMe := str_to_bool(GetCookie(r, kRememberMe, "false"))

	form := makeForm(
		MakeTextField(kEmailOrUsername, 50, 6, 345),
		MakePasswordField(kPassword, 50, 8, 40),
		MakeBoolField(kRememberMe, true),
	)

	if r.Method == "POST" && form.validateData(r) { // On POST, validates and captures the request data.
		prVal("form", form)

		emailOrUsername := form.val(kEmailOrUsername)

		isEmail := strings.Contains(emailOrUsername, "@")

		var rows *sql.Rows
		if isEmail {
			rows = DbQuery("SELECT Id, PasswordHash[1], PasswordHash[2], PasswordHash[3], PasswordHash[4] " +
							"FROM $$User WHERE Email = $1;",
							emailOrUsername)
		} else {
			rows = DbQuery("SELECT Id, PasswordHash[1], PasswordHash[2], PasswordHash[3], PasswordHash[4] " +
							"FROM $$User WHERE Username = $1;",
							emailOrUsername)
		}

		var userId int64
		var passwordHashInts int256

		defer rows.Close()
		if !rows.Next() {
			if isEmail {
				form.setFieldError(kEmailOrUsername, "That email does not exist. Do you need to register?")
			} else {
				form.setFieldError(kEmailOrUsername, "That username does not exist. Do you need to register?")
			}
		} else {
			err := rows.Scan(&userId, &passwordHashInts[0], &passwordHashInts[1], &passwordHashInts[2], &passwordHashInts[3]);
			check(err)
			check(rows.Err())
			prf("User found! - id: '%d' passwordHashInts: %#v\n", userId, passwordHashInts)

			passwordHash := GetPasswordHash256(form.val(kPassword))
			if passwordHash[0] != passwordHashInts[0] ||
			   passwordHash[1] != passwordHashInts[1] ||
			   passwordHash[2] != passwordHashInts[2] ||
			   passwordHash[3] != passwordHashInts[3] {
				form.setFieldError(kPassword, "Invalid password.  Forgot password?")
			} else {
				bRememberMe = form.boolVal(kRememberMe)

				CreateSession(w, r, userId, bRememberMe)

				http.Redirect(w, r, "/news?alert=LoggedIn", http.StatusSeeOther)
				return
			}
		}
	}

	// handle GET, or invalid form data from POST...
	{
		args := FormArgs {
			PageArgs: PageArgs{Title: "Login"},
			Footer: `<a href="/forgotPassword">Forgot your password?</a>`,
			Form: TableForm {
				Form: *form,
				CallToAction: "Login",
		}}
		executeTemplate(w, kForm, args)
	}
}

///////////////////////////////////////////////////////////////////////////////
//
// logout
//
///////////////////////////////////////////////////////////////////////////////
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	DestroySession(w, r)

	// TODO: /logout should bring up a pop-up.  This will fix the session cookie bug!!
	//       (which is, UserId cookie gets cleared, then re-set by http.Redirect below:

	//http.Redirect(w, r, "/news?alert=LoggedOut", http.StatusSeeOther)
	return
}

///////////////////////////////////////////////////////////////////////////////
//
// forgotPassword
//
///////////////////////////////////////////////////////////////////////////////
func forgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	nyi()  // TODO: implement forgotPassword
}

///////////////////////////////////////////////////////////////////////////////
//
// register
//
///////////////////////////////////////////////////////////////////////////////
func registerHandler(w http.ResponseWriter, r *http.Request) {

	form := makeForm(
		MakeTextField(kEmail, 50, 6, 345),
		MakeTextField(kUsername, 50, 4, 345).noSpellCheck(),
		MakePasswordField(kPassword, 40, 8, 40),
		MakeBoolField(kRememberMe, true),
	)

	// Validate the password is complex enough.
	form.field(kPassword).addFnValidator(
		func(pw string) (bool, string) {
			password := MakePassword(pw)
			password.ProcessPassword()
			if password.CommonPassword {
				return false, "Your password is a common password.  Try making it harder to guess."
			}
			if password.Score < 3 {
				return false, "Your password is too simple.  Try adding lower and uppercase characters, numbers, and/or special characters."
			}
			return true, ""
		})

	form.field(kUsername).addFnValidator(
		func(username string) (bool, string) {
			if strings.Contains(form.val(kEmail), username) {
				return false, "Username cannot be contained in the email."
			}
			return true, ""
		})

	if r.Method == "POST" && form.validateData(r) { // Handle POST, with valid data...

		// Use a hashed password for security.
		passwordHashInts := GetPasswordHash256(form.val(kPassword))

		// TODO: Gotta send verification email... user doesn't get created until email gets verified.
		// TODO: Verify email and set emailverified=True when email is verified

		// Check for duplicate email
		if DbExists("SELECT * FROM $$User WHERE Email = $1;", form.val(kEmail)) {
			pr("That email is taken... have you registered already?")

			form.setFieldError(kEmail, "That email is taken... have you registered already?")
        } else {
        	// Check for duplicate username
			if DbExists("SELECT * FROM $$User WHERE Username = $1;", form.val(kUsername)) {
				pr("That username is taken... try another one.  Or, have you registered already?")
				form.setFieldError(kUsername, "That username is taken... try another one.  Or, have you registered already?")
			} else {
				// Add new user to the database
				prf("passwordHashInts[0]: %T %#v\n", passwordHashInts[0], passwordHashInts[0])
				userId := DbInsert(
					"INSERT INTO $$User(Email, Username, PasswordHash) " +
					"VALUES ($1, $2, ARRAY[$3::bigint, $4::bigint, $5::bigint, $6::bigint]) returning id;",
					form.val(kEmail),
					form.val(kUsername),
					passwordHashInts[0],
					passwordHashInts[1],
					passwordHashInts[2],
					passwordHashInts[3])

				// Create session (encrypted userId).
				CreateSession(w, r, userId, form.boolVal(kRememberMe))

				http.Redirect(w, r, "/registerDetails", http.StatusSeeOther)
				return
			}
		}
	}

	// handle GET, or invalid form data from POST...
	{
		args := FormArgs {
			PageArgs: PageArgs{Title: "Register"},
			Form: TableForm{
				Form: *form,
				CallToAction: "Register",
		}}
		executeTemplate(w, kForm, args)
	}
}

///////////////////////////////////////////////////////////////////////////////
//
// register details about the user
//
///////////////////////////////////////////////////////////////////////////////
func registerDetailsHandler(w http.ResponseWriter, r *http.Request){//, userId int64) {
	//RefreshSession(w, r)

	const (
		kName = "name"
		kZipCode = "zip code"
		kBirthYear = "birth year"
		kCountry = "country"
		kGender = "gender"
		kParty = "party"
		kRace = "race"
		kMaritalStatus = "marital status"
		kSchoolCompleted = "school completed"
	)

	form := makeForm(
		makeTextField(kName, "full name:", "Your full name...", 50, 1, 100),
		makeTextField(kZipCode, "zip code:", "Your zip code...", 5, 4, 10),
		makeTextField(kBirthYear, "birth year:", "Your birth year...", 4, 4, 4),
		MakeSelectField(kCountry, countries, true, true, false),
		MakeSelectField(kGender, genders, true, true, true),
		MakeSelectField(kParty, parties, true, true, true),
		MakeSelectField(kRace, races, true, true, true),
		MakeSelectField(kMaritalStatus, maritalStatuses, true, true, true),
		MakeSelectField(kSchoolCompleted, schoolDegrees, true, true, true),
	)

	form.field(kName).addRegexValidator(`^[\p{L}]+( [\p{L}]+)+$`, "Enter a valid full name (i.e. 'John Doe').")
	form.field(kZipCode).addRegexValidator(`^\d{5}(?:[-\s]\d{4})?$`, "Invalid zip code")  // TODO: different countries have different zip code formats.
	form.field(kBirthYear).addFnValidator(
		func(input string) (bool, string) {
			year, err := strconv.Atoi(input)
			if err != nil {
				return false, "Please enter a valid year."
			}
			currentYear := time.Now().Year()
			age := currentYear - year // true age would be either this expression, or this minus 1
			if age < 0 || age > 200 {
				return false, "Please enter the year you were born."
			} else {
				return true, ""
			}
		})

	userId := GetSession(r)
	if userId == -1 { // Secure cookie not found.  Either session expired, or someone is hacking.
		// So go to the register page.
		pr("secure cookie not found")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	if r.Method == "POST" && form.validateData(r) { // Handle POST, with valid data...
		// Passwords match, everything is good - Register the user

		prVal("userId", userId)

		// Update the user record with registration details.
		DbQuery(
			`UPDATE $$User
				SET (Name, Country, Location, BirthYear, Gender, Party, Race, Marital, Schooling)
				= ($2, $3, $4, $5, $6, $7, $8, $9, $10)
				WHERE Id = $1::bigint`,
			userId,
			form.val(kName),
			form.val(kCountry),
			form.val(kZipCode),
			form.val(kBirthYear),
			form.val(kGender),
			form.val(kParty),
			form.val(kRace),  // TODO: I think this should multi-select input, with a comma-delimited join of races here.
			form.val(kMaritalStatus),
			form.val(kSchoolCompleted))

		serveHTML(w, `<h2>Congrats, you just registered</h2>
					  <script>alert('Congrats, you just registered')</script>`)
		// TODO: do registration as a pop-up.  Commenting out this for now, as it breaks the UserId cookie:
		//http.Redirect(w, r, "/news?alert=AccountCreated", http.StatusSeeOther)
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
			Form: TableForm{
				Form: *form,
				CallToAction: "Submit",
		}}
		executeTemplate(w, kForm, args)
	}
}
