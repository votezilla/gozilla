// forms.go
package main

import (
    "github.com/votezilla/gforms"
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
    username = gforms.NewTextField(
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
            gforms.FullNameValidator(),
        },
        gforms.TextInputWidget(map[string]string{
            "autocorrect": "off",
            "spellcheck": "false",
        }),
    )
    birthYear = gforms.NewTextField( //NewFloatField(
        "year of birth",
        gforms.Validators{
            gforms.Required(),
            gforms.MinLengthValidator(4),
            gforms.MaxLengthValidator(4),
    })
    country = gforms.NewTextField(
        "country",
        gforms.Validators{
            gforms.Required(),
        },
        gforms.SelectWidgetEasy(countries),
    )
    location = gforms.NewTextField(
        "location",
        gforms.Validators{
            gforms.Required(),
            gforms.MaxLengthValidator(60),
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
    Email                   string `gforms:"email"`
    Password                string `gforms:"password"`
    RememberMe              string `gforms:"remember me"`
}

type RegisterData struct {
    Email                	string `gforms:"email"`
    Username				string `gforms:"username"`
    Password                string `gforms:"password"`
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
        email,
        password,
        rememberMe,
    ))
    RegisterForm = gforms.DefineForm(gforms.NewFields(
        email,
        username,
        password,
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

type FormArgs struct{
	Forms			[]TableForm
	Title			string
	Congrats		string
	Introduction	string
	Footer			string
	Script			string
}
