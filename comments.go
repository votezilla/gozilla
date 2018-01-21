package main

import (
	"net/http"
	"strconv"
)


//////////////////////////////////////////////////////////////////////////////
//
// display comments
//
//////////////////////////////////////////////////////////////////////////////
func commentsHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)
	  
	prVal(nw_, "r.URL.Query()", r.URL.Query())

	reqPostId := parseUrlParam(r, "postId")
	postId, err := strconv.ParseInt(reqPostId, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // TODO: prettify error displaying - use dinosaurs.
		return
	}
	
	// TODO_REFACTOR: unify articles and posts in database.
	article, err := fetchArticle(postId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // TODO: prettify error displaying - use dinosaurs.
		return
	}
	
	comments := "TODO: NESTED COMMENTS!"

	// Get the username.
	userId := GetSession(r)
	username := getUsername(userId)
	
	// TODO: make a function to convert Article to ArticleArg, news.go also uses this.
	articleArg := ArticleArg{}
	articleArg.Article	= article			
	if article.NewsSourceId != "" {
		articleArg.AuthorIconUrl = "/static/newsSourceIcons/" + article.NewsSourceId + ".png"
	} else {
		articleArg.AuthorIconUrl = "/static/mozilla dinosaur head.png" // TODO: we need real dinosaur icons for users.
	}

	// Render the news articles.
	commentsArgs := struct {
		PageArgs
		Username		string
		NavMenu			[]string
		UrlPath			string
		Article			ArticleArg
		Comments		string
	}{
		PageArgs:		PageArgs{Title: "votezilla - Comments"},
		Username:		username,
		NavMenu:		navMenu,
		UrlPath:		"news",
		Article:		articleArg,
		Comments:		comments,
	}
	
	executeTemplate(w, "comments", commentsArgs)
}