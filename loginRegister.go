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

func makeLoginForm() *Form {
	return makeForm(
		nuTextField(kEmailOrUsername, "Email / Username", 50, 6, 345, "email / username").noSpellCheckOrCaps(),
		nuPasswordField(kPassword, "Password", 50, 8, 40),
	)
}

func makeRegisterForm() *Form {
	return makeForm(
		nuTextField(kEmail, "Email", 50, 6, 345, "email").noSpellCheckOrCaps().addFnValidator(emailValidator()),
		nuTextField(kUsername, "Pick a Username", 50, 4, 25, "username").noSpellCheckOrCapsOrAutocomplete(),
		nuPasswordField(kPassword, "Create Password", 40, 8, 40),
		nuPasswordField(kConfirmPassword, "Confirm Password", 40, 8, 40).noDefaultValidators(),
	)
}

func loginRegisterHandler(w http.ResponseWriter, r *http.Request) {
	pr("loginRegisterHandler")

	loginForm := makeLoginForm()
	registerForm := makeRegisterForm()

	executeTemplate(
		w,
		kLoginRegister,
		struct {
			PageArgs
			LoginForm		Form
			RegisterForm	Form
		} {
			PageArgs: 		makePageArgs(r, "Log In / Sign Up", "", ""),
			LoginForm:		*loginForm,
			RegisterForm:	*registerForm,
		},
	)
}
///////////////////////////////////////////////////////////////////////////////
//
// login
//
///////////////////////////////////////////////////////////////////////////////
func loginHandler(w http.ResponseWriter, r *http.Request) {
	pr("loginHandler")

	prVal("r.Method", r.Method)

	userId := GetSession(r)
	assert(userId == -1) // User must not be already logged in!  TODO: handle this case gracefully instead of crashing here!

	form := makeLoginForm()

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

	form.field(kUsername).addFnValidator(
		func(username string) (bool, string) {
			if strings.Contains(form.val(kEmail), username) {
				return false, "Username cannot be contained in the email."
			}
			return true, ""
		})

	if r.Method == "POST" && form.validateData(r) { // Handle POST, with valid data...
		if form.val(kPassword) != form.val(kConfirmPassword) { // Check for mismatched passwords
			pr("Passwords don't match")
			form.setFieldError(kConfirmPassword, "Passwords don't match")
		} else if DbExists("SELECT * FROM $$User WHERE Email = $1;", form.val(kEmail)) {       // Check for duplicate email
			pr("That email is taken... have you registered already?")
			form.setFieldError(kEmail, "That email is taken... have you registered already?")
        } else if DbExists("SELECT * FROM $$User WHERE Username = $1;", form.val(kUsername)) { // Check for duplicate username
			pr("That username is taken... try another one.  Or, have you registered already?")
			form.setFieldError(kUsername, "That username is taken... try another one.  Or, have you registered already?")
		} else {
			// Use a hashed password for security.
			passwordHashInts := GetPasswordHash256(form.val(kPassword))

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

			// Send confirmation email
			sendEmail(BUSINESS_EMAIL, form.val(kEmail), "Account Creation Confirmation", generateConfEmail(form.val(kUsername)))

			// Create session (encrypted userId).
			CreateSession(w, r, userId)//, form.boolVal(kRememberMe))

			http.Redirect(w, r, "/registerDetails", http.StatusSeeOther)
			return
		}
	}

	// handle GET, or invalid form data from POST...
	executeTemplate(w, kRegister, makeFormFrameArgs(r, form, "Sign Up"))
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
		nuTextField(kName, "Enter Full Name", 50, 0, 100, "full name").noSpellCheck(),
		nuTextField(kZipCode, "Enter Zip Code", 5, 0, 10, "zip code"),
		nuTextField(kBirthYear, "Enter Birth Year", 4, 0, 4, "birth year"),
		nuSelectField(kCountry, "Select Country", countries, true, true, true, true, "Please select your country"),
			nuOtherField(kCountry, "Enter Country", 50, 0, 100, "country"),
		nuSelectField(kGender, "Select Gender", genders, true, true, true, true, "Please select your gender"),
			nuOtherField(kGender, "Enter Gender", 50, 0, 100, "gender"),
		nuSelectField(kParty, "Select Party", parties, true, true, true, true, "Please select your party"),
			nuOtherField(kParty, "Enter Party", 50, 0, 100, "party"),
		nuSelectField(kRace, "Select Race", races, true, true, true, true, "Please select your race"),
			nuOtherField(kRace, "Enter Race", 50, 0, 100, "race"),
		nuSelectField(kMaritalStatus, "Select Marital Status", maritalStatuses, true, true, true, true, "Please select your marital status"),
			nuOtherField(kMaritalStatus, "Enter Marital Status", 50, 0, 100, "marital status"),
		nuSelectField(kSchoolCompleted, "Select Furthest Schooling", schoolDegrees, true, true, true, true, "Please select your furthest schooling"),
			nuOtherField(kSchoolCompleted, "Enter Furthest Schooling", 50, 0, 100, "furthest schooling"),
	)

	form.field(kName).addRegexValidator(`^[\p{L}]+( [\p{L}]+)+$`, "Enter a valid full name (i.e. 'John Doe').")
	form.field(kZipCode).addRegexValidator(`^\d{5}(?:[-\s]\d{4})?$`, "Invalid zip code")  // TODO: different countries have different zip code formats.
	form.field(kBirthYear).addFnValidator(
		func(input string) (bool, string) {
			year, err := strconv.Atoi(input)
			if err != nil {
				return false, "Please enter the year you were born."
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
					SET (Name, Country, Location, BirthYear, Gender, Party, Race, Marital, Schooling,
					     OtherGender, OtherParty, OtherRace, OtherCountry, OtherMaritalStatus, OtherSchoolCompleted)
					= ($2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
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
				form.val(kSchoolCompleted),
				form.otherVal(kGender),
				form.otherVal(kParty),
				form.otherVal(kRace),
				form.otherVal(kCountry),
				form.otherVal(kMaritalStatus),
				form.otherVal(kSchoolCompleted))
		} else {
			pr("Skipping vote info")
		}

		//serveHTML(w, `<h2>Congrats, you just registered</h2>
		//			  <script>alert('Congrats, you just registered')</script>`)
		// TODO: do registration as a pop-up.  Commenting out this for now, as it breaks the UserId cookie:
		http.Redirect(w, r, "/news?alert=Welcome to Votezilla!!!", http.StatusSeeOther)
		return
	}

	executeTemplate(w, kRegisterDetails, makeFormFrameArgs(r, form, "Voter Info"))
}

///////////////////////////////////////////////////////////////////////////////
//
// update password
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

	serveHTML(w, "<h2>You successfully updated your password!</h2>")
}
