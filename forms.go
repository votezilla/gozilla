// forms.go
package main

import (
	"errors"
	"fmt"
    "github.com/votezilla/gforms"
    "net/http"
    "strconv"
	"strings"
	"time"
)

// ================================================================================
//
// -------------------------------- struct Field ----------------------------------
//
// ================================================================================
type Field struct {
	Name			string
	Type			string
	Value			string
	Label			string 
	Placeholder		string
	Error			string
	InputLength		int
	Html			func() string // Closure that outputs the html of this field
	HtmlRow			func() string // Closure that outputs the html of this field's entire table row
	
	Validators	[]func(Field)(bool, string) 
}

func (f Field) validate() bool {
	for _, validator := range f.Validators {
		isValid, errorMsg := validator(f)
		if !isValid {
			// Note: Just return the first error, don't accumulate them.
			f.Error = errorMsg
			return false
		}
	}
	return true
}

func (f Field) intVal() int {
	val, err := strconv.Atoi(f.Value)
	return ternary_int(err != nil, 0, val)
}

func (f Field) boolVal() bool {
	return f.Value != ""
}

func (f Field) getErrorHtml() string {
	return ternary_str(f.Error != "", fmt.Sprintf("<label class=\"error\">%s</label>", f.Error), "")
}

func requiredValidator() func(Field)(bool, string) {
	return func(f Field)(bool, string) {
		if f.Value != "" {
			return true, ""
		} else {
			return false, "This field is required."
		}
	}
}

func minMaxLengthValidator(minLength, maxLength int) func(Field)(bool, string) {
	return func(f Field)(bool, string) {
		length := len(f.Value)
			
		if length < minLength {
			return false, fmt.Sprintf("Ensure this value has at least %v characters", minLength)
		} else if length > maxLength {
			return false, fmt.Sprintf("Ensure this value has at most %v characters", maxLength)
		} else {
			return true, ""
		}	
	}
}

func optionValidator(validOptions []string) func(Field)(bool, string) {
	return func(f Field)(bool, string) {
		for _, validOption := range validOptions {
	        if f.Value == validOption {
	            return true, ""
	        }
	    }
	    return false, fmt.Sprintf("Invalid option selected")
	}
}


// ================================================================================
//
// -------------------------------- struct Form -----------------------------------
//
// ================================================================================
type Form map[string]Field

func (f Form) processData(r *http.Request) {
	for name, field := range f {
		field.Value = r.FormValue(name)
	}
}

func (f Form) validate() bool {
	valid := true
	for _, field := range f {
		valid = valid && field.validate()
	}
	return valid
}

func (f Form) validateData(r *http.Request) bool {
	f.processData(r)
	return f.validate()
}

func makeForm(fields ...Field) Form {
	f := make(map[string]Field)
	for _, field := range(fields) {
		f[field.Name] = field
	}
	return f
}

