﻿// location.go
package main


var (
	genders = OptionData{
		{"M", "Male"},
		{"F", "Female"},
	}

	parties = OptionData{
		{"R", "Republican"},
		{"D", "Democrat"},
		{"I", "Independent"},
	}

	schoolDegrees = OptionData{
		{"L", "Less than a high school diploma"},
		{"H", "High school degree or equivalent"},
		{"S", "Some college, but no degree"},
		{"C", "College graduate"},
		{"P", "Postgraduate study"},
	}

	maritalStatuses = OptionData{
		{"S", "Single (Never Married)"},
		{"M", "Married or Domestic Partnership"},
		{"D", "Divorced or Separated"},
		{"W", "Widowed"},
	}

	races = OptionData{
		{"I", "American Indian or Alaska Native"},
		{"A", "Asian"},
		{"B", "Black or African American"},
		{"H", "Hispanic, Latino, or Spanish"},
		{"P", "Native Hawaiian or Pacific Islander"},
		{"W", "White"},
		{"O", "Other"},
	}

	// ISO-3166-1 Alpha-2 country list.  See: https://www.freeformatter.com/iso-country-list-html-select.html
	countries = OptionData{
		{"US", "United States"},
		{"UM", "United States Minor Outlying Islands"},
		{"AF", "Afghanistan"},
		{"AX", "Åland Islands"},
		{"AL", "Albania"},
		{"DZ", "Algeria"},
		{"AS", "American Samoa"},
		{"AD", "Andorra"},
		{"AO", "Angola"},
		{"AI", "Anguilla"},
		{"AQ", "Antarctica"},
		{"AG", "Antigua and Barbuda"},
		{"AR", "Argentina"},
		{"AM", "Armenia"},
		{"AW", "Aruba"},
		{"AU", "Australia"},
		{"AT", "Austria"},
		{"AZ", "Azerbaijan"},
		{"BS", "Bahamas"},
		{"BH", "Bahrain"},
		{"BD", "Bangladesh"},
		{"BB", "Barbados"},
		{"BY", "Belarus"},
		{"BE", "Belgium"},
		{"BZ", "Belize"},
		{"BJ", "Benin"},
		{"BM", "Bermuda"},
		{"BT", "Bhutan"},
		{"BO", "Bolivia, Plurinational State of"},
		{"BQ", "Bonaire, Sint Eustatius and Saba"},
		{"BA", "Bosnia and Herzegovina"},
		{"BW", "Botswana"},
		{"BV", "Bouvet Island"},
		{"BR", "Brazil"},
		{"IO", "British Indian Ocean Territory"},
		{"BN", "Brunei Darussalam"},
		{"BG", "Bulgaria"},
		{"BF", "Burkina Faso"},
		{"BI", "Burundi"},
		{"KH", "Cambodia"},
		{"CM", "Cameroon"},
		{"CA", "Canada"},
		{"CV", "Cape Verde"},
		{"KY", "Cayman Islands"},
		{"CF", "Central African Republic"},
		{"TD", "Chad"},
		{"CL", "Chile"},
		{"CN", "China"},
		{"CX", "Christmas Island"},
		{"CC", "Cocos (Keeling) Islands"},
		{"CO", "Colombia"},
		{"KM", "Comoros"},
		{"CG", "Congo"},
		{"CD", "Congo, the Democratic Republic of the"},
		{"CK", "Cook Islands"},
		{"CR", "Costa Rica"},
		{"CI", "Côte d'Ivoire"},
		{"HR", "Croatia"},
		{"CU", "Cuba"},
		{"CW", "Curaçao"},
		{"CY", "Cyprus"},
		{"CZ", "Czech Republic"},
		{"DK", "Denmark"},
		{"DJ", "Djibouti"},
		{"DM", "Dominica"},
		{"DO", "Dominican Republic"},
		{"EC", "Ecuador"},
		{"EG", "Egypt"},
		{"SV", "El Salvador"},
		{"GQ", "Equatorial Guinea"},
		{"ER", "Eritrea"},
		{"EE", "Estonia"},
		{"ET", "Ethiopia"},
		{"FK", "Falkland Islands (Malvinas)"},
		{"FO", "Faroe Islands"},
		{"FJ", "Fiji"},
		{"FI", "Finland"},
		{"FR", "France"},
		{"GF", "French Guiana"},
		{"PF", "French Polynesia"},
		{"TF", "French Southern Territories"},
		{"GA", "Gabon"},
		{"GM", "Gambia"},
		{"GE", "Georgia"},
		{"DE", "Germany"},
		{"GH", "Ghana"},
		{"GI", "Gibraltar"},
		{"GR", "Greece"},
		{"GL", "Greenland"},
		{"GD", "Grenada"},
		{"GP", "Guadeloupe"},
		{"GU", "Guam"},
		{"GT", "Guatemala"},
		{"GG", "Guernsey"},
		{"GN", "Guinea"},
		{"GW", "Guinea-Bissau"},
		{"GY", "Guyana"},
		{"HT", "Haiti"},
		{"HM", "Heard Island and McDonald Islands"},
		{"VA", "Holy See (Vatican City State)"},
		{"HN", "Honduras"},
		{"HK", "Hong Kong"},
		{"HU", "Hungary"},
		{"IS", "Iceland"},
		{"IN", "India"},
		{"ID", "Indonesia"},
		{"IR", "Iran, Islamic Republic of"},
		{"IQ", "Iraq"},
		{"IE", "Ireland"},
		{"IM", "Isle of Man"},
		{"IL", "Israel"},
		{"IT", "Italy"},
		{"JM", "Jamaica"},
		{"JP", "Japan"},
		{"JE", "Jersey"},
		{"JO", "Jordan"},
		{"KZ", "Kazakhstan"},
		{"KE", "Kenya"},
		{"KI", "Kiribati"},
		{"KP", "Korea, Democratic People's Republic of"},
		{"KR", "Korea, Republic of"},
		{"KW", "Kuwait"},
		{"KG", "Kyrgyzstan"},
		{"LA", "Lao People's Democratic Republic"},
		{"LV", "Latvia"},
		{"LB", "Lebanon"},
		{"LS", "Lesotho"},
		{"LR", "Liberia"},
		{"LY", "Libya"},
		{"LI", "Liechtenstein"},
		{"LT", "Lithuania"},
		{"LU", "Luxembourg"},
		{"MO", "Macao"},
		{"MK", "Macedonia, the former Yugoslav Republic of"},
		{"MG", "Madagascar"},
		{"MW", "Malawi"},
		{"MY", "Malaysia"},
		{"MV", "Maldives"},
		{"ML", "Mali"},
		{"MT", "Malta"},
		{"MH", "Marshall Islands"},
		{"MQ", "Martinique"},
		{"MR", "Mauritania"},
		{"MU", "Mauritius"},
		{"YT", "Mayotte"},
		{"MX", "Mexico"},
		{"FM", "Micronesia, Federated States of"},
		{"MD", "Moldova, Republic of"},
		{"MC", "Monaco"},
		{"MN", "Mongolia"},
		{"ME", "Montenegro"},
		{"MS", "Montserrat"},
		{"MA", "Morocco"},
		{"MZ", "Mozambique"},
		{"MM", "Myanmar"},
		{"NA", "Namibia"},
		{"NR", "Nauru"},
		{"NP", "Nepal"},
		{"NL", "Netherlands"},
		{"NC", "New Caledonia"},
		{"NZ", "New Zealand"},
		{"NI", "Nicaragua"},
		{"NE", "Niger"},
		{"NG", "Nigeria"},
		{"NU", "Niue"},
		{"NF", "Norfolk Island"},
		{"MP", "Northern Mariana Islands"},
		{"NO", "Norway"},
		{"OM", "Oman"},
		{"PK", "Pakistan"},
		{"PW", "Palau"},
		{"PS", "Palestinian Territory, Occupied"},
		{"PA", "Panama"},
		{"PG", "Papua New Guinea"},
		{"PY", "Paraguay"},
		{"PE", "Peru"},
		{"PH", "Philippines"},
		{"PN", "Pitcairn"},
		{"PL", "Poland"},
		{"PT", "Portugal"},
		{"PR", "Puerto Rico"},
		{"QA", "Qatar"},
		{"RE", "Réunion"},
		{"RO", "Romania"},
		{"RU", "Russia"},
		{"RW", "Rwanda"},
		{"BL", "Saint Barthélemy"},
		{"SH", "Saint Helena, Ascension and Tristan da Cunha"},
		{"KN", "Saint Kitts and Nevis"},
		{"LC", "Saint Lucia"},
		{"MF", "Saint Martin (French part)"},
		{"PM", "Saint Pierre and Miquelon"},
		{"VC", "Saint Vincent and the Grenadines"},
		{"WS", "Samoa"},
		{"SM", "San Marino"},
		{"ST", "Sao Tome and Principe"},
		{"SA", "Saudi Arabia"},
		{"SN", "Senegal"},
		{"RS", "Serbia"},
		{"SC", "Seychelles"},
		{"SL", "Sierra Leone"},
		{"SG", "Singapore"},
		{"SX", "Sint Maarten (Dutch part)"},
		{"SK", "Slovakia"},
		{"SI", "Slovenia"},
		{"SB", "Solomon Islands"},
		{"SO", "Somalia"},
		{"ZA", "South Africa"},
		{"GS", "South Georgia and the South Sandwich Islands"},
		{"SS", "South Sudan"},
		{"ES", "Spain"},
		{"LK", "Sri Lanka"},
		{"SD", "Sudan"},
		{"SR", "Suriname"},
		{"SJ", "Svalbard and Jan Mayen"},
		{"SZ", "Swaziland"},
		{"SE", "Sweden"},
		{"CH", "Switzerland"},
		{"SY", "Syrian Arab Republic"},
		{"TW", "Taiwan, Province of China"},
		{"TJ", "Tajikistan"},
		{"TZ", "Tanzania, United Republic of"},
		{"TH", "Thailand"},
		{"TL", "Timor-Leste"},
		{"TG", "Togo"},
		{"TK", "Tokelau"},
		{"TO", "Tonga"},
		{"TT", "Trinidad and Tobago"},
		{"TN", "Tunisia"},
		{"TR", "Turkey"},
		{"TM", "Turkmenistan"},
		{"TC", "Turks and Caicos Islands"},
		{"TV", "Tuvalu"},
		{"UG", "Uganda"},
		{"UA", "Ukraine"},
		{"AE", "United Arab Emirates"},
		{"GB", "United Kingdom"},
		{"US", "United States"},
		{"UM", "United States Minor Outlying Islands"},
		{"UY", "Uruguay"},
		{"UZ", "Uzbekistan"},
		{"VU", "Vanuatu"},
		{"VE", "Venezuela"},
		{"VN", "Viet Nam"},
		{"VG", "Virgin Islands, British"},
		{"VI", "Virgin Islands, U.S."},
		{"WF", "Wallis and Futuna"},
		{"EH", "Western Sahara"},
		{"YE", "Yemen"},
		{"ZM", "Zambia"},
		{"ZW", "Zimbabwe"},
	}

	// Countries with states.  See: https://en.m.wikipedia.org/wiki/Federated_state
	CountriesWithStates = map[string]bool{"AE":true,"AR":true,"AT":true,"AU":true,"BA":true,"BE":true,"BR":true,"CA":true,"CH":true,"DE":true,"ET":true,"FM":true,"IN":true,"IQ":true,"KM":true,"KN":true,"MX":true,"MY":true,"NG":true,"NP":true,"PK":true,"RU":true,"SD":true,"SO":true,"SS":true,"US":true,"VE":true}

	// Countries with postal codes.  See: https://www.ups.com/worldshiphelp/WS16/ENU/AppHelp/Codes/Countries_Territories_Requiring_Postal_Codes.htm
	CountriesWithPostalCodes = map[string]bool{"A2":true,"AM":true,"AR":true,"AT":true,"AU":true,"AZ":true,"BA":true,"BD":true,"BE":true,"BG":true,"BN":true,"BR":true,"BY":true,"CA":true,"CH":true,"CN":true,"CS":true,"CY":true,"CZ":true,"DE":true,"DK":true,"DZ":true,"EE":true,"EN":true,"ES":true,"FI":true,"FO":true,"FR":true,"GB":true,"GE":true,"GG":true,"GL":true,"GR":true,"GU":true,"HO":true,"HR":true,"HU":true,"IC":true,"ID":true,"IL":true,"IN":true,"IT":true,"JE":true,"JP":true,"KG":true,"KO":true,"KR":true,"KZ":true,"LI":true,"LK":true,"LT":true,"LU":true,"LV":true,"M3":true,"ME":true,"MG":true,"MH":true,"MK":true,"MN":true,"MQ":true,"MX":true,"MY":true,"NB":true,"NL":true,"NO":true,"NT":true,"NZ":true,"PH":true,"PK":true,"PL":true,"PO":true,"PR":true,"PT":true,"RE":true,"RU":true,"SA":true,"SE":true,"SF":true,"SG":true,"SI":true,"SK":true,"SX":true,"TH":true,"TJ":true,"TM":true,"TN":true,"TR":true,"TU":true,"TW":true,"UA":true,"US":true,"UV":true,"UY":true,"UZ":true,"VA":true,"VI":true,"VL":true,"VN":true,"WL":true,"YA":true,"YT":true,"ZA":true}
)