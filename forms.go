// forms.go
package main

import (
	"errors"
    "github.com/votezilla/gforms"
	"strconv"
	"time"
)

// === FIELDS ===
var (
	// Login data
    email = gforms.NewTextField(
        "email",
        gforms.Validators{
			gforms.EmailValidator(),
            gforms.Required(),
            gforms.MaxLengthValidator(345),
        },
        gforms.TextInputWidget(map[string]string{
            "autocorrect": "off",
            "spellcheck": "false",
            "autocapitalize": "off",
        }),
    )
    username = gforms.NewTextField( // TODO: validate the username does not contain the '@' symbol, and is not a substring of the email.
        "username",
        gforms.Validators{
            gforms.Required(),
            gforms.MaxLengthValidator(50),
        },
        gforms.TextInputWidget(map[string]string{
            "autocorrect": "off",
            "spellcheck": "false",
            "autocapitalize": "off",
        }),
    )
    emailOrUsername = gforms.NewTextField(
		"email or username",
		gforms.Validators{
			gforms.Required(),
			gforms.MaxLengthValidator(345),
		},
		gforms.TextInputWidget(map[string]string{
			"autocorrect": "off",
			"spellcheck": "false",
			"autocapitalize": "off",
		}),
    )
    password = gforms.NewTextField(
        "password",
        gforms.Validators{
            gforms.Required(),
            gforms.MinLengthValidator(8),
            gforms.MaxLengthValidator(40),
            gforms.PasswordStrengthValidator(3), // Require strong password.
        },
    )
    // Not currently used.  Keep code in case I decide to re-enable later.
    //confirmPassword = gforms.NewTextField(
    //    "confirm password",
    //    gforms.Validators{
    //        gforms.FieldMatchValidator("password"),
    //    },
    //    gforms.PasswordInputWidget(map[string]string{}),
    //)
    rememberMe = gforms.NewTextField(
        "remember me",
        gforms.Validators{},
        gforms.CheckboxMultipleWidget(
            map[string]string{},
            func() gforms.CheckboxOptions { return gforms.StringCheckboxOptions([][]string{
                {"Stay logged in (uncheck if a shared computer)", "R", "false", "false"},
            })},
        ),
    )    
    
    // Demographics
    name = gforms.NewTextField(
        "full name",
        gforms.Validators{
            gforms.Required(),
            gforms.MaxLengthValidator(100),
            gforms.RegexpValidator(`^[\p{L}]+( [\p{L}]+)+$`, "Enter a valid full name (i.e. 'John Doe')."),
        },
        gforms.TextInputWidget(map[string]string{
            "autocorrect": "off",
            "spellcheck": "false",
        }),
    )
    birthYear = gforms.NewTextField( //TODO: validate date
        "year of birth",
        gforms.Validators{
            gforms.Required(),
            gforms.MinLengthValidator(4),
            gforms.MaxLengthValidator(4),
            gforms.FnValidator(func(fi *gforms.FieldInstance, fo *gforms.FormInstance) error {
				printVal(`fo.Data`, fo.Data)
				year, err := strconv.Atoi(fo.Data["year of birth"].RawStr)
				if err != nil {
					return errors.New("Please enter a valid year.")
				}
				currentYear := time.Now().Year()
				age := currentYear - year // true age would be either this expression, or this minus 1
				if age < 0 || age > 200 {
					return errors.New("Please enter the year you were born.")
				} else {
					return nil
				}
			}),
    })
    country = gforms.NewTextField(
        "country",
        gforms.Validators{
            gforms.Required(),
        },
        gforms.SelectWidgetEasy(countries),
    )
    location = gforms.NewTextField( // TODO: validate countries with a state to have ',', add JS to set location to US by default... eventually base it on the user's IP address.
        "location",
        gforms.Validators{
            gforms.Required(),
            gforms.MaxLengthValidator(60),
            gforms.FnValidator(func(fi *gforms.FieldInstance, fo *gforms.FormInstance) error {
				printVal("fo.Data", fo.Data)
				if fo.Data["country"].RawStr == "US" {
					rvl := gforms.RegexpValidator(`^\d{5}(?:[-\s]\d{4})?$`, "Invalid zip code")
					return rvl.Validate(fi, fo)
				} else {
					return nil // Only validating US zip codes for now
				}
			}),
    })
    gender = gforms.NewTextField(
        "gender",
        gforms.Validators{
            gforms.Required(),
        },
        gforms.SelectWidgetEasy([][2]string{
			{"-",      ""},
            {"Male",   "M"},
            {"Female", "F"},
            {"Other",  "O"},
		}),
    )
    party = gforms.NewTextField(
        "party",
        gforms.Validators{
            gforms.Required(),
        },
		gforms.SelectWidgetEasy([][2]string{
			{"-",           "" },
			{"Republican",  "R"},
			{"Democrat",    "D"},
			{"Independent", "I"},
			{"Other",       "O"},
		}),
    )
    race = gforms.NewMultipleTextField(
		"race / ethnicity",
        gforms.Validators{
            gforms.Required(),
        },
        gforms.CheckboxMultipleWidget(
            map[string]string{},
            func() gforms.CheckboxOptions { return gforms.StringCheckboxOptions([][]string{
                {"American Indian or Alaska Native",    "I", "false", "false"},
                {"Asian",                               "A", "false", "false"},
                {"Black or African American",           "B", "false", "false"},
                {"Hispanic, Latino, or Spanish",        "H", "false", "false"},
                {"Native Hawaiian or Pacific Islander", "P", "false", "false"},
                {"White",                               "W", "false", "false"},
                {"Other",                               "O", "false", "false"},
            })},
        ),
    )
    marital = gforms.NewTextField(
        "marital status",
        gforms.Validators{
            gforms.Required(),
        },
        gforms.SelectWidgetEasy([][2]string{
			{"-",                               "" },
			{"Single (Never Married)",          "S"},
			{"Divorced or Separated",           "D"},
			{"Widowed",                         "W"},
			{"Married or Domestic Partnership", "M"},
		}),
    )   
    schooling = gforms.NewTextField(
        "furthest schooling completed",
        gforms.Validators{
            gforms.Required(),
        },
        gforms.SelectWidgetEasy([][2]string{
			{"-",                                "" },
			{"Less than a high school diploma",  "L"},
			{"High school degree or equivalent", "H"},
			{"Some college, but no degree",      "S"},
			{"College graduate",                 "C"},
			{"Postgraduate study",               "P"},
		}),
    )
)

