//[] Use Colly for web scraping: https://github.com/gocolly/colly
//[] ++ Fox News
//[] Scroll keeps bringing up new topics.  When you reach bottom, infinite scroll has MORE NEWS, MORE WORLD NEWS, etc.
//   Then EVEN MORE NEWS, YET EVEN MORE NEWS, YET EVEN MORE MORE NEWS, OODLES OF NEWS, etc.  Kind of humorous.

package main

import (
	"net/http"
	//"math/rand"
	"sort"
	//"strconv"
	//"net/url"
)

// A group of articles, separated by a header.
type ArticleGroup struct {
	Articles		[][]Article // 2 columns, 3 Articles each.
	Category		string
	HeaderColor		string
	HeadlineSide	int		// 0=left, 1=right (On large, i.e. non-mobile, devices...)
	More			string	// category if "More..." appears at end of group.
}

type CategoryInfo struct {
	CategoryOrder	[]string
	HeaderColors	map[string]string
	CategorySelect	[][2]string		// for forms
}

const (
	kNumCols = 2
	kRowsPerCategory = 3 //4
	kMaxArticles = 100 // 250 //120 //60
	kMaxTitleLength = 122

	kSubmittedPolls = "submitted polls"
	kVotedPolls = "voted polls"
	kSubmittedPosts = "created posts"
	kCommentedPosts = "commented posts"
	kVotedPosts = "up/down voted posts"

	kNoHeadlines = 0
	kAlternateHeadlines = 1
	kAllHeadlines = 2
)

