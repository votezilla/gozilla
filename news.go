package main

import (
	"net/http"
	"math/rand"
	"sort"
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
	kSubmittedPosts = "created polls & posts"
	kCommentedPosts = "commented posts"
	kVotedPosts = "up/down voted posts"

	kNoHeadlines = 0
	kAlternateHeadlines = 1
	kAllHeadlines = 2
)

var (
	newsSourceList = map[string]bool{}

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
			"votezilla",
			"other",
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
			"votezilla"			: "#fa2",
			"other"				: "#4af392",
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
			{"votezilla",		"votezilla"},
			{"other",			"other"},
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

	//prVal("formatArticleGroups   onlyCategory", onlyCategory)

	//prVal("len(articles)", len(articles))
	//for _, article := range articles {
	//	prVal("article.Category", article.Category)
	//}

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
					//prVal("category", category)
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

	// Prune empty article groups.
	for g := len(articleGroups) - 1; g >= 0; g-- {
		//prVal("g", g)
		numArticles := 0
		for _, articleRow := range articleGroups[g].Articles {
			//prf("  len(articleRow) %d", len(articleRow))

			for _, article := range articleRow {
				//prf("    article.Title", article.Title)
				if article.Title != "" {
					numArticles++
				}
			}
		}
		//prf("For g = %d, numArticles = %d", g, numArticles)

		if numArticles == 0 {
			//prf("Deleting ArticleGroup g", g)
			articleGroups = append(articleGroups[:g], articleGroups[g+1:]...) // Delete this empty article group.
		}
	}

	return articleGroups
}

//////////////////////////////////////////////////////////////////////////////
//
// Deduce voting arrows - for articles you have voted on
//
//////////////////////////////////////////////////////////////////////////////
func deduceVotingArrows(articles []Article) (upvotes []int64, downvotes []int64) {
	//prVal("deduceVotingArrows len(articles)", len(articles))

	for _, article := range articles {

		if article.Upvoted == 1 {
			upvotes = append(upvotes, article.Id)
		} else if article.Upvoted == -1 {
			downvotes = append(downvotes, article.Id)
		}
	}

	//prVal("upvotes", upvotes)
	//prVal("downvotes", downvotes)

	return upvotes, downvotes
}

//////////////////////////////////////////////////////////////////////////////
//
// Render a news template
//
//////////////////////////////////////////////////////////////////////////////
func renderNews(w http.ResponseWriter,
				r *http.Request,
				title,
				username string,
				userId int64,
				viewUsername string,
				articleGroups []ArticleGroup,
				urlPath,
				template string,
				upvotes,
				downvotes []int64,
				category,
				alertMessage string) {
	// REVERT!!!
	//username = "AVeryLongUsername5526dks3232W"

	pr("renderNews")
	prVal("  username", username)
	prVal("  userId", userId)
	prVal("  category", category)
	prVal("  alertMessage", alertMessage)

	title = "votezilla - " + title

	// TODO: use a cookie to only alert about being logged in once?  Also, make this into a pop-up.
	script := alertMessage

	_, isNewsSource := newsSourceList[viewUsername]

	// Render the news articles.
	newsArgs := struct {
		FrameArgs
		ArticleGroups	[]ArticleGroup
		Category		string
		ViewUsername	string
		IsNewsSource	bool
	}{
		FrameArgs:		makeFrameArgs2(r, title, script, urlPath, userId, username, upvotes, downvotes),
		ArticleGroups:	articleGroups,
		Category:		category,
		ViewUsername:	viewUsername,
		IsNewsSource:	isNewsSource,
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
	startTimer("newsHandler")

	startTimer("A")
	RefreshSession(w, r)

	pr("newsHandler")

	reqCategory		:= parseUrlParam(r, "category")
	reqAlert		:= parseUrlParam(r, "alert")

	prVal("reqCategory", reqCategory)
	prVal("newsCategoryInfo.HeaderColors", newsCategoryInfo.HeaderColors)

	if reqCategory != "" {
		_, foundCategory := newsCategoryInfo.HeaderColors[reqCategory]  // Ensure we have a valid category (to prevent SQL injection).
		if !foundCategory {
			reqAlert = "InvalidCategory"
			reqCategory = ""
		}
	}
	endTimer("A")

	startTimer("B")
	userId, username := GetSessionInfo(w, r)
	endTimer("B")

	startTimer("fetchArticles")
	var articles []Article
	if reqCategory == "" { // /news
		pr("  a")

		// If logged in, cycle between 3 materialized tables; if not logged in, pick one at random.
		var newsCycle int
		//if userId >= 0 {
		//	switch GetCookie(r, "newsCycle", "0") {
		//		case "0": newsCycle = 1; SetCookie(w, r, "newsCycle", "1"); break
		//		case "1": newsCycle = 2; SetCookie(w, r, "newsCycle", "2"); break
		//		default : newsCycle = 0; SetCookie(w, r, "newsCycle", "0"); break
		//	}
		//} else {
			newsCycle = rand.Intn(3)
		//}
		prVal("newsCycle", newsCycle)

		// Fetch 5 articles from each category
		articles = fetchNews(kRowsPerCategory + 1, userId, kMaxArticles, newsCycle) // kRowsPerCategory on one side, and 1 headline on the other.
	} else if reqCategory == "polls" {
		pr("  b")
		// Fetch only polls.
		articles = fetchPolls(userId, kMaxArticles)
		prVal("len(articles)", len(articles))
	} else {
		pr("  c")
		// Fetch articles in requested category
		articles = fetchArticlesWithinCategory(reqCategory, userId, kMaxArticles)
	}
	endTimer("fetchArticles")

	startTimer("deduceVotingArrows")
	upvotes, downvotes := deduceVotingArrows(articles)
	endTimer("deduceVotingArrows")

	startTimer("formatArticleGroups")
	articleGroups := formatArticleGroups(articles, newsCategoryInfo, reqCategory, kAlternateHeadlines)
	endTimer("formatArticleGroups")

	// vv WORKS! - TODO_OPT: fix so poll don't require an extra db query

	if reqCategory == "" { // /news
		startTimer("formatPolls")

		// Reformat the polls to have 6 visible, with no headlines.  (Polls with headlines waste a lot of space.)
		pollArticleGroups := formatArticleGroups(articles, newsCategoryInfo, "polls", kNoHeadlines)

		//prVal("articleGroups[0]", articleGroups[0])
		//prVal("pollArticleGroups[0]", pollArticleGroups[0])
		articleGroups[0] = pollArticleGroups[0]  // Try copying the polls, as a test
		articleGroups[0].More = "polls"

		endTimer("formatPolls")
	}

	startTimer("renderNews")
	renderNews(w, r, "News", username, userId, "", articleGroups, "news", kNews, upvotes, downvotes, reqCategory, reqAlert)
	endTimer("renderNews")

	endTimer("newsHandler")
}



func InitNewsSources() {
	rows := DbQuery("SELECT NewsSourceId FROM $$NewsPost GROUP BY 1 ORDER BY 1")
	for rows.Next() {
		newsSource := ""
		err := rows.Scan(&newsSource)
		check(err)

		newsSourceList[newsSource] = true
	}
	check(rows.Err())
	rows.Close()
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
		PageArgs: makePageArgs(r, "News Sources", "", ""),
		NewsSources: newsSources,
	}
	fmt.Println("newsSourcesArgs: %#v\n", newsSourcesArgs)
	executeTemplate(w, kNewsSources, newsSourcesArgs)
}
*/