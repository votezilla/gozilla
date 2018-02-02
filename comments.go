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
	
	// Render the news articles.
	commentsArgs := struct {
		PageArgs
		Username		string
		UserId			int64
		NavMenu			[]string
		UrlPath			string
		Article			Article
		Comments		string
	}{
		PageArgs:		PageArgs{Title: "votezilla - Comments"},
		Username:		username,
		UserId:			userId,
		NavMenu:		navMenu,
		UrlPath:		"news",
		Article:		article,
		Comments:		comments,
	}
	
	executeTemplate(w, "comments", commentsArgs)
}