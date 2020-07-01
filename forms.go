// forms.go
package main

import (
	"fmt"
	"html"
    "net/http"
	"regexp"
    "strconv"
//    "net/url"
)

type OptionData [][2]string
type Attributes map[string]string

const (
	kNuField = "nuField"
	kNuCheckbox = "nuCheckbox"
)

var (
	NoSpellCheck = Attributes{
		"autocorrect": "off",
		"spellcheck": "false",
		"autocapitalize": "off",
	}
)


// ================================================================================
//
// -------------------------------- type Validator --------------------------------
//
// ================================================================================
type Validator func(value string)(bool, string)

func requiredValidator() func(string)(bool, string) {

	return func(value string)(bool, string) {
		if value != "" {
			return true, ""
		} else {
			return false, "This field is required."
		}
	}
}

func minMaxLengthValidator(minLength, maxLength int) Validator {

	return func(value string)(bool, string) {
		length := len(value)

		if length < minLength {
			return false, fmt.Sprintf("Ensure this value has at least %v characters", minLength)
		} else if length > maxLength {
			return false, fmt.Sprintf("Ensure this value has at most %v characters", maxLength)
		} else {
			return true, ""
		}
	}
}

func optionValidator(validOptions []string) Validator {

	return func(value string)(bool, string) {
		for _, validOption := range validOptions {
	        if value == validOption {
	            return true, ""
	        }
	    }
	    return false, fmt.Sprintf("Invalid option selected")
	}
}


// The regular expression pattern to search for the provided value.
// Returns error if regxp#MatchString is False.
func regexValidator(regex, errorMsg string) Validator {
	return func(value string)(bool, string) {
		rx, err := regexp.Compile(regex)
		if err != nil {
			return false, err.Error()
		}

		if rx.MatchString(value) {
			return true, ""
		} else {
			return false, errorMsg
		}
	}
}

// An EmailValidator that ensures a value looks like an international email address.
func emailValidator() Validator {
	return regexValidator(`^.+@.+$`, "Enter a valid email address.")
}

// A FullNameValidator that ensures that we have a full name (e.g. 'John Doe').
func fullNameValidator() Validator {
	return regexValidator(`^[\p{L}]+( [\p{L}]+)+$`, "Enter a valid full name (i.e. 'John Doe').")
}

// An URLValidator that ensures a value looks like an url.
// TODO: Security check for malicious website links.  See: https://geekflare.com/security-threats-detection-api/
func urlValidator(schemeRequired bool) Validator {
	return regexValidator(`^(https?|ftp)(:\/\/[-_.!~*\'()a-zA-Z0-9;\/?:\@&=+\$,%#]+)$`, "Enter a valid url.")

	//if schemeRequired {
	//	return regexValidator(`^(https?|ftp)(:\/\/[-_.!~*\'()a-zA-Z0-9;\/?:\@&=+\$,%#]+)$`, "Enter a valid url.")
	//} else {
	//	return regexValidator(`^([-_.!~*\'()a-zA-Z0-9;\/?:\@&=+\$,%#]+)$`, "Enter a valid url.")
	//}

	//return func(value string)(bool, string) {
	//	prVal("\n\n\n!!!!!urlValidator called, value", value)
	//
	//	URL, err := url.Parse(value)
	//
	//	prVal("URL", URL)
	//	prVal("err", err)
	//
	//	if err != nil {
	//		return false, "Invalid URL"
	//	} else {
	//		return true, ""
	//	}
	//}
}

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
	Classes			string
	Id				string
	Error			string
	Subtext			string
	Length			int

	Html			func() string // Closure that outputs the html of this field
	HtmlRow			func() string // Closure that outputs the html of this field's entire table row

	Validators		[]Validator
	Attributes		Attributes

	// Radio form data:
	OptionKeyValues OptionData
	StartAtNil		bool
	Required		bool
	HasOther		bool
	Skippable		bool
}

func (f *Field) validate() bool {
//	prVal("Field.validate() for field", f)

	for k, _ := range f.Validators {
		validator := f.Validators[k]

		isValid, errorMsg := validator(f.Value)

//		prf("  Field.validate() - isValid, errorMsg = %s, %s for validator %#v", bool_to_str(isValid), errorMsg, validator)

		if !isValid {
			// Note: Just return the first error, don't accumulate them.
			f.Error = errorMsg

//			prVal("    !isValid --> f.Error", f.Error)

			return false
		}
	}
	return true
}

func (f Field) val() string {
	return f.Value
}

func (f Field) intVal() int {
	val, err := strconv.Atoi(f.Value)
	check(err)
	return ternary_int(err != nil, 0, val)
}

func (f Field) int64Val() int64 {
	val, err := strconv.ParseInt(f.Value, 10, 64)
	check(err)
	return ternary_int64(err != nil, int64(0), val)
}

func (f Field) boolVal() bool {
	return f.Value != ""
}

func (f Field) getErrorHtml() string {
	//prf("Field.getErrorHtml() for field %s f.Error = %s", f, f.Error)
	return ternary_str(f.Error != "", fmt.Sprintf("<label class=\"error\">%s</label>", f.Error), "")
}

func (f *Field) setError(errorMsg string) {
	f.Error = errorMsg
}

func (f *Field) noSpellCheck() *Field {
	f.Attributes = NoSpellCheck
	return f
}

func (f *Field) subtext(text string) *Field {
	f.Subtext = text
	return f
}

func (f *Field) addFnValidator(validator Validator) (*Field) {
	f.Validators = append(f.Validators, validator)
	return f
}

func (f *Field) addRegexValidator(regexp, errorMsg string) (*Field) {
	f.Validators = append(f.Validators, regexValidator(regexp, errorMsg))
	return f
}


// ================================================================================
//
// -------------------------------- struct Form -----------------------------------
//
// ================================================================================
type Form struct {
	FieldList		[]*Field			// To remember the sequential order of field.
	FieldMap		map[string]*Field	// To lookup fields by name.
}

func (f *Form) processData(r *http.Request) {
//	pr("Form.processData")

	for name, _ := range f.FieldMap {
		value := r.FormValue(name)

		f.FieldMap[name].Value = value

//		prf("Form.processData field: '%s' value: '%s'", name, value)
	}

//	prVal("AFTER Form.processData f", *f) // << ERROR first seen here!
}

// Accessors
func (f Form) field(fieldName string) *Field	{ return f.FieldMap[fieldName] 		   }
func (f Form) val(fieldName string) 	 string { return f.field(fieldName).val()	   }
func (f Form) intVal(fieldName string) 	 int 	{ return f.field(fieldName).intVal()   }
func (f Form) int64Val(fieldName string) int64	{ return f.field(fieldName).int64Val() }
func (f Form) boolVal(fieldName string)  bool 	{ return f.field(fieldName).boolVal()  }

func (f *Form) setFieldError(fieldName string, errorMsg string) {
	f.field(fieldName).setError(errorMsg)
}

func (f *Form) validate() bool {
//	prVal("Form.validate for form", *f)

	valid := true
	for _, field := range f.FieldList {
		v := field.validate()

//		prf("Form.validation is %s for field %#v", bool_to_str(v), field)

		valid = valid && v
	}

//	prVal("Form.validate return", valid)

	return valid
}

func (f *Form) validateData(r *http.Request) bool {
//	pr("Form.validateData")

	f.processData(r)

//	prVal("Form.validateData processed data form", f)

	return f.validate()
}

func makeForm(fields ...*Field) *Form {
	f := new(Form)
	f.FieldList = make([]*Field, len(fields))
	f.FieldMap = make(map[string]*Field)

	for i, field := range(fields) {
		f.FieldList[i] = field
		f.FieldMap[field.Name] = field
	}

	return f
}

func (f *Form) addField(field *Field) {
	f.FieldList = append(f.FieldList, field)
	f.FieldMap[field.Name] = field
}

func (f Field) getHtml() string {
	return fmt.Sprintf(
		"<input type=%s name=\"%s\" value=\"%s\" placeholder=\"%s\" length=\"%d\">%s",
		f.Type,
		f.Name,
		html.EscapeString(f.Value),  // Prevents HTML-injection attack!!!  (Since the user can affect this value.)
		f.Placeholder,
		f.Length,
		f.getErrorHtml())
}

// Field factories

func makeTextField(name, label, placeholder string, inputLength, minLength, maxLength int) *Field {
	f := Field{Name: name, Type: "text", Label: label, Placeholder: placeholder, Length: inputLength}

	if minLength > 0 {
		f.Validators = append(f.Validators, requiredValidator())
	}

	if minLength > 0 || maxLength != -1 {
		f.Validators = append(f.Validators, minMaxLengthValidator(minLength, maxLength))
	}

	// TODO: HTML-escape this
	f.Html = func()string {
		prVal("f.Html f", f)

		return f.getHtml()
	}
	f.HtmlRow = func()string {
		prVal("f.HtmlRow f", f)

		return fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>\n", f.Label, f.getHtml())
	}

	//TODO: [] add RowHtml function (which includes Label == Placeholder parameter)

//	prVal("makeTextField Type", f.Type)

	return &f
}
func MakeTextField(name string, inputLength, minLength, maxLength int) *Field {
	return makeTextField(name, name + ":", name + "...", inputLength, minLength, maxLength)
}
func nuTextField(name, placeholder string, inputLength, minLength, maxLength int) *Field {
	f := makeTextField(name, "", placeholder, inputLength, minLength, maxLength)
	f.Placeholder = placeholder
	f.Classes = kNuField

//	prVal("nuTextField f.Type", f.Type)

	return f
}

func makePasswordField(name, label, placeholder string, inputLength, minLength, maxLength int) *Field {
	f := makeTextField(name, label, placeholder, inputLength, minLength, maxLength)
	f.Placeholder = placeholder
	f.Type = "password"
	return f
}
func MakePasswordField(name string, inputLength, minLength, maxLength int) *Field {
	return makePasswordField(name,  name + ":", name + "...", inputLength, minLength, maxLength)
}
func nuPasswordField(name, placeholder string, inputLength, minLength, maxLength int) *Field {
	f := makePasswordField(name, "", placeholder, inputLength, minLength, maxLength)
	f.Placeholder = placeholder
	f.Classes = kNuField
	return f
}

func makeHiddenField(name, defaultValue string) *Field {
	f := Field{Name: name, Value: defaultValue, Type: "hidden"}

	// TODO: HTML-escape this
	f.Html = func()string {
		return fmt.Sprintf(
			"<input type=hidden name=\"%s\" value=\"%s\">",
			f.Name,
			html.EscapeString(f.Value),  // Prevents against HTML-injection attacks!
	)}
	f.HtmlRow = func()string { return f.Html() }

	return &f
}
func nuHiddenField(name, defaultValue string) *Field { return makeHiddenField(name, defaultValue); }

// TODO: implement makeRichTextField().  It's just a copy of makeTextField at the moment.
func makeRichTextField(name, label, placeholder string, inputLength, minLength, maxLength int) *Field {
	f := Field{Name: name, Type: "text", Label: label, Placeholder: placeholder, Length: inputLength}

	if minLength > 0 {
		f.Validators = append(f.Validators, requiredValidator())
	}

	if minLength > 0 || maxLength != -1 {
		f.Validators = append(f.Validators, minMaxLengthValidator(minLength, maxLength))
	}

	// TODO: HTML-escape this
	f.Html = func()string {
		prVal("f.Html f", f)

		return f.getHtml()
	}
	f.HtmlRow = func()string {
		prVal("f.HtmlRow f", f)

		return fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>\n", f.Label, f.getHtml())
	}

	//TODO: [] add RowHtml function (which includes Label == Placeholder parameter)

	return &f
}
func MakeRichTextField(name string, inputLength, minLength, maxLength int) *Field {
	return makeRichTextField(name,  name + ":", name + "...", inputLength, minLength, maxLength)
}

func makeBoolField(name, label, optionText string, defaultValue bool) *Field {
	// Hack: using Placeholder to hold optionText value
	f := Field{Name: name, Type: "checkbox", Label: label, Placeholder: optionText, Value: ternary_str(defaultValue, "true", "")}

	// TODO: HTML-escape this
	f.Html = func()string {
		return fmt.Sprintf(
			"<input type=checkbox name=\"%s\" value=\"true\" %s>%s",
			f.Name,
			ternary_str(f.boolVal(), "checked", ""),
			f.getErrorHtml())
	}
	f.HtmlRow = func()string { return fmt.Sprintf("<tr><td>%s</td><td>%s %s</td></tr>\n", f.Label, f.Html(), f.Placeholder) }

	return &f
}
func MakeBoolField(name string, defaultValue bool) *Field {
	return makeBoolField(name, name + ":", "", defaultValue)
}
func nuBoolField(name, optionText string, defaultValue bool) *Field {
	f := makeBoolField(name, "", optionText, defaultValue)
	f.Classes = kNuCheckbox
	return f
}

func makeSelectField(name, label string, optionKeyValues OptionData, startAtNil, required, hasOther bool) *Field {
	f := Field{Name: name, Type: "select", Label: label}

	if required {
		f.Validators = append(f.Validators, requiredValidator())
	}

	validOptions := make([]string, len(optionKeyValues) + 1)

	for _, optionKeyValue := range optionKeyValues {
		validOptions = append(validOptions, optionKeyValue[0]) // add the key
	}

	f.Validators = append(f.Validators, optionValidator(validOptions))

	f.OptionKeyValues = optionKeyValues  // Needed for nuForm.

	// TODO: HTML-escape this
	f.Html = func()string {
		str := fmt.Sprintf("\n<select name=\"%s\">\n", f.Name)

		if startAtNil {
			str += "<option value=\"\">-</option>\n"
		}

		if hasOther {
			str += "<option value=\"0\">Other</option>\n"
		}

		for _, optionKeyValue := range optionKeyValues {
			str += fmt.Sprintf("<option value=\"%s\"%s>%s</option>\n",
				optionKeyValue[0], // key
				ternary_str(f.Value == optionKeyValue[0], " selected", ""),
				optionKeyValue[1]) // value
		}
		str += "</select>\n"
		str += f.getErrorHtml()
		return str
	}
	f.HtmlRow = func()string { return fmt.Sprintf("<tr><td>%s</td><td>%s</td></tr>\n", f.Label, f.Html()) }

	return &f
}
func MakeSelectField(name string, optionKeyValues OptionData, startAtNil, required, hasOther bool) *Field {
	return makeSelectField(name, name + ":", optionKeyValues, startAtNil, required, hasOther)
}
func nuSelectField(name, placeholder string, optionKeyValues OptionData, startAtNil, required, hasOther, skippable bool) *Field {
	f := makeSelectField(name, "", optionKeyValues, startAtNil, required, hasOther)
	f.Placeholder = placeholder
	f.Classes = kNuField

	f.StartAtNil = startAtNil
	f.Required	 = required
	f.HasOther	 = hasOther
	f.Skippable	 = skippable

	return f
}

/*
// === FORM TEMPLATE ARGS ===
type TableFormArgs struct {
	Form		Form
	CallToAction	string
	AdditionalError string
}

type NuFormArgs struct {
	Form			Form
	Title			string
	Congrats		string
	Introduction	string
	Footer			string
	Script			string
}

// Template arguments for form webpage template.
type FormArgs struct {
	PageArgs
	Form			Form
	Congrats		string
	Introduction	string
	Footer			string
}
*/

type FormFrameArgs struct {
	PageArgs
	Form			Form
}
func makeFormFrameArgs(form *Form, title string) FormFrameArgs {
	return FormFrameArgs {
		PageArgs: 		PageArgs{Title: title},
		Form: 			*form,
	}
}