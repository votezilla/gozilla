//[] Scroll keeps bringing up new topics.  When you reach bottom, infinite scroll has MORE NEWS, MORE WORLD NEWS, etc.
//   Then EVEN MORE NEWS, etc.  Kind of humorous.

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
	Articles		[][]Article // Arrow of rows, each row has 2 articles.
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
	kRowsPerCategory = 4
	kMaxArticles = 60
	kMaxTitleLength = 122

	kSubmittedPosts = "created posts"
	kCommentedPosts = "commented posts"
	kVotedPosts = "voted posts"

	kNoHeadlines = 0
	kAlternateHeadlines = 1
	kAllHeadlines = 2
)

var (
	newsCategoryInfo = CategoryInfo {
		CategoryOrder : []string{
			"news",
			"world news",
			"business",
			"sports",
			"entertainment",
			"technology",
			"science",
		},
		HeaderColors : map[string]string{
			"news" 			 	: "#ccc",
			"world news"		: "#ea3ce7",
			"business" 			: "#8e8",
			"sports" 			: "#88f",
			"entertainment" 	: "#e85be4",
			"technology" 		: "#8ff",
			"science"			: "#8cf",
		},
		CategorySelect : [][2]string{
			{"news", 			"news"},
			{"world news",		"world news"},
			{"business",		"business"},
			{"sports",			"sports"},
			{"entertainment",	"entertainment"},
			{"technology",		"technology"},
			{"science",			"science"},
		},
	}

	historyCategoryInfo = CategoryInfo {
		CategoryOrder : []string{
			kCommentedPosts,
			kSubmittedPosts,
			kVotedPosts,
		},
		HeaderColors : map[string]string{
			kCommentedPosts: "#f90",
			kSubmittedPosts : "#aaf",
			kVotedPosts : "#ccc",
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
	if len(article.Title) > kMaxTitleLength {
		article.Title = article.Title[0:kMaxTitleLength] + "..."
	}
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
//                         for display on the webpage.
//	 categoryInfo - describes the category names and banner background colors.
//	 onlyCategory - if == "", displays for articles grouped by category
//				       != "", only display articles from a specific category
//   headlines    - whether to display some articles as headlines (larger articles):
//		    kNoHeadlines, kAlternateHeadlines, or kAllHeadlines.
//
// TODO: HTML-escape this!!!
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

					//articleGroups[cat].Articles[col][row].Title =
					//	articleGroups[cat].Articles[col][row].Title[0:29] + " " + strconv.Itoa(row) + " " + strconv.Itoa(col) + " " + strconv.Itoa(currArticle)
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
	for a, article := range articles {
		articles[a].Bucket = kVotedPosts

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
func renderNews(w http.ResponseWriter, title string, username string, userId int64,
				articleGroups []ArticleGroup, urlPath string, template string,
				upvotes []int64, downvotes []int64, alertMessage string) {

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
		PageArgs
		Username		string
		UserId			int64
		ArticleGroups	[]ArticleGroup
		NavMenu			[]string
		UrlPath			string
		UpVotes			[]int64
		DownVotes		[]int64
	}{
		PageArgs:		PageArgs{Title: "votezilla - " + title, Script: script},
		Username:		username,
		UserId:			userId,
		ArticleGroups:	articleGroups,
		NavMenu:		navMenu,
		UrlPath:		urlPath,
		UpVotes:		upvotes,
		DownVotes:		downvotes,
	}

	executeTemplate(w, template, newsArgs)
}

//////////////////////////////////////////////////////////////////////////////
//
// News handler
// TODO: santize (html- and url-escape the arguments).
//       (Make sure URL's don't point back to votezilla.)
//
//////////////////////////////////////////////////////////////////////////////
func newsHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)

	//prVal("r.URL.Query()", r.URL.Query())

	reqCategory		:= parseUrlParam(r, "category")

	reqAlert		:= parseUrlParam(r, "alert")

	// Get the username.
	userId := GetSession(r)
	username := getUsername(userId)

	// TODO: cache this, fetch every minute?
	var articles []Article
	if reqCategory == "" {
		// Fetch 5 articles from each category
		articles = fetchArticlesPartitionedByCategory(kRowsPerCategory + 1, userId, kMaxArticles) // kRowsPerCategory on one side, and 1 headline on the other.
	} else {
		// Ensure we have a valid category (prevent SQL injection)
		if _, ok := newsCategoryInfo.HeaderColors[reqCategory]; !ok {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Fetch articles in requested category
		articles = fetchArticlesWithinCategory(reqCategory, userId, kMaxArticles)
	}

	articleGroups := formatArticleGroups(articles, newsCategoryInfo, reqCategory, kAlternateHeadlines)

	renderNews(w, "News", username, userId, articleGroups, "news", "news", []int64{}, []int64{}, reqAlert)
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

	// Get the username.
	userId := GetSession(r)
	username := getUsername(userId)

	articleGroups := []ArticleGroup{}

	// Get articles posted by user
	pr("Get articles posted by user")
	{
		articles := fetchArticlesPostedByUser(userId, kNumCols * kRowsPerCategory)

		for a, _ := range articles {
			articles[a].Bucket = kSubmittedPosts
		}

		articleGroups = append(articleGroups,
			formatArticleGroups(articles, historyCategoryInfo, kSubmittedPosts, kNoHeadlines)...)
	}

	// Get articles commented on by user
	pr("Get articles commented on by user")
	{
		articles := fetchArticlesCommentedOnByUser(userId, kNumCols * kRowsPerCategory)

		for a, _ := range articles {
			articles[a].Bucket = kCommentedPosts
		}

		articleGroups = append(articleGroups,
			formatArticleGroups(articles, historyCategoryInfo, kCommentedPosts, kNoHeadlines)...)
	}

	// Get articles up/down voted on by user, and set their bucket accordingly.
	var upvotes, downvotes []int64
	pr("Get articles voted on by user, and set their bucket accordingly.")
	{
		articles := fetchArticlesUpDownVotedOnByUser(userId, kNumCols * kRowsPerCategory)

		upvotes, downvotes = deduceVotingArrows(articles)

		prVal("upvotes", upvotes)
		prVal("downvotes", downvotes)

		articleGroups = append(articleGroups,
			formatArticleGroups(articles, historyCategoryInfo, kVotedPosts, kNoHeadlines)...)
	}


	// Render the history just like we render the news.
	renderNews(w, "History", username, userId, articleGroups, "history", "news", upvotes, downvotes, reqAlert)
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