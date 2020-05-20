package main

import (
	"net/http"
	"strconv"
	"time"
)


// JSON-parsed poll options - all the data that defines a poll.
type PollOptionData struct {
	Options						[]string	//`db:"options"`
	AnyoneCanAddOptions			bool		//`db:"bAnyoneCanAddOptions"`
	CanSelectMultipleOptions	bool		//`db:"bCanSelectMultipleOptions"`
	RankedChoiceVoting			bool		//`db:"bRankedChoiceVoting"`
}


// JSON-parsed format of an article.
type Article struct {
	Author			string
	Title			string
	Description		string
	Url				string
	UrlToImage		string
	PublishedAt		string
	// Custom parameters:
	Id				int64
	UrlToThumbnail	string
	NewsSourceId	string
	Host			string
	Category		string
	Language		string
	Country			string
	PublishedAtUnix	time.Time
	TimeSince		string
	Size			int		// 0=normal, 1=large (headline)
	AuthorIconUrl	string
	Bucket			string  // "" by default, but can override Category as a way to categorize articles
	Upvoted			int
	VoteTally		int
	NumComments		int

	IsPoll			bool
	PollOptionData	PollOptionData
}


//////////////////////////////////////////////////////////////////////////////
//
// display article
//
//////////////////////////////////////////////////////////////////////////////
func articleHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)

	pr("articleHandler")

	prVal("r.URL.Query()", r.URL.Query())

	reqPostId := parseUrlParam(r, "postId")
	postId, err := strconv.ParseInt(reqPostId, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // TODO: prettify error displaying - use dinosaurs.
		return
	}

	// Get the username.
	userId := GetSession(r)
	username := getUsername(userId)

	prVal("userId", userId)
	prVal("username", username)

	// TODO_REFACTOR: unify articles and posts in database.
	article, err := fetchArticle(postId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // TODO: prettify error displaying - use dinosaurs.
		return
	}

	upvotes, downvotes := deduceVotingArrows([]Article{article})

	articleGroups := make([]ArticleGroup, len(newsCategoryInfo.CategoryOrder))

	for c, category := range newsCategoryInfo.CategoryOrder {
		articleGroups[c].Category = category
		articleGroups[c].HeaderColor = newsCategoryInfo.HeaderColors[category]
	}

	moreArticles := []Article{}
	if article.IsPoll {
		moreArticles = fetchPolls()
	} else {
		moreArticles = fetchArticlesFromThisNewsSource(article.NewsSourceId)
	}
	prVal("moreArticles", moreArticles)

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
		ArticleGroups	[]ArticleGroup
		MoreArticlesFromThisSource	[]Article
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
		ArticleGroups:	articleGroups,
		MoreArticlesFromThisSource:	moreArticles,
	}

	executeTemplate(w, kArticle, articleArgs)
}
