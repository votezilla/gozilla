// gozilla.go
package main

import (
	//"bytes"
	"database/sql"
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
)

///////////////////////////////////////////////////////////////////////////////
//
// login
//
///////////////////////////////////////////////////////////////////////////////
func loginHandler(w http.ResponseWriter, r *http.Request) {

	pr("loginHandler")

	prVal("r.Method", r.Method)

	userId := GetSession(r)
	assert(userId == -1) // User must not be already logged in!

	//bRememberMe := str_to_bool(GetCookie(r, kRememberMe, "false"))

	form := makeForm(
		nuTextField(kEmailOrUsername, "Email / Username", 50, 6, 345),
		nuPasswordField(kPassword, "Password", 50, 8, 40),
	)

	prVal("form", form)

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
				CreateSession(w, r, userId)//, true)

				//bRememberMe = form.boolVal(kRememberMe)
				//CreateSession(w, r, userId, bRememberMe)

				http.Redirect(w, r, "/news?alert=LoggedIn", http.StatusSeeOther)
				return
			}
		}
	}

	// handle GET, or invalid form data from POST...
	executeTemplate(w, kLogin, makeFormFrameArgs(form, "Log In"))
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
	form := makeForm(
		nuTextField(kEmail, "Email", 50, 6, 345),
		nuTextField(kUsername, "Pick a Username", 50, 4, 345).noSpellCheck(),
		nuTextField(kPassword, "Create Password", 40, 8, 40),
	//	nuBoolField(kRememberMe, "Remember Me", true).
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
				CreateSession(w, r, userId)//, form.boolVal(kRememberMe))

				http.Redirect(w, r, "/registerDetails", http.StatusSeeOther)
				return
			}
		}
	}

	// handle GET, or invalid form data from POST...
	executeTemplate(w, kRegister, makeFormFrameArgs(form, "Sign Up"))
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
		kZipCode = "zipCode"
		kBirthYear = "birthYear"
		kCountry = "country"
		kGender = "gender"
		kParty = "party"
		kRace = "race"
		kMaritalStatus = "maritalStatus"
		kSchoolCompleted = "schoolCompleted"
	)

	form := makeForm(
		nuTextField(kName, "Full Name", 50, 0, 100),
		nuTextField(kZipCode, "Zip Code", 5, 0, 10),
		nuTextField(kBirthYear, "Birth Year", 4, 0, 4),
		nuSelectField(kCountry, "Country", countries, true, false, true, true),
		nuSelectField(kGender, "Gender", genders, true, false, true, true),
		nuSelectField(kParty, "Party", parties, true, false, true, true),
		nuSelectField(kRace, "Race", races, true, false, true, true),
		nuSelectField(kMaritalStatus, "Marital Status", maritalStatuses, true, false, true, true),
		nuSelectField(kSchoolCompleted, "Furthest Schooling", schoolDegrees, true, false, true, true),
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

	pr("form(skip_button:")
	prVal("  ", r.FormValue("skip_button"))

	pr("form(submit_button:")
	prVal("  ", r.FormValue("submit_button"))

	bSkip := r.FormValue("skip_button") != ""

	prVal("bSkip", bSkip)

	if r.Method == "POST" && (	 // If this is handling form POST data and...
		bSkip ||                 //   the user chose SKIP, or
		form.validateData(r)) {  //   the data is valid...

		if !bSkip {
			prVal("userId", userId)

			pr("Updating vote info")

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
		} else {
			pr("Skipping vote info")
		}

		//serveHTML(w, `<h2>Congrats, you just registered</h2>
		//			  <script>alert('Congrats, you just registered')</script>`)
		// TODO: do registration as a pop-up.  Commenting out this for now, as it breaks the UserId cookie:
		http.Redirect(w, r, "/news?alert=Welcome to Votezilla!!!", http.StatusSeeOther)
		return
	}

	// handle GET, or invalid form data from POST...
/*	{
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
	*/

	executeTemplate(w, kRegisterDetails, makeFormFrameArgs(form, "Voter Info"))
}