var (
	newsCategoryInfo = CategoryInfo {
		CategoryOrder : []string{
			"polls",
			"news",
			"world news",
			"business",
			"sports",
			"entertainment",
			"technology",
			"science",
//			"other",
		},
		HeaderColors : map[string]string{
			"polls"				: "#4482ff", //"#fe8",
			"news" 			 	: "#68fc48", //#68fc68", //#7ff4f4",    //"#8ff",
			"world news"		: "#ea3ce7",
			"business" 			: "#8e8",
			"sports" 			: "#88f",
			"entertainment" 	: "#e85be4",
			"technology" 		: "#aaa",    //"#ccc",
			"science"			: "#8cf",
//			"other"				: "#4af392",
		},
		CategorySelect : [][2]string{
			{"polls", 			"polls"},
			{"news", 			"news"},
			{"world news",		"world news"},
			{"business",		"business"},
			{"sports",			"sports"},
			{"entertainment",	"entertainment"},
			{"technology",		"technology"},
			{"science",			"science"},
//			{"other",			"other"},
		},
	}

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

//////////////////////////////////////////////////////////////////////////////
//
// TODO: santize (html- and url-escape the arguments).
//       (Make sure URL's don't point back to votezilla.)
//		 possibly based on whether mobile, and whether a headline.
//
//////////////////////////////////////////////////////////////////////////////
func formatArticle(article *Article) {
	// Truncate the title if it's too long.
	article.Title = ellipsify(article.Title, kMaxTitleLength)
}

//////////////////////////////////////////////////////////////////////////////
//
// Sort articles
//
//////////////////////////////////////////////////////////////////////////////
func sortArticles(articles []Article) {
	// Sort by category, then by how recently it was published.
	sort.Slice(articles, func(i, j int) bool {
		return articles[i].Category < articles[j].Category ||
	  		   (articles[i].Category == articles[j].Category &&
	  		    articles[i].PublishedAtUnix.After(articles[j].PublishedAtUnix))
	})
}

//////////////////////////////////////////////////////////////////////////////
//
// Format article groups - take an array of articles, arrange it into article groups
//                         for display on /news.
//	 categoryInfo - describes the category names and banner background colors.
//	 onlyCategory - if == "", displays for articles grouped by category
//				       != "", only display articles from a specific category
//   headlines    - whether to display some articles as headlines (larger articles):
//		    		kNoHeadlines, kAlternateHeadlines, or kAllHeadlines.
//
//////////////////////////////////////////////////////////////////////////////
func formatArticleGroups(articles []Article, categoryInfo CategoryInfo, onlyCategory string, headlines int) ([]ArticleGroup) {
	//rowsPerCategory := ternary_int(onlyCategory == "", kRowsPerCategory, kMaxArticles)

	pr("formatArticleGroups")

	var categoryOrder []string
	if onlyCategory != "" {
		//prVal("headlines", headlines)
		articlesPerCategoryGroup :=
			switch_int(headlines,
				kNoHeadlines,		 kRowsPerCategory * kNumCols,
				kAlternateHeadlines, kRowsPerCategory + 1,
				kAllHeadlines, 		 2)
		//prVal("articlesPerCategoryGroup", articlesPerCategoryGroup)

		assert(articlesPerCategoryGroup != -1)

		numCategoryGroups := kMaxArticles / articlesPerCategoryGroup

		categoryOrder = make([]string, numCategoryGroups)
		for i := range categoryOrder {
			categoryOrder[i] = onlyCategory
		}
	} else {
		categoryOrder = categoryInfo.CategoryOrder
	}


	articleGroups := make([]ArticleGroup, len(categoryOrder))

	cat := 0
	headlineSide := 0 // The side that has the headline (large article).
	currArticle := 0
	for ccc, category := range categoryOrder {
		row := 0
		col := 0
		filled := false


		// Set category header text and background color.
		if onlyCategory == "" { // Mixed categories
			articleGroups[cat].Category = category
			articleGroups[cat].More = category
		} else { 			   // Single category
			category = onlyCategory // Make all categories the same
			// Only the first articleGroup has a category name, the rest have "",
			// which is a flag to have no category header.
			if ccc == 0 {
				articleGroups[cat].Category = category
			} else {
				articleGroups[cat].Category = ""
			}
			articleGroups[cat].More = ""
		}

		articleGroups[cat].HeaderColor = categoryInfo.HeaderColors[category]
		articleGroups[cat].HeadlineSide = headlineSide

		// Mixed categories - causing all articles to reiterate, but it will test against the category later.
		if onlyCategory == "" {
			currArticle = 0
		}

		// TODO: if a single category, with headlines, either large image should be set to always
		// 4 article height, or all articles should stack verticlally in each column.
		// (I prefer the second idea, because it might look nicer.)

		for currArticle < len(articles) {
			article := articles[currArticle]
			currArticle++


			formatArticle(&article)


			// This works since we've sorted by bucket/category.
			if coalesce_str(article.Bucket, article.Category) == category {

				if row == 0 {
					// Allocate a new column of categories
					articleGroups[cat].Articles = append(articleGroups[cat].Articles,
														 make([]Article, kRowsPerCategory))
				}

				// The first article is always the headline.  Articles after the headline get skipped.
				size := 0


				if headlines != kNoHeadlines {
					if col == 0 || headlines == kAllHeadlines {
						if row == 0 { // first article is the headline, i.e. big
							size =  1 // 1 means large article (headline)
						} else {      // the rest of the articles get skipped, since the headline takes all the space.
							size = -1 // -1 means skip the article
							currArticle-- // don't skip the article, since the slot is skipped
										  // TODO: there's a bug where a headline article gets displayed twice, if we're in a specific category!!!
						}
					}
				}

				// Assign this slot the next article, as long as this is not an empty slot.  Make sure size gets assigned!
				if size == -1 {
					articleGroups[cat].Articles[col][row].Size = -1
				} else {
					articleGroups[cat].Articles[col][row] = article
					articleGroups[cat].Articles[col][row].Size = size

					// HACK: make polls never be headlines.  TODO: Clean this up.
					prVal("category", category)
					if category == "polls" {
						articleGroups[cat].Articles[col][row].Size = 0
					}
				}

				// Inc row, col
				col++
				if col == kNumCols {
					col = 0
					row++

					if row == kRowsPerCategory {
						filled = true
						break
					}
				}
			}
		}

		// If we ran out of articles, skip the rest
		for !filled {
			if row == 0 {
				// Make room for new row
				articleGroups[cat].Articles = append(articleGroups[cat].Articles,
													 make([]Article, kRowsPerCategory))
			}

			articleGroups[cat].Articles[col][row].Size = -1 // -1 means skip the article

			// Inc row, col
			col++
			if col == kNumCols {
				col = 0
				row++

				if row == kRowsPerCategory {
					filled = true
					break
				}
			}
		}

		cat++

		if headlines == kAlternateHeadlines {
			headlineSide = (headlineSide + 1) % 2 // The side with the headline switches each time, to look nice.
		}
	}

	// If a single category, only the last articleGroup should have a "More..." link.
	if onlyCategory != "" {
		articleGroups[cat - 1].More = onlyCategory
	}

	return articleGroups
}

//////////////////////////////////////////////////////////////////////////////
//
// Deduce voting arrows - for articles you have voted on
//
//////////////////////////////////////////////////////////////////////////////
func deduceVotingArrows(articles []Article) (upvotes []int64, downvotes []int64) {
	prVal("deduceVotingArrows len(articles)", len(articles))

	for _, article := range articles {

		if article.Upvoted == 1 {
			upvotes = append(upvotes, article.Id)
		} else if article.Upvoted == -1 {
			downvotes = append(downvotes, article.Id)
		}
	}

	prVal("upvotes", upvotes)
	prVal("downvotes", downvotes)

	return upvotes, downvotes
}

//////////////////////////////////////////////////////////////////////////////
//
// Render a news template
//
//////////////////////////////////////////////////////////////////////////////
func renderNews(w http.ResponseWriter,
				title, username string,
				userId int64,
				articleGroups []ArticleGroup,
				urlPath, template string,
				upvotes, downvotes []int64,
				category, alertMessage string) {

	pr("renderNews")
	prVal("  username", username)
	prVal("  userId", userId)

	title = "votezilla - " + title

	// TODO: use a cookie to only alert about being logged in once?
	script := ""
	switch(alertMessage) {
		case "LoggedIn": 		script = "alert('You are now logged in :)')"
		case "LoggedOut": 		script = "alert('You are now logged out :)')"
		case "AccountCreated": 	script = "alert('Your account has been created, good work!')"
		case "SubmittedLink": 	script = "alert('Your link has been created, and will appear shortly')"
		case "SubmittedPoll": 	script = "alert('Your poll has been created, and will appear shortly')"
	}

	// Render the news articles.
	newsArgs := struct {
		FrameArgs
		ArticleGroups	[]ArticleGroup
		Category		string
	}{
		FrameArgs:		makeFrameArgs2(title, script, urlPath, userId, username, upvotes, downvotes),
		ArticleGroups:	articleGroups,
		Category:		category,
	}

	//prVal("UpVotes", upvotes)
	//prVal("DownVotes", downvotes)

	executeTemplate(w, template, newsArgs)
}

//////////////////////////////////////////////////////////////////////////////
//
// News handler
//
//////////////////////////////////////////////////////////////////////////////
func newsHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)

	pr("newsHandler")

	reqCategory		:= parseUrlParam(r, "category")
	reqAlert		:= parseUrlParam(r, "alert")

	_, foundCategory := newsCategoryInfo.HeaderColors[reqCategory]  // Ensure we have a valid category (to prevent SQL injection).
	if !foundCategory {
		reqAlert = "Invalid category; displaying all news posts"
		reqCategory = ""
	}

	userId, username := GetSessionInfo(w, r)

	var articles []Article
	if reqCategory == "" { // /news
		// Fetch 5 articles from each category
		articles = fetchArticlesPartitionedByCategory(kRowsPerCategory + 1, userId, kMaxArticles) // kRowsPerCategory on one side, and 1 headline on the other.
	} else if reqCategory == "polls" {
		// Fetch only polls.
		articles = fetchPolls(userId, kMaxArticles)
		prVal("len(articles)", len(articles))
	} else {
		// Fetch articles in requested category
		articles = fetchArticlesWithinCategory(reqCategory, userId, kMaxArticles)
	}

	upvotes, downvotes := deduceVotingArrows(articles)

	articleGroups := formatArticleGroups(articles, newsCategoryInfo, reqCategory, kAlternateHeadlines)

	// vv WORKS! - TODO_OPT: fix so poll don't require an extra db query
	polls := fetchPolls(userId, 2 * kRowsPerCategory)
	pollArticleGroups := formatArticleGroups(polls, newsCategoryInfo, "polls", kNoHeadlines)
	
	if reqCategory == "" { // /news
		//prVal("articleGroups[0]", articleGroups[0])
		//prVal("pollArticleGroups[0]", pollArticleGroups[0])
		articleGroups[0] = pollArticleGroups[0]  // Try copying the polls, as a test
		articleGroups[0].More = "polls"
	}

	renderNews(w, "News", username, userId, articleGroups, "news", kNews, upvotes, downvotes, reqCategory, reqAlert)
}

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

	userId, username := GetSessionInfo(w, r)

	articleGroups := []ArticleGroup{}
	allArticles := []Article{}
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
		articles := fetchPollsPostedByUser(userId, kNumCols * kRowsPerCategory)
		//articles = removeDupIds(articles)

		allArticles = append(allArticles, articles...)

		for a := range articles {
			articles[a].Bucket = kSubmittedPolls
		}

		articleGroups = append(articleGroups,
			formatArticleGroups(articles, historyCategoryInfo, kSubmittedPolls, kNoHeadlines)...)
	}