// === FORM POST DATA ===
type LoginData struct {
    EmailOrUsername         string `gforms:"email or username"`
    Password                string `gforms:"password"`
    RememberMe              bool   `gforms:"remember me"`
}

type RegisterData struct {
    Email                	string `gforms:"email"`
    Username				string `gforms:"username"`
    Password                string `gforms:"password"`
    RememberMe              bool   `gforms:"remember me"`
}

type RegisterDetailsData struct {
    Name                    string `gforms:"full name"`

    // location
    Country                 string `gforms:"country"`
    Location                string `gforms:"location"`

    // demographic
    BirthYear               string `gforms:"year of birth"`
    Gender                  string `gforms:"gender"`
    Party                   string `gforms:"party"`
    Races					[]string `gforms:"race / ethnicity"`
    Marital                 string `gforms:"marital status"`
    Schooling               string `gforms:"furthest schooling completed"`
}

// === FORMS ===
var (
    LoginForm = gforms.DefineForm(gforms.NewFields(
        emailOrUsername,
        password,
        rememberMe,
    ))
    RegisterForm = gforms.DefineForm(gforms.NewFields(
        email,
        username,
        password,
        rememberMe,
    ))
    RegisterDetailsForm = gforms.DefineForm(gforms.NewFields(
        // name
        name,
        
        // location
        country,
        location,

        // demographic
        birthYear,
        gender,
        party,
        race,
        marital,
        schooling,      
    ))
) // var

// === FORM TYPES ===
type TableForm struct {
	Form			*gforms.FormInstance
	CallToAction	string
	AdditionalError string
}

// Template arguments for form webpage template.
type FormArgs struct {
	PageArgs
	Forms			[]TableForm
	Congrats		string
	Introduction	string
	Footer			string
}
