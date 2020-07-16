// gozilla.go
package main
import (
	"encoding/json"
	"fmt"
	"net/http"
)

var (
	anonymityLevels = OptionData {
		{"R",	"Real name - Aaron Smith"},
		{"A",	"Alias - magicsquare666"},
		{"F",	"Random Anonymous Name - Wacky Panda"},
	}
)

const (
	kTitle = "title"
	kCategory = "category"
	kAnonymity = "anonymity"
	kThumbnail = "thumbnail"
)

///////////////////////////////////////////////////////////////////////////////
//
// create top menu - create dropdown with creation choices
//
///////////////////////////////////////////////////////////////////////////////
func createHandler(w http.ResponseWriter, r *http.Request) {
	executeTemplate(w, kCreate, makeFormFrameArgs(makeForm(), "Create"))
}

///////////////////////////////////////////////////////////////////////////////
//
// create link post
//
///////////////////////////////////////////////////////////////////////////////
func createLinkHandler(w http.ResponseWriter, r *http.Request) {
	const kLink = "link"

	form := makeForm(
		nuTextField(kLink, "Share an article link", 50, 1, 1024, "article link").addFnValidator(urlValidator(false)).noSpellCheckOrCaps(),
		nuTextField(kTitle, "Add a title", 50, 12, kMaxTitleLength, "article title"),
		nuSelectField(kCategory, "Select Category", newsCategoryInfo.CategorySelect, true, true, false, false, "Please select a news category"),
		nuHiddenField(kThumbnail, ""),
	)

	userId := GetSession(r)
	if userId == -1 { // Secure cookie not found.  Either session expired, or someone is hacking.
		// So go to the register page.
		pr("Must be logged in create a post.  TODO: add createLinkHandler to stack somehow... popup window?")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	if r.Method == "POST" && form.validateData(r) { // On POST, validates and captures the request data.
		prVal("form", form)

		pr("Inserting new LinkPost into database.")

		prVal("form", form)

		// TODO: some sort of security check for malicious linkUrl.

		// Update the user record with registration details.
		newPostId := DbInsert(
			`INSERT INTO $$LinkPost(UserId, LinkURL, Title, Category, UrlToImage)
			 VALUES($1::bigint, $2, $3, $4, $5) returning id;`,
			userId,
			form.val(kLink),
			form.val(kTitle),
			form.val(kCategory),
			form.val(kThumbnail))

		// Have user like their own posts by default.
		voteUpDown(newPostId, userId, true, true, false)

		http.Redirect(w, r, fmt.Sprintf("/article/?postId=%d", newPostId), http.StatusSeeOther)
		return
	}

	executeTemplate(w, kCreateLink, makeFormFrameArgs(form, "Create Link Post"))
}

///////////////////////////////////////////////////////////////////////////////
//
// create poll post
//
///////////////////////////////////////////////////////////////////////////////
func createPollHandler(w http.ResponseWriter, r *http.Request) {
	pr("createPollHandler")

	const (
		kOption1 = "option1"
		kOption2 = "option2"
		kAnyoneCanAddOptions = "bAnyoneCanAddOptions"
		kCanSelectMultipleOptions = "bCanSelectMultipleOptions"
		kRankedChoiceVoting = "bRankedChoiceVoting"
	)

	userId := GetSession(r)
	if userId == -1 { // Secure cookie not found.  Either session expired, or someone is hacking.
		// So go to the register page.
		pr("Must be logged in create a post.  TODO: add createPollHandler to stack somehow.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	prVal("r.Method", r.Method)

	form := makeForm(
		nuTextField(kTitle, "Ask a poll question...", 50, 12, 100, "poll question"),
		nuTextField(kOption1, "add option...", 50, 1, 50, "poll option 1"),
		nuTextField(kOption2, "add option...", 50, 1, 50, "poll option 2"),
		nuBoolField(kAnyoneCanAddOptions, "Allow anyone to add options", true),
		nuBoolField(kCanSelectMultipleOptions, "Allow people to select multiple options", true),
		nuBoolField(kRankedChoiceVoting, "Enable ranked-choice voting", false),
		nuSelectField(kCategory, "Select Category", newsCategoryInfo.CategorySelect, true, true, false, false, "Please select a poll category"),
	)

	// Add fields for additional options that were added, there could be an arbitrary number, we'll cap it at 1024 for now.
	pr("Adding additional poll options")
	pollOptions := []*Field{form.field(kOption1), form.field(kOption2)}

	// Just use brute force for now.  Don't break at the end, as we don't want the bricks to fall when someone erases the name of an option in the middle.
	// TODO: optimize this later, if necessary, possibly with a hidden length field, if necessary.

	for i := 3; i < 1024; i++ {
		optionName := fmt.Sprintf("option%d", i)
		// TODO: How should this case work?  Could be used as a case for removing options, if poll is not yet live.
		//       Once live, options with votes should not be removable.
		//       Leave the ""'s in the list so the position within the array can map directly to votes and indexes.
		if r.FormValue(optionName) != "" {
			prVal("Adding new poll option", optionName)
			newOption := nuTextField(optionName, "add option...", 50, 1, 50, fmt.Sprintf("poll option %d", i))
			form.addField(newOption)
			pollOptions = append(pollOptions, newOption)
		}
	}

	prVal("r.Method", r.Method)
	prVal("r.PostForm", r.PostForm)
	//prVal("form", form)

	if r.Method == "POST" && form.validateData(r) {
		pr("Valid form!")

		prVal("form.val(kCanSelectMultipleOptions)", form.val(kCanSelectMultipleOptions))
		prVal("form.val(kRankedChoiceVoting)", form.val(kRankedChoiceVoting))

		if form.boolVal(kCanSelectMultipleOptions) && form.boolVal(kRankedChoiceVoting) { // These flags that cannot coexist.
			pr("Both flags are set at once!")
			form.setFieldError(kCanSelectMultipleOptions, "Cannot select Multiple Options and Ranked Choice options both at once")
			form.setFieldError(kRankedChoiceVoting,       "Cannot select Multiple Options and Ranked Choice options both at once")
		} else {
			pr("Inserting new PollPost into database.")

			// Serialize all of the poll options and flags into variables that can be inserted into database.
			var pollOptionData PollOptionData
			for i := 1; i < 1024; i++ {
				value := r.FormValue(fmt.Sprintf("option%d", i))
				if value != "" {
					pollOptionData.Options = append(pollOptionData.Options, value)
				}
			}
			pollOptionData.AnyoneCanAddOptions      = form.boolVal(kAnyoneCanAddOptions)
			pollOptionData.CanSelectMultipleOptions = form.boolVal(kCanSelectMultipleOptions)
			pollOptionData.RankedChoiceVoting       = form.boolVal(kRankedChoiceVoting)

			pollOptionsJson, err := json.Marshal(pollOptionData)
			check(err)

			//prVal("pollOptionsJson", pollOptionsJson)

			// Create the new poll.
			pollPostId := DbInsert(
				`INSERT INTO $$PollPost(UserId, Title, Category, Language, Country, UrlToImage,
										PollOptionData)
				 VALUES($1::bigint, $2, $3, $4, $5, $6,
						$7) returning id;`,
				userId,
				form.val(kTitle),
				form.val(kCategory),
				"en",
				"us",
				"http://localhost:8080/static/ballotbox 3dinos.jpg", // TODO: generate poll url from image search
				pollOptionsJson,
			)
			prVal("Just added a poll #", pollPostId)

			// Have user like their own polls by default.
			voteUpDown(pollPostId, userId, true, true, false)

			http.Redirect(w, r, fmt.Sprintf("/article/?postId=%d", pollPostId), http.StatusSeeOther)
			return
		}
	} else if r.Method == "POST" {
		prVal("Invalid form!!", form)
	}

	args := struct {
		PageArgs
		Form			Form
		PollOptions		[]*Field
	} {
		PageArgs: 		PageArgs{Title: "Create Poll"},
		Form: 			*form,
		PollOptions:	pollOptions,
	}

	executeTemplate(w, kCreatePoll, args)
}

///////////////////////////////////////////////////////////////////////////////
//
// create blog post
//
///////////////////////////////////////////////////////////////////////////////
func createBlogHandler(w http.ResponseWriter, r *http.Request) {
	pr("createBlogHandler")

	const kBlogVal = "blogVal"

	userId := GetSession(r)
	if userId == -1 { // Secure cookie not found.  Either session expired, or someone is hacking.
		// So go to the register page.
		pr("Must be logged in create a post.  TODO: add createPollHandler to stack somehow.")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	prVal("r.Method", r.Method)

	form := makeForm(
		nuTextField(kTitle, "Your blog post title...", 50, 12, 255, "blog title"),
		nuSelectField(kCategory, "Select Category", newsCategoryInfo.CategorySelect, true, true, false, false, "Please select a blog category"),
		nuHiddenField(kBlogVal, ""),  // Hidden field that gets the value from the blog.  Because there is JS required to get blog value.
	)

	form.field(kBlogVal).addFnValidator(requiredValidator("blog article"))
	form.field(kBlogVal).addFnValidator(minMaxLengthValidator(12, 40000, "blog article"))

	if r.Method == "POST" && form.validateData(r) {
		prVal("Valid form!!", form)

		prVal(kTitle, form.val(kTitle))
		prVal(kBlogVal, form.val(kBlogVal))

		// Update the user record with registration details.
		blogPostId := DbInsert(
			`INSERT INTO $$LinkPost(UserId, Title, Category, UrlToImage, Blog)
			 VALUES($1::bigint, $2, $3, $4) returning id;`,
			userId,
			form.val(kTitle),
			form.val(kCategory),
			"http://localhost:8080/static/ballotbox.png", // TODO: generate poll url from image search
			form.val(kBlogVal))

		// Have user like their own blogs by default.
		voteUpDown(blogPostId, userId, true, true, false)

		return
	} else if r.Method == "POST" {
		prVal("Invalid form!!", form)
	}

	executeTemplate(w, kCreateBlog, makeFormFrameArgs(form, "Create Blog Post"))
}
