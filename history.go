package main

import (
	"net/http"
)

var (
	historyCategoryInfo = CategoryInfo {
		CategoryOrder : []string{
			kCommentedPosts,
			kSubmittedPosts,
			kVotedPosts,
			kSubmittedPolls,
			kVotedPolls,
		},
		HeaderColors : map[string]string{
			kCommentedPosts: "#f90",
			kSubmittedPosts : "#aaf",
			kVotedPosts : "#ccc",
			kSubmittedPolls : "#da8",
			kVotedPolls : "#fa9",
		},
	}
)

///////////////////////////////////////////////////////////////////////////////
//
// History handler - of user posts, up/down votes,
//                   TODO: comments
//
///////////////////////////////////////////////////////////////////////////////
func historyHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)

	pr("historyHandler")

	//prVal("r.URL.Query()", r.URL.Query())

	reqAlert		:= parseUrlParam(r, "alert")

	// User is current (logged in) user by default.
	userId, username := GetSessionInfo(w, r)
	isMe 			 := true
	viewUserId		 := userId
	viewUsername	 := ""  // "" if viewing ourselves, and if we are viewing someone else it is their username.

	//TODO: do something appropriate if user is not logged in

	// If we request a different user, query the userId we are viewing
	reqUsername := parseUrlParam(r, "username")
	prVal("reqUsername", reqUsername)
	newsSource := ""
	if reqUsername != "" && reqUsername != username {
		viewUsername = reqUsername
		prVal("viewUsername", viewUsername)

		// Check if it's a news source.
		_, found := newsSourceList[viewUsername]
		if found {
			isMe = false
			newsSource = viewUsername
		} else {  // Else, check if it's a username.  (TODO: disallow usernames to match news sources during registration.)
			pr("Valid requested username")
			viewUserId = UsernameToUserId(viewUsername)

			prVal("viewUserId", viewUserId)
			if viewUserId == -1 {
				serveError(w, "User not found")
				return
			}

			isMe = false // We're viewing another user's history.
		}
	}

	prVal("isMe", isMe)

/*
	var userId		int64
	var username 	string
	var isMe		bool
	{
		// User is current (logged in) user by default.
		userId, username := GetSessionInfo(w, r)
		isMe = true

		// If we request a different user, update the user info to match.
		reqUserId := parseUrlParam(r, "userId")
		if reqUserId != "" {
			newUserId, err := str_to_int64(reqUserId)
			check(err)
			if newUserId != userId {
				username	= GetUsername(userId)
				if username != "" { // If user is found...
					isMe		= false
					userId		= userId
				}
			}
		}
	}
*/

	articleGroups := []ArticleGroup{}
	allArticles := []Article{}


	if newsSource != "" {
		allArticles = fetchArticlesFromThisNewsSource(newsSource, userId, -1)

		prVal("len(allArticles)", len(allArticles))
		//for a := range allArticles {
		//	allArticles[a].Bucket = newsSource
		//}

		// We need to determine the news category, otherwise only a single row of news gets displayed due to
		//   formatArticleGroups hackiness.
		// All articles from each news source just get labeled with a single cat, so we can
		//   tell what the category is by just examining a single article.
		newsCat := ""
		if len(allArticles) > 0 {
			newsCat = allArticles[0].Category
		}
		prVal("newsCat", newsCat)

		articleGroups = formatArticleGroups(allArticles, newsCategoryInfo, newsCat, kAlternateHeadlines)
	} else {
	/*	dupIds := map[int64]bool{}

		removeDupIds := func(articles []Article) (filteredArticles []Article) {
			numAddedArticles := 0
			for _, article := range articles {

				// If duplicate id exists, purge the article.
				//_, found := dupIds[article.Id]
				//if !found {
					filteredArticles = append(filteredArticles, article)
					dupIds[article.Id] = true
					numAddedArticles++

				//	if numAddedArticles >= 6 {
				//		break
				//	}
				//}
			}
			return
		}
	*/
	/*
		pr("Get polls posted by user") // << Removed because it's a subset of "articles posted by user".
		{
			articles := fetchPollsPostedByUser(viewUserId, userId, kNumCols * kRowsPerCategory)
			//articles = removeDupIds(articles)

			allArticles = append(allArticles, articles...)

			for a := range articles {
				articles[a].Bucket = kSubmittedPolls
			}

			articleGroups = append(articleGroups,
				formatArticleGroups(articles, historyCategoryInfo, kSubmittedPolls, kNoHeadlines)...)
		}
	*/


		{
			pr("Get articles posted by user")

			articles := fetchArticlesPostedByUser(viewUserId, userId, kNumCols * kRowsPerCategory)
			//articles = removeDupIds(articles)

			allArticles = append(allArticles, articles...)

			for a := range articles {
				articles[a].Bucket = kSubmittedPosts
			}

			articleGroups = append(articleGroups,
				formatArticleGroups(articles, historyCategoryInfo, kSubmittedPosts, kNoHeadlines)...)
		}

		if isMe {
			pr("Get polls voted on by user")

			articles := fetchPollsVotedOnByUser(viewUserId, userId, kNumCols * kRowsPerCategory)
			//articles = removeDupIds(articles)

			allArticles = append(allArticles, articles...)

			for a := range articles {
				articles[a].Bucket = kVotedPolls
			}

			articleGroups = append(articleGroups,
				formatArticleGroups(articles, historyCategoryInfo, kVotedPolls, kNoHeadlines)...)
		}

		{
			pr("Get articles commented on by user")

			articles := fetchArticlesCommentedOnByUser(viewUserId, userId, kNumCols * kRowsPerCategory)
			//articles = removeDupIds(articles)

			allArticles = append(allArticles, articles...)

			for a := range articles {
				articles[a].Bucket = kCommentedPosts
			}

			articleGroups = append(articleGroups,
				formatArticleGroups(articles, historyCategoryInfo, kCommentedPosts, kNoHeadlines)...)
		}


		if isMe {
			pr("Get articles voted on by user, and set their bucket accordingly.")

			articles := fetchArticlesUpDownVotedOnByUser(viewUserId, userId, kNumCols * kRowsPerCategory)
			//articles = removeDupIds(articles)

			allArticles = append(allArticles, articles...)

			for a := range articles {
				articles[a].Bucket = kVotedPosts
			}

			articleGroups = append(articleGroups,
				formatArticleGroups(articles, historyCategoryInfo, kVotedPosts, kAlternateHeadlines)...)
		}

		for g, _ := range articleGroups {
			articleGroups[g].More = "" // Clear any "More ___..." links, since they don't lead anywhere yet.
		}
	}

	upvotes, downvotes := deduceVotingArrows(allArticles)

	// Render the history just like we render the news.
	renderNews(w, "History", username, userId, viewUsername, articleGroups, "history", kNews, upvotes, downvotes, "", reqAlert)
}