func makeTextField(name, label, placeholder string, inputLength, minLength, maxLength int) Field {
	f := Field{Name: name, Type: "text", Label: label, Placeholder: placeholder, InputLength: inputLength}
	
	if minLength > 0 {
		f.Validators = append(f.Validators, requiredValidator())
	}

	if minLength > 0 || maxLength != -1 {
		f.Validators = append(f.Validators, minMaxLengthValidator(minLength, maxLength))
	}
	
	// TODO: HTML-escape this
	f.Html = func()string {
		return fmt.Sprintf(
			"<input type=%s name=\"%s\" value=\"%s\" placeholder=\"%s\" length=\"%d\">%s",
			f.Type,
			f.Name,
			f.Value,
			f.Placeholder, 
			f.InputLength,
			f.getErrorHtml())
	}
	f.HtmlRow = func()string { return fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>", f.Label, f.Html()) }
	
	//TODO: [] add RowHtml function (which includes Label == Placeholder parameter)
	
	return f
}

func makeBoolField(name, label, optionText string, defaultValue bool) Field {
	// Hack: using Placeholder to hold optionText value
	f := Field{Name: name, Type: "checkbox", Label: label, Placeholder: optionText, Value: ternary_str(defaultValue, "1", "")}
	
	// TODO: HTML-escape this
	f.Html = func()string {
		return fmt.Sprintf(
			"<input type=checkbox name=\"%s\" value=\"1\" %s>%s",
			f.Name,
			ternary_str(f.boolVal(), "checked", ""),
			f.getErrorHtml())
	}
	f.HtmlRow = func()string { return fmt.Sprintf("<tr><td>%s</td><td>%s %s</td></tr>", f.Label, f.Html(), f.Placeholder) }	
	
	return f
}

func makeSelectField(name, label string, optionKeyValues [][2]string, startAtNil, required bool) Field {
	f := Field{Name: name, Type: "select", Label: label}
	
	if required {
		f.Validators = append(f.Validators, requiredValidator())
	}
	
	validOptions := make([]string, len(optionKeyValues) + 1)
	
	for _, optionKeyValue := range optionKeyValues {
		validOptions = append(validOptions, optionKeyValue[0]) // add the key
	}
	
	f.Validators = append(f.Validators, optionValidator(validOptions))
		
	// TODO: HTML-escape this
	f.Html = func()string {
		str := fmt.Sprintf("<select name=\"%s\">", f.Name)
		
		if startAtNil {
			str += "<option value=\"\">-</option>"
		}
		
		for _, optionKeyValue := range optionKeyValues {
			str += fmt.Sprintf("<option value=\"%s\" %s>%s</option>",
				optionKeyValue[0], // key
				ternary_str(f.Value == optionKeyValue[0], "checked", ""),
				optionKeyValue[1]) // value
		}			
		str += "</select>";
		str += f.getErrorHtml()
		return str
	}
	f.HtmlRow = func()string { return fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>", f.Label, f.Html()) }		
	
	return f
}



// === FIELDS ===
var (
	// NEW:

	
	
	// vv OLD!!! vv
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
            gforms.MinLengthValidator(4),
            gforms.MaxLengthValidator(50),
        	gforms.RegexpValidator(`^[^@]+$`, "Username cannot contain the '@' symbol."),
			gforms.FnValidator(func(fi *gforms.FieldInstance, fo *gforms.FormInstance) error {
				if strings.Contains(fo.Data["email"].RawStr, fo.Data["username"].RawStr) {
					return errors.New("Username cannot be contained in the email.")
				}
    			return nil
			}),
        },
        gforms.TextInputWidget(map[string]string{
            "autocorrect": "off",
            "spellcheck": "false",
            "autocapitalize": "off",
        }),
    )
    password = gforms.NewTextField( // TODO: get rid of validators for entry form
        "password",
        gforms.Validators{
            gforms.Required(),
        },
		gforms.PasswordInputWidget(map[string]string{}),
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
    createPassword = gforms.NewTextField( // TODO: get rid of validators for entry form
		"password",
		gforms.Validators{
			gforms.Required(),
			gforms.MinLengthValidator(8),
			gforms.MaxLengthValidator(40),
			gforms.PasswordStrengthValidator(1), // Require at least a level 1(weak) password.  So people don't get frustrated trying to create/remember a strong one.
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
    rememberMe = gforms.NewBooleanField(
		"remember me",
        gforms.Validators{},
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
				prVal(fo_, "fo.Data", fo.Data)
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
    
    // Submit post
    title = gforms.NewTextField(
        "title",
        gforms.Validators{
            gforms.Required(),
            gforms.MaxLengthValidator(50),
        },
    )    
	link = gforms.NewTextField(
		"link",
		gforms.Validators{
			gforms.Required(),
			gforms.URLValidator(),
			gforms.MaxLengthValidator(250),
		},
    )    
	category = gforms.NewTextField(
		"category",
		gforms.Validators{
            gforms.Required(),
        },
        gforms.SelectWidgetEasy(
			func() [][2]string {
				categories := make([][2]string, len(newsCategoryInfo.CategoryOrder))
				for i, category := range newsCategoryInfo.CategoryOrder {
					categories[i] = [2]string{category, category}
				}
				return categories
			}(),
		),
    )        
	thumbnail = gforms.NewTextField(
		"thumbnail",
		gforms.Validators{
            gforms.Required(),
        },
        gforms.HiddenInputWidget(map[string]string{}),
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

type SubmitLinkData struct {
	Link					string `gforms:"link"`
	Title					string `gforms:"title"`
	Category				string `gforms:"category"`
	Thumbnail				string `gforms:"thumbnail"` // Created with HTML in submitLink, since it's a hidden field.
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
        createPassword,
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
    
    SubmitLinkForm = gforms.DefineForm(gforms.NewFields(
		link,
		title,
		category,
		thumbnail,
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
