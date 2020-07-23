package main

import (
	"fmt"
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
	// News parameters:
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
	Size			int		// 0=normal, 1=large (headline), 2=full page (article or viewPollResults)
	AuthorIconUrl	string
	Bucket			string  // "" by default, but can override Category as a way to categorize articles
	Upvoted			int
	VoteTally		int
	NumComments		int
	NumLines		int
	ThumbnailStatus	int
	IsThumbnail		bool

	// Poll parameters:
	IsPoll				bool
	WeVoted				bool
	PollOptionData		PollOptionData
	PollTallyResults	PollTallyResults
	VoteOptionIds	 	[]int64
	VoteData			[]bool
	//VoteOptionsMap		map[int64]bool
	LongestItem			int

	Ellipsify			func(text string, maxLength int) string
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

	prVal("r.URL", r.URL)
	prVal("r.URL.Path", r.URL.Path)

	reqPostId := parseUrlParam(r, "postId")
	postId, err := strconv.ParseInt(reqPostId, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // TODO: prettify error displaying - use dinosaurs.
		return
	}

	userId, username := GetSessionInfo(w, r)

	// TODO_REFACTOR: unify articles and posts in database.
	article, err := fetchArticle(postId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // TODO: prettify error displaying - use dinosaurs.
		return
	}

	articleGroups := make([]ArticleGroup, len(newsCategoryInfo.CategoryOrder))

	for c, category := range newsCategoryInfo.CategoryOrder {
		articleGroups[c].Category = category
		articleGroups[c].HeaderColor = newsCategoryInfo.HeaderColors[category]
	}

	// Suggested articles for further reading - on the sidebar.
	moreArticles := []Article{}
	if article.IsPoll {
		reqChangeVote := parseUrlParam(r, "changeVote")
		prVal("reqChangeVote", reqChangeVote)
		// Don't skip if this link is a request to change the vote
		if reqChangeVote == "" {
			// Check if user has already voted in this poll, and if so, take them to view the poll results.
			if DbExists("SELECT * FROM $$PollVote WHERE UserId=$1 AND PollId=$2", userId, postId) {
				http.Redirect(w, r, fmt.Sprintf("/viewPollResults/?postId=%d", postId), http.StatusSeeOther)
				return
			}
		}

		moreArticles = fetchSuggestedPolls(userId, postId)
	} else {
		moreArticles = fetchArticlesFromThisNewsSource(article.NewsSourceId, userId, postId)
	}

	prVal("len(moreArticles)", len(moreArticles))

	prVal("len(concated articles)", len(append(moreArticles, article)))

	upvotes, downvotes := deduceVotingArrows(append(moreArticles, article))

	headComment, upcommentvotes, downcommentvotes := ReadCommentsFromDB(article.Id, userId)

	prVal("upvotes", upvotes)
	prVal("downvotes", downvotes)
	prVal("upcommentvotes", upcommentvotes)
	prVal("downcommentvotes", downcommentvotes)

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
		UpCommentVotes	[]int64
		DownCommentVotes []int64
		HeadComment		Comment
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
		UpCommentVotes:	upcommentvotes,
		DownCommentVotes: downcommentvotes,
		HeadComment:	headComment,
		ArticleGroups:	articleGroups,
		MoreArticlesFromThisSource:	moreArticles,
	}

	executeTemplate(w, kArticle, articleArgs)
}