*/
	pr("Get polls voted on by user")
	{
		articles := fetchPollsVotedOnByUser(userId, kNumCols * kRowsPerCategory)
		//articles = removeDupIds(articles)

		allArticles = append(allArticles, articles...)

		for a := range articles {
			articles[a].Bucket = kVotedPolls
		}

		articleGroups = append(articleGroups,
			formatArticleGroups(articles, historyCategoryInfo, kVotedPolls, kNoHeadlines)...)
	}

	pr("Get articles posted by user")
	{
		articles := fetchArticlesPostedByUser(userId, kNumCols * kRowsPerCategory)
		//articles = removeDupIds(articles)

		allArticles = append(allArticles, articles...)

		for a := range articles {
			articles[a].Bucket = kSubmittedPosts
		}

		articleGroups = append(articleGroups,
			formatArticleGroups(articles, historyCategoryInfo, kSubmittedPosts, kNoHeadlines)...)
	}

	pr("Get articles commented on by user")
	{
		articles := fetchArticlesCommentedOnByUser(userId, kNumCols * kRowsPerCategory)
		//articles = removeDupIds(articles)

		allArticles = append(allArticles, articles...)

		for a := range articles {
			articles[a].Bucket = kCommentedPosts
		}

		articleGroups = append(articleGroups,
			formatArticleGroups(articles, historyCategoryInfo, kCommentedPosts, kNoHeadlines)...)
	}

	pr("Get articles voted on by user, and set their bucket accordingly.")
	{
		articles := fetchArticlesUpDownVotedOnByUser(userId, kNumCols * kRowsPerCategory)
		//articles = removeDupIds(articles)

		allArticles = append(allArticles, articles...)

		for a := range articles {
			articles[a].Bucket = kVotedPosts
		}

		articleGroups = append(articleGroups,
			formatArticleGroups(articles, historyCategoryInfo, kVotedPosts, kNoHeadlines)...)
	}

	upvotes, downvotes := deduceVotingArrows(allArticles)

	prVal("upvotes", upvotes)
	prVal("downvotes", downvotes)

	for g, _ := range articleGroups {
		articleGroups[g].More = "" // Clear any "More ___..." links, since they don't lead anywhere yet.
	}


	// Render the history just like we render the news.
	renderNews(w, "History", username, userId, articleGroups, "history", kNews, upvotes, downvotes, "", reqAlert)
}


///////////////////////////////////////////////////////////////////////////////
//
// display news sources - TODO: checkboxes so user can pick
//                        which news sources they want to see.
//
///////////////////////////////////////////////////////////////////////////////
/*
func newsSourcesHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)

	newsSourcesArgs := struct {
		PageArgs
		NewsSources NewsSources
	}{
		PageArgs: PageArgs{Title: "News Sources"},
		NewsSources: newsSources,
	}
	fmt.Println("newsSourcesArgs: %#v\n", newsSourcesArgs)
	executeTemplate(w, kNewsSources, newsSourcesArgs)
}
*/