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
	
	// Get the username.
	userId := GetSession(r)
	username := getUsername(userId)	
	
	// TODO_REFACTOR: unify articles and posts in database.
	article, err := fetchArticle(postId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // TODO: prettify error displaying - use dinosaurs.
		return
	}
	
	upvotes, downvotes := deduceVotingArrows([]Article{article})
	
	comments := "TODO: NESTED COMMENTS!"
	
	// Render the news articles.
	commentsArgs := struct {
		PageArgs
		Username		string
		UserId			int64
		NavMenu			[]string
		UrlPath			string
		Article			Article
		UpVotes			[]int64
		DownVotes		[]int64
		Comments		string
	}{
		PageArgs:		PageArgs{Title: "votezilla - Comments"},
		Username:		username,
		UserId:			userId,
		NavMenu:		navMenu,
		UrlPath:		"news",
		Article:		article,
		UpVotes:		upvotes, 
		DownVotes:		downvotes,
		Comments:		comments,
	}
	
	executeTemplate(w, "comments", commentsArgs)
}

// TODO: This is just duplicate code, make it view the results.  (Same or different handler for adding the vote?)
func viewPollResultsHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)
	  
	prVal(nw_, "r.URL.Query()", r.URL.Query())

	reqPostId := parseUrlParam(r, "postId")
	postId, err := strconv.ParseInt(reqPostId, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // TODO: prettify error displaying - use dinosaurs.
		return
	}
	
	// Get the username.
	userId := GetSession(r)
	username := getUsername(userId)	
	
	// TODO_REFACTOR: unify articles and posts in database.
	article, err := fetchArticle(postId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // TODO: prettify error displaying - use dinosaurs.
		return
	}
	
	upvotes, downvotes := deduceVotingArrows([]Article{article})
	
	comments := "TODO: NESTED COMMENTS!"
	
	// Render the news articles.
	commentsArgs := struct {
		PageArgs
		Username		string
		UserId			int64
		NavMenu			[]string
		UrlPath			string
		Article			Article
		UpVotes			[]int64
		DownVotes		[]int64
		Comments		string
	}{
		PageArgs:		PageArgs{Title: "View Poll Results"},
		Username:		username,
		UserId:			userId,
		NavMenu:		navMenu,
		UrlPath:		"news",
		Article:		article,
		UpVotes:		upvotes, 
		DownVotes:		downvotes,
		Comments:		comments,
	}
	
	executeTemplate(w, "viewPollResults", commentsArgs)	
}