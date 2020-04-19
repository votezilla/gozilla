package main

import (
	"net/http"
	"strconv"
)

//////////////////////////////////////////////////////////////////////////////
//
// display article
//
//////////////////////////////////////////////////////////////////////////////
func articleHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)

	prVal(co_, "r.URL.Query()", r.URL.Query())

	reqPostId := parseUrlParam(r, "postId")
	postId, err := strconv.ParseInt(reqPostId, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // TODO: prettify error displaying - use dinosaurs.
		return
	}

	// Get the username.
	userId := GetSession(r)
	username := getUsername(userId)

	prVal(co_, "userId", userId);
	prVal(co_, "username", username);

	// TODO_REFACTOR: unify articles and posts in database.
	article, err := fetchArticle(postId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // TODO: prettify error displaying - use dinosaurs.
		return
	}

	upvotes, downvotes := deduceVotingArrows([]Article{article})

	// Render the news articles.
	articleArgs := struct {
		PageArgs
		Username		string
		UserId			int64
		NavMenu			[]string
		UrlPath			string
		Article			Article
		UpVotes			[]int64
		DownVotes		[]int64
		Comments		[]CommentTag
	}{
		PageArgs:		PageArgs{Title: "votezilla - Article"},
		Username:		username,
		UserId:			userId,
		NavMenu:		navMenu,
		UrlPath:		"news",
		Article:		article,
		UpVotes:		upvotes,
		DownVotes:		downvotes,
		Comments:		ReadCommentTagsFromDB(article.Id),
	}

	executeTemplate(w, "article", articleArgs)
}
