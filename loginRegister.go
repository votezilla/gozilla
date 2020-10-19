// gozilla.go
package main

import (
	//"bytes"
	"database/sql"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	// Common field names
	kEmailOrUsername = "emailOrUsername"
	kEmail		 = "email"
	kUsername	 = "username"
	kPassword        = "password"
	kConfirmPassword = "confirmPassword"

	// String Constants (mirrored in frame.html):
	// Cookie names:
	kLoginReturnAddress 	= "loginReturnAddress"
	kAlertCode          	= "alertCode"

	// Alert codes:
	kLoggedIn				= "LoggedIn"
	kLoggedOut				= "LoggedOut"
	kWelcomeToVotezilla		= "WelcomeToVotezilla"
	kInvalidCategory		= "InvalidCategory"
	kPreferencesSaved		= "PreferencesSaved"
)

func makeLoginForm() *Form {
	if flags.requirePassword {
		return makeForm(
			nuTextField(kEmailOrUsername, "Email / Username", 50, 6, 345, "email / username").noSpellCheckOrCaps(),
			nuPasswordField(kPassword, "Password", 50, 8, 40),
		)
	} else {
		return makeForm(
			nuTextField(kEmailOrUsername, "Email", 50, 6, 345, "email").noSpellCheckOrCaps(),
		)
	}
}

func makeRegisterForm() *Form {
	form := makeForm(
		nuTextField(kEmail, "Email", 50, 6, 345, "email").noSpellCheckOrCaps().addFnValidator(emailValidator()),
		nuTextField(kUsername, "Pick a Username", 50, 4, 25, "username").noSpellCheckOrCapsOrAutocomplete(),
	)

	if flags.requirePassword {
		form.addField(nuPasswordField(kPassword, "Create Password", 40, 8, 40))
		form.addField(nuPasswordField(kConfirmPassword, "Confirm Password", 40, 8, 40).noDefaultValidators())
	}

	return form
}


// After login, return to the article or page you were interacting with.
func gotoReturnAddress(w http.ResponseWriter, r *http.Request, userId int64, alertCode string) {
	// Return address (pre-login) was saved as a cookie.  Return the user to that address,
	// so they can continue what they were doing before logging in.
	returnAddress := GetAndDecodeCookie(r, kLoginReturnAddress, "/news/")

	prVal("returnAddress", returnAddress)

	returnAddress = insertUrlParam(returnAddress, kAlertCode, alertCode)

	prVal("injected returnAddress", returnAddress)

	http.Redirect(w, r, returnAddress, http.StatusSeeOther)
	return
}

///////////////////////////////////////////////////////////////////////////////
//
// login
//
///////////////////////////////////////////////////////////////////////////////
func loginHandler(w http.ResponseWriter, r *http.Request) {
	pr("loginHandler")

	prVal("r.Method", r.Method)

	userId := GetSession(w, r)
	//assert(userId == -1) // So what if the user is logged in when they get here?

	form := makeLoginForm()

	prVal("form", form)

	if r.Method == "POST" && form.validateData(r) { // On POST, validates and captures the request data.
		prVal("form", form)

		emailOrUsername := form.val(kEmailOrUsername)

		isEmail := strings.Contains(emailOrUsername, "@")

		if !flags.requirePassword && !isEmail {
			form.setFieldError(kEmailOrUsername, "Email address required to log in.")
		} else {
			var rows *sql.Rows
			// If not reqiring password, only email can be used to log in.
			// If requiring password, email or username can be used to log in.
			// Use case-insensitive compares.
			queryEmail := (isEmail || !flags.requirePassword);
			if queryEmail {
				rows = DbQuery("SELECT Id, PasswordHash[1], PasswordHash[2], PasswordHash[3], PasswordHash[4] " +
								"FROM $$User WHERE LOWER(Email) = LOWER($1);",
								emailOrUsername)
			} else {
				rows = DbQuery("SELECT Id, PasswordHash[1], PasswordHash[2], PasswordHash[3], PasswordHash[4] " +
								"FROM $$User WHERE LOWER(Username) = LOWER($1);",
								emailOrUsername)
			}

			var passwordHashInts int256

			defer rows.Close()
			if !rows.Next() {
				if queryEmail {
					form.setFieldError(kEmailOrUsername, "That email does not exist. Do you need to register?")
				} else {
					form.setFieldError(kEmailOrUsername, "That username does not exist. Do you need to register?")
				}
			} else {
				err := rows.Scan(&userId, &passwordHashInts[0], &passwordHashInts[1], &passwordHashInts[2], &passwordHashInts[3]);
				check(err)
				check(rows.Err())
				prf("User found! - id: '%d' passwordHashInts: %#v\n", userId, passwordHashInts)

				// TODO: if we ever re-enable flags.requirePassword, we'll need to check if stored password is all 0's and make them enter a new password, or else let them login password-free.
				passwordHash := int256{}
				if flags.requirePassword {
					passwordHash = GetPasswordHash256(form.val(kPassword))
				}
				if flags.requirePassword && (
						passwordHash[0] != passwordHashInts[0] ||
						passwordHash[1] != passwordHashInts[1] ||
						passwordHash[2] != passwordHashInts[2] ||
						passwordHash[3] != passwordHashInts[3]) {
					form.setFieldError(kPassword, "Invalid password.  Forgot password?")
				} else {
					CreateSession(w, r, userId)

					gotoReturnAddress(w, r, userId, kLoggedIn)

					return
				}
			}
		}
	}

	// handle GET, or invalid form data from POST...
	executeTemplate(w, kLogin, makeFormFrameArgs(r, form, "Log In"))
}

///////////////////////////////////////////////////////////////////////////////
//
// logout
//
///////////////////////////////////////////////////////////////////////////////
func logoutHandler(w http.ResponseWriter, r *http.Request) {
	DestroySession(w, r)

	http.Redirect(w, r, "/news", http.StatusSeeOther)
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
	// TODO: check that the user is not already logged in, do something appropriate.

	form := makeRegisterForm()

	// Username cannot contain ' '.
	form.field(kUsername).addFnValidator(
		func(username string) (bool, string) {
			if strings.Contains(username, " ") {
				return false, "Username cannot contain any spaces."
			}
			return true, ""
		})

	// Validate the password is complex enough.
	if flags.requirePassword {
		form.field(kPassword).addFnValidator(
			func(pw string) (bool, string) {
				password := MakePassword(pw)
				password.ProcessPassword()
				if password.CommonPassword {
					return false, "Your password is a common password.  Try making it harder to guess."
				}
				if password.Score < 2 {
					return false, "Your password is too simple.  Try adding lower and uppercase characters, numbers, and/or special characters."
				}
				return true, ""
			})
	}

	form.field(kUsername).addFnValidator(
		func(username string) (bool, string) {
			if strings.Contains(form.val(kEmail), username) {
				return false, "Username cannot be contained in the email."
			}
			return true, ""
		})

	if r.Method == "POST" && form.validateData(r) { // Handle POST, with valid data...
		if flags.requirePassword &&
				form.val(kPassword) != form.val(kConfirmPassword) { // Check for mismatched passwords
			pr("Passwords don't match")
			form.setFieldError(kConfirmPassword, "Passwords don't match")
		} else if DbExists("SELECT * FROM $$User WHERE LOWER(Email) = LOWER($1);", form.val(kEmail)) {       // Check for duplicate email
			pr("That email is taken... have you registered already?")
			form.setFieldError(kEmail, "That email is taken... have you registered already?")
        } else if DbExists("SELECT * FROM $$User WHERE LOWER(Username) = LOWER($1);", form.val(kUsername)) { // Check for duplicate username
			pr("That username is taken... try another one.  Or, have you registered already?")
			form.setFieldError(kUsername, "That username is taken... try another one.  Or, have you registered already?")
		} else {
			// Use a hashed password for security.
			passwordHashInts := int256{}
			if flags.requirePassword {
				passwordHashInts = GetPasswordHash256(form.val(kPassword))
			}

			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			check(err)
			prVal("ip", ip)

			// Add new user to the database
			prf("passwordHashInts[0]: %T %#v\n", passwordHashInts[0], passwordHashInts[0])
			userId := DbInsert(
				"INSERT INTO $$User(Email, Username, PasswordHash, Ip) " +
				"VALUES ($1, $2, ARRAY[$3::bigint, $4::bigint, $5::bigint, $6::bigint], $7) returning id;",
				form.val(kEmail),
				form.val(kUsername),
				passwordHashInts[0],
				passwordHashInts[1],
				passwordHashInts[2],
				passwordHashInts[3],
				ip)

			sendAccountConfirmationEmail(form.val(kEmail), form.val(kUsername))

			// Create session (encrypted userId).
			CreateSession(w, r, userId)

			http.Redirect(w, r, "/registerDetails", http.StatusSeeOther)
			return
		}
	}

	// handle GET, or invalid form data from POST...
	executeTemplate(w, kRegister, makeFormFrameArgs(r, form, "Sign Up"))
}


///////////////////////////////////////////////////////////////////////////////
//
// import and export subscribers
//
///////////////////////////////////////////////////////////////////////////////
func exportSubsHandler(w http.ResponseWriter, r *http.Request){
	// TODO: assert(userId == 5)

	pr("exportSubsHandler")

	assert( GetSession(w, r) == 5)

	tr := func(s string) string { return "<tr>" + s + "</tr>" }
	td := func(s string) string { return "<td>" + s + "</td>" }

	table := "<table>"
	table = table + tr(td("email") + td("name") + td("first name") + td("last name"))
	DoQuery(
		func(rows *sql.Rows) {
			var email, name string

			err := rows.Scan(&email, &name)
			check(err)

			names := strings.Split(name, " ")

			prVal("name", name)
			prVal("names", names)

			var firstName, lastName string

			if len(names) > 0 {
				firstName = names[0]
				lastName = names[len(names)-1]
			}

			table = table + tr(td(email) + td(name) + td(firstName) + td(lastName))

		},
		//"SELECT Email, COALESCE(Name, '') FROM $$User")
		"SELECT Email, COALESCE(Name, '') FROM $$User WHERE NOT FakeEmail")
	table = table + "</table>"

	serveHtml(w, table)
}

func importSubsHandler(w http.ResponseWriter, r *http.Request){

}



///////////////////////////////////////////////////////////////////////////////
//
// register details about the user
//
///////////////////////////////////////////////////////////////////////////////
func registerDetailsHandler(w http.ResponseWriter, r *http.Request){
	//RefreshSession(w, r)

	const (
		//kName = "name"
		//kZipCode = "zipCode"
		kBirthYear = "birthYear"
		//kCountry = "country"
		kGender = "gender"
		kParty = "party"
		kRace = "race"
		//kMaritalStatus = "maritalStatus"
		kSchoolCompleted = "schoolCompleted"
	)

	// Make sure all fields are skippable, since all this info is optional, and we don't want the login process to frustrate users.
	form := makeForm(
		//nuTextField(kName, "Enter Full Name", 50, 0, 100, "full name").noSpellCheck(),
		//nuTextField(kZipCode, "Enter Zip Code", 5, 0, 10, "zip code"),
		nuTextField(kBirthYear, "Enter Birth Year", 4, 0, 4, "birth year"),
		//nuSelectField(kCountry, "Select Country", countries, true, false, true, true, "Please select your country"),
		// nuOtherField(kCountry, "Enter Country", 50, 0, 100, "country"),
		nuSelectField(kGender, "Select Gender", genders, true, false, true, true, "Please select your gender"),
		 nuOtherField(kGender, "Enter Gender", 50, 0, 100, "gender"),
		nuSelectField(kParty, "Select Party", parties, true, false, true, true, "Please select your party"),
		 nuOtherField(kParty, "Enter Party", 50, 0, 100, "party"),
		nuSelectField(kRace, "Select Race", races, true, false, true, true, "Please select your race"),
		 nuOtherField(kRace, "Enter Race", 50, 0, 100, "race"),
		//nuSelectField(kMaritalStatus, "Select Marital Status", maritalStatuses, true, false, true, true, "Please select your marital status"),
		// nuOtherField(kMaritalStatus, "Enter Marital Status", 50, 0, 100, "marital status"),
		nuSelectField(kSchoolCompleted, "Select Furthest Schooling", schoolDegrees, true, false, true, true, "Please select your furthest schooling"),
		 nuOtherField(kSchoolCompleted, "Enter Furthest Schooling", 50, 0, 100, "furthest schooling"),
	)

	//form.field(kName).addRegexValidator(`^[\p{L}]+( [\p{L}]+)+$`, "Enter a valid full name (i.e. 'John Doe').")  // No validation since we are letting them skip fields.
	//form.field(kZipCode).addRegexValidator(`^\d{5}(?:[-\s]\d{4})?$`, "Invalid zip code")  // TODO: different countries have different zip code formats.
	form.field(kBirthYear).addFnValidator(
		func(input string) (bool, string) {
			year, err := strconv.Atoi(input)
			if err != nil {
				return true, "" // If the input is blank, just let the user skip this.

				//return false, "Please enter the year you were born."
			}
			currentYear := time.Now().Year()
			age := currentYear - year // true age would be either this expression, or this minus 1
			if age < 0 || age > 200 {
				return false, "Please enter the year you were born."
			} else {
				return true, ""
			}
		})

	userId := GetSession(w, r)
	if userId == -1 { // Secure cookie not found.  Either session expired, or someone is hacking.
		// So go to the register page.
		pr("secure cookie not found")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	bSkip := r.FormValue("skip_button") != ""
	prVal("bSkip", bSkip)

	prVal("form", form)

	if r.Method == "POST" && (	 // If this is handling form POST data and...
		bSkip ||                 //   the user chose SKIP, or
		form.validateData(r)) {  //   the data is valid...

		if !bSkip {
			prVal("userId", userId)

			pr("Updating vote info")

			// Update the user record with registration details.
			DbQuery(
				`UPDATE $$User
					SET (BirthYear, Gender, Party, Race, Schooling,
					     OtherGender, OtherParty, OtherRace, OtherSchoolCompleted)
					= ($2, $3, $4, $5, $6, $7, $8, $9, $10)
					WHERE Id = $1::bigint`,
				userId,
				//form.val(kName),
				//form.val(kCountry),
				//form.val(kZipCode),
				form.intVal(kBirthYear, 0),
				form.val(kGender),
				form.val(kParty),
				form.val(kRace),  // TODO: I think this should multi-select input, with a comma-delimited join of races here.
				//form.val(kMaritalStatus),
				form.val(kSchoolCompleted),
				form.otherVal(kGender),
				form.otherVal(kParty),
				form.otherVal(kRace),
				//form.otherVal(kCountry),
				//form.otherVal(kMaritalStatus),
				form.otherVal(kSchoolCompleted))
		} else {
			pr("Skipping vote info")
		}

		gotoReturnAddress(w, r, userId, kWelcomeToVotezilla)
		return
	}

	executeTemplate(w, kRegisterDetails, makeFormFrameArgs(r, form, "Voter Info"))
}

///////////////////////////////////////////////////////////////////////////////
//
// email preference
//
///////////////////////////////////////////////////////////////////////////////
func emailPreferenceHandler(w http.ResponseWriter, r *http.Request){
	form := makeForm(
		nuSelectField(kEmailPreference, "Select Email Preference", emailPref, true, false, false, false, "Please select your email preference"),
	)

	userId := GetSession(w, r)
	if userId == -1 { // Secure cookie not found.  Either session expired, or someone is hacking.
		// So go to the register page.
		pr("secure cookie not found")
		http.Redirect(w, r, "/register", http.StatusSeeOther)
		return
	}

	// Get the current value
	rows := DbQuery(`SELECT COALESCE(EmailPreference, '') FROM $$User WHERE Id = $1::bigint`, userId)
	if rows.Next() {
		check(rows.Scan(&form.field(kEmailPreference).Value))
	}
	check(rows.Err())
	rows.Close()

	// On post, set value
	if r.Method == "POST" && form.validateData(r) {
		DbQuery(
			`UPDATE $$User SET EmailPreference = $2 WHERE Id = $1::bigint`,
			userId,
			form.val(kEmailPreference),
		)

		http.Redirect(w, r, "/history/?alertCode=" + kPreferencesSaved, http.StatusSeeOther)
	}

	executeTemplate(w, kEmailPreference, makeFormFrameArgs(r, form, "Email Preference"))
}

///////////////////////////////////////////////////////////////////////////////
//
// update password - not tested, but could work in theory.  Passwords are not currently in use.
//
///////////////////////////////////////////////////////////////////////////////
func updatePasswordHandler(w http.ResponseWriter, r *http.Request){
	userId, err := strconv.ParseInt(r.FormValue("userId"), 10, 64)
	check(err)

	// TODO: implement updatePassword form, and get actual password from there.
	passwordHashInts := GetPasswordHash256("#NewPassword1234")

	// Update user's password in the database.
	prf("passwordHashInts[0]: %T %#v\n", passwordHashInts[0], passwordHashInts[0])
	DbQuery(`
		UPDATE $$User
		SET PasswordHash = ARRAY[$2::bigint, $3::bigint, $4::bigint, $5::bigint]
		WHERE Id = $1::bigint`,
		userId,
		passwordHashInts[0],
		passwordHashInts[1],
		passwordHashInts[2],
		passwordHashInts[3])

	// TODO: Send password update confirmation email

	// Create session (encrypted userId).
	CreateSession(w, r, userId)

	prf("Updated password for user %d", userId)

	serveHtml(w, "<h2>You successfully updated your password!</h2>")
}

///////////////////////////////////////////////////////////////////////////////
//
// login / signup
//
///////////////////////////////////////////////////////////////////////////////
func loginSignupHandler(w http.ResponseWriter, r *http.Request){
	args := struct {
		PageArgs
		Reason		string
	}{
		PageArgs:	makePageArgs(r, "Login / Signup", "", ""),
		Reason:		parseUrlParam(r, "reason"),
	}

	executeTemplate(w, kLoginSignup, args)
}

///////////////////////////////////////////////////////////////////////////////
//
// login via Facebook
//
///////////////////////////////////////////////////////////////////////////////
func loginFBHandler(w http.ResponseWriter, r *http.Request){
	args := struct {
		PageArgs
	}{
		PageArgs:	makePageArgs(r, "Login with Facebook", "", ""),
	}

	executeTemplate(w, kLoginFB, args)
}
