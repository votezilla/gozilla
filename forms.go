// forms.go
package main

import (
    "github.com/votezilla/gforms"
)

// === FIELDS ===
var (
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
        gforms.SelectWidgetEasy([][2]string{
            // ISO-3166-1 Alpha-2 country list.  See: https://www.freeformatter.com/iso-country-list-html-select.html
			{"-", ""},
			{"United States", "US"},
			{"United States Minor Outlying Islands", "UM"},
			{"Afghanistan", "AF"},
			{"Åland Islands", "AX"},
			{"Albania", "AL"},
			{"Algeria", "DZ"},
			{"American Samoa", "AS"},
			{"Andorra", "AD"},
			{"Angola", "AO"},
			{"Anguilla", "AI"},
			{"Antarctica", "AQ"},
			{"Antigua and Barbuda", "AG"},
			{"Argentina", "AR"},
			{"Armenia", "AM"},
			{"Aruba", "AW"},
			{"Australia", "AU"},
			{"Austria", "AT"},
			{"Azerbaijan", "AZ"},
			{"Bahamas", "BS"},
			{"Bahrain", "BH"},
			{"Bangladesh", "BD"},
			{"Barbados", "BB"},
			{"Belarus", "BY"},
			{"Belgium", "BE"},
			{"Belize", "BZ"},
			{"Benin", "BJ"},
			{"Bermuda", "BM"},
			{"Bhutan", "BT"},
			{"Bolivia, Plurinational State of", "BO"},
			{"Bonaire, Sint Eustatius and Saba", "BQ"},
			{"Bosnia and Herzegovina", "BA"},
			{"Botswana", "BW"},
			{"Bouvet Island", "BV"},
			{"Brazil", "BR"},
			{"British Indian Ocean Territory", "IO"},
			{"Brunei Darussalam", "BN"},
			{"Bulgaria", "BG"},
			{"Burkina Faso", "BF"},
			{"Burundi", "BI"},
			{"Cambodia", "KH"},
			{"Cameroon", "CM"},
			{"Canada", "CA"},
			{"Cape Verde", "CV"},
			{"Cayman Islands", "KY"},
			{"Central African Republic", "CF"},
			{"Chad", "TD"},
			{"Chile", "CL"},
			{"China", "CN"},
			{"Christmas Island", "CX"},
			{"Cocos (Keeling) Islands", "CC"},
			{"Colombia", "CO"},
			{"Comoros", "KM"},
			{"Congo", "CG"},
			{"Congo, the Democratic Republic of the", "CD"},
			{"Cook Islands", "CK"},
			{"Costa Rica", "CR"},
			{"Côte d'Ivoire", "CI"},
			{"Croatia", "HR"},
			{"Cuba", "CU"},
			{"Curaçao", "CW"},
			{"Cyprus", "CY"},
			{"Czech Republic", "CZ"},
			{"Denmark", "DK"},
			{"Djibouti", "DJ"},
			{"Dominica", "DM"},
			{"Dominican Republic", "DO"},
			{"Ecuador", "EC"},
			{"Egypt", "EG"},
			{"El Salvador", "SV"},
			{"Equatorial Guinea", "GQ"},
			{"Eritrea", "ER"},
			{"Estonia", "EE"},
			{"Ethiopia", "ET"},
			{"Falkland Islands (Malvinas)", "FK"},
			{"Faroe Islands", "FO"},
			{"Fiji", "FJ"},
			{"Finland", "FI"},
			{"France", "FR"},
			{"French Guiana", "GF"},
			{"French Polynesia", "PF"},
			{"French Southern Territories", "TF"},
			{"Gabon", "GA"},
			{"Gambia", "GM"},
			{"Georgia", "GE"},
			{"Germany", "DE"},
			{"Ghana", "GH"},
			{"Gibraltar", "GI"},
			{"Greece", "GR"},
			{"Greenland", "GL"},
			{"Grenada", "GD"},
			{"Guadeloupe", "GP"},
			{"Guam", "GU"},
			{"Guatemala", "GT"},
			{"Guernsey", "GG"},
			{"Guinea", "GN"},
			{"Guinea-Bissau", "GW"},
			{"Guyana", "GY"},
			{"Haiti", "HT"},
			{"Heard Island and McDonald Islands", "HM"},
			{"Holy See (Vatican City State)", "VA"},
			{"Honduras", "HN"},
			{"Hong Kong", "HK"},
			{"Hungary", "HU"},
			{"Iceland", "IS"},
			{"India", "IN"},
			{"Indonesia", "ID"},
			{"Iran, Islamic Republic of", "IR"},
			{"Iraq", "IQ"},
			{"Ireland", "IE"},
			{"Isle of Man", "IM"},
			{"Israel", "IL"},
			{"Italy", "IT"},
			{"Jamaica", "JM"},
			{"Japan", "JP"},
			{"Jersey", "JE"},
			{"Jordan", "JO"},
			{"Kazakhstan", "KZ"},
			{"Kenya", "KE"},
			{"Kiribati", "KI"},
			{"Korea, Democratic People's Republic of", "KP"},
			{"Korea, Republic of", "KR"},
			{"Kuwait", "KW"},
			{"Kyrgyzstan", "KG"},
			{"Lao People's Democratic Republic", "LA"},
			{"Latvia", "LV"},
			{"Lebanon", "LB"},
			{"Lesotho", "LS"},
			{"Liberia", "LR"},
			{"Libya", "LY"},
			{"Liechtenstein", "LI"},
			{"Lithuania", "LT"},
			{"Luxembourg", "LU"},
			{"Macao", "MO"},
			{"Macedonia, the former Yugoslav Republic of", "MK"},
			{"Madagascar", "MG"},
			{"Malawi", "MW"},
			{"Malaysia", "MY"},
			{"Maldives", "MV"},
			{"Mali", "ML"},
			{"Malta", "MT"},
			{"Marshall Islands", "MH"},
			{"Martinique", "MQ"},
			{"Mauritania", "MR"},
			{"Mauritius", "MU"},
			{"Mayotte", "YT"},
			{"Mexico", "MX"},
			{"Micronesia, Federated States of", "FM"},
			{"Moldova, Republic of", "MD"},
			{"Monaco", "MC"},
			{"Mongolia", "MN"},
			{"Montenegro", "ME"},
			{"Montserrat", "MS"},
			{"Morocco", "MA"},
			{"Mozambique", "MZ"},
			{"Myanmar", "MM"},
			{"Namibia", "NA"},
			{"Nauru", "NR"},
			{"Nepal", "NP"},
			{"Netherlands", "NL"},
			{"New Caledonia", "NC"},
			{"New Zealand", "NZ"},
			{"Nicaragua", "NI"},
			{"Niger", "NE"},
			{"Nigeria", "NG"},
			{"Niue", "NU"},
			{"Norfolk Island", "NF"},
			{"Northern Mariana Islands", "MP"},
			{"Norway", "NO"},
			{"Oman", "OM"},
			{"Pakistan", "PK"},
			{"Palau", "PW"},
			{"Palestinian Territory, Occupied", "PS"},
			{"Panama", "PA"},
			{"Papua New Guinea", "PG"},
			{"Paraguay", "PY"},
			{"Peru", "PE"},
			{"Philippines", "PH"},
			{"Pitcairn", "PN"},
			{"Poland", "PL"},
			{"Portugal", "PT"},
			{"Puerto Rico", "PR"},
			{"Qatar", "QA"},
			{"Réunion", "RE"},
			{"Romania", "RO"},
			{"Russia", "RU"},
			{"Rwanda", "RW"},
			{"Saint Barthélemy", "BL"},
			{"Saint Helena, Ascension and Tristan da Cunha", "SH"},
			{"Saint Kitts and Nevis", "KN"},
			{"Saint Lucia", "LC"},
			{"Saint Martin (French part)", "MF"},
			{"Saint Pierre and Miquelon", "PM"},
			{"Saint Vincent and the Grenadines", "VC"},
			{"Samoa", "WS"},
			{"San Marino", "SM"},
			{"Sao Tome and Principe", "ST"},
			{"Saudi Arabia", "SA"},
			{"Senegal", "SN"},
			{"Serbia", "RS"},
			{"Seychelles", "SC"},
			{"Sierra Leone", "SL"},
			{"Singapore", "SG"},
			{"Sint Maarten (Dutch part)", "SX"},
			{"Slovakia", "SK"},
			{"Slovenia", "SI"},
			{"Solomon Islands", "SB"},
			{"Somalia", "SO"},
			{"South Africa", "ZA"},
			{"South Georgia and the South Sandwich Islands", "GS"},
			{"South Sudan", "SS"},
			{"Spain", "ES"},
			{"Sri Lanka", "LK"},
			{"Sudan", "SD"},
			{"Suriname", "SR"},
			{"Svalbard and Jan Mayen", "SJ"},
			{"Swaziland", "SZ"},
			{"Sweden", "SE"},
			{"Switzerland", "CH"},
			{"Syrian Arab Republic", "SY"},
			{"Taiwan, Province of China", "TW"},
			{"Tajikistan", "TJ"},
			{"Tanzania, United Republic of", "TZ"},
			{"Thailand", "TH"},
			{"Timor-Leste", "TL"},
			{"Togo", "TG"},
			{"Tokelau", "TK"},
			{"Tonga", "TO"},
			{"Trinidad and Tobago", "TT"},
			{"Tunisia", "TN"},
			{"Turkey", "TR"},
			{"Turkmenistan", "TM"},
			{"Turks and Caicos Islands", "TC"},
			{"Tuvalu", "TV"},
			{"Uganda", "UG"},
			{"Ukraine", "UA"},
			{"United Arab Emirates", "AE"},
			{"United Kingdom", "GB"},
			{"United States", "US"},
			{"United States Minor Outlying Islands", "UM"},
			{"Uruguay", "UY"},
			{"Uzbekistan", "UZ"},
			{"Vanuatu", "VU"},
			{"Venezuela", "VE"},
			{"Viet Nam", "VN"},
			{"Virgin Islands, British", "VG"},
			{"Virgin Islands, U.S.", "VI"},
			{"Wallis and Futuna", "WF"},
			{"Western Sahara", "EH"},
			{"Yemen", "YE"},
			{"Zambia", "ZM"},
			{"Zimbabwe", "ZW"},
        }),
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

// === COUNTRY DATA ===
var (
	// https://en.m.wikipedia.org/wiki/Federated_state
	CountriesWithStates = map[string]bool{"AE":true,"AR":true,"AT":true,"AU":true,"BA":true,"BE":true,"BR":true,"CA":true,"CH":true,"DE":true,"ET":true,"FM":true,"IN":true,"IQ":true,"KM":true,"KN":true,"MX":true,"MY":true,"NG":true,"NP":true,"PK":true,"RU":true,"SD":true,"SO":true,"SS":true,"US":true,"VE":true}
	
	// https://www.ups.com/worldshiphelp/WS16/ENU/AppHelp/Codes/Countries_Territories_Requiring_Postal_Codes.htm
	CountriesWithPostalCodes = map[string]bool{"A2":true,"AM":true,"AR":true,"AT":true,"AU":true,"AZ":true,"BA":true,"BD":true,"BE":true,"BG":true,"BN":true,"BR":true,"BY":true,"CA":true,"CH":true,"CN":true,"CS":true,"CY":true,"CZ":true,"DE":true,"DK":true,"DZ":true,"EE":true,"EN":true,"ES":true,"FI":true,"FO":true,"FR":true,"GB":true,"GE":true,"GG":true,"GL":true,"GR":true,"GU":true,"HO":true,"HR":true,"HU":true,"IC":true,"ID":true,"IL":true,"IN":true,"IT":true,"JE":true,"JP":true,"KG":true,"KO":true,"KR":true,"KZ":true,"LI":true,"LK":true,"LT":true,"LU":true,"LV":true,"M3":true,"ME":true,"MG":true,"MH":true,"MK":true,"MN":true,"MQ":true,"MX":true,"MY":true,"NB":true,"NL":true,"NO":true,"NT":true,"NZ":true,"PH":true,"PK":true,"PL":true,"PO":true,"PR":true,"PT":true,"RE":true,"RU":true,"SA":true,"SE":true,"SF":true,"SG":true,"SI":true,"SK":true,"SX":true,"TH":true,"TJ":true,"TM":true,"TN":true,"TR":true,"TU":true,"TW":true,"UA":true,"US":true,"UV":true,"UY":true,"UZ":true,"VA":true,"VI":true,"VL":true,"VN":true,"WL":true,"YA":true,"YT":true,"ZA":true}
)

// === FORM POST DATA ===
type LoginData struct {
    Email                   string `gforms:"email"`
    Password                string `gforms:"password"`
    RememberMe              string `gforms:"remember me"`
}

type RegisterData struct {
    Email                	string `gforms:"email"`
    Userame					string `gforms:"username"`
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
	Introduction	string
	Footer			string
	Script			string
}
