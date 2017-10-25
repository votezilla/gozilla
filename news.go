package main

import (
	"net/http"
	"sort"
	"sync"
)

// For rendering the news article information.
type ArticleArg struct {
	Article
	Size			int		// 0=normal, 1=large, 2=x-large
}

type ArticleGroup struct {
	ArticleArgs		[][]ArticleArg // Arrow of rows, each row has 2 articles.
	Category		string
	HeaderColor		string
	BgColor			string
	HeadlineSide	int
}

// A news source to request the news from.
// TODO: turn NewsSource into a table as well?
type NewsSource struct {
	Id					string
	Name				string
	Description			string
	Url					string
	Category			string
	Language			string
	Country				string
	SortBysAvailable	[]string
	// Custom parameters:
	Icon				string
}
type NewsSources map[string]NewsSource

var (
	// newsServer populates the articles.
	mutex = &sync.RWMutex{}
	articles []Article
	
	// Custom-written data from https://newsapi.org/v1/sources?language=en query
	newsSources NewsSources
	
	categoryOrder = []string{
		"politics", 			
		"general", 			
		"business", 			
		"sport", 			
		"entertainment", 	
		"science-and-nature",
		"technology",		
		"gaming",			
		"music", 			
	}
	
	headerColors map[string]string = map[string]string{
		"politics" 			: "#aaa",
		"general" 			: "#ccc",
		"business" 			: "#8e8",
		"sport" 			: "#88f",
		"entertainment" 	: "#e85be4",
		"science-and-nature": "#8cf",
		"technology" 		: "#8ff",
		"gaming" 			: "#58d858",
		"music" 			: "#fd8",
	}

	bgColors map[string]string = map[string]string{
		"politics" 			: "#c7c6c6",
		"general" 			: "#ddd",
		"business" 			: "#b2fdb2",
		"sport" 			: "#bbf",
		"entertainment" 	: "#fda5fd",
		"science-and-nature": "#bdf",
		"technology" 		: "#bff",
		"gaming" 			: "#afa",
		"music" 			: "#feb",
	}
)

//////////////////////////////////////////////////////////////////////////////
//
// display news
// TODO: santize (html- and url-escape the arguments).  (Make sure URL's don't point back to votezilla.)
// TODO: use a caching, resizing image proxy for the images.
//
//////////////////////////////////////////////////////////////////////////////
func newsHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)

	numArticlesToDisplay := len(articles)//min(50, len(articles))
	prVal(nw_, "numArticlesToDisplay", numArticlesToDisplay)
	
	articleArgs := make([]ArticleArg, numArticlesToDisplay)
	
	/*
	posts := fetchPosts()

	if float32(len(newArticles)) >= .8 * float32(len(articles)) {
		pr(ns_, "Copying new articles")
		mutex.Lock()
		articles = append(posts, newArticles...)

		prf(ns_, "posts: %d newArticles: %d --> articles: %d\n", len(posts), len(newArticles)) 

		mutex.Unlock()
		pr(ns_, "New articles copied")
	} else {
		pr(ns_, "Too many articles failed to fetch, probably Internet connectivity issues.  Will try again in 5 minutes.")
	}
	*/
	
	// TODO: Grab news from votezilla.NewsPost mixed with votezilla.Post... something like that!
	/*
	perm := rand.Perm(len(articles))
	perm[0] = 0 // HACK!!
	
	prVal(nw_, "perm", perm)
	
	mutex.RLock()
	// TODO: change type ArticleArgs to just be []Article
	for i := 0; i < numArticlesToDisplay; i++ {
		article := articles[perm[i]] // shuffle the article order (to mix between sources)

		// Copy the article information.
		articleArgs[i].Article	= article
		articleArgs[i].Size		= 0 // normal size
	}
	mutex.RUnlock()
	*/
	
	// TODO: add this code somewhere
	/*
	// Truncate the title if it's too long.
	const kMaxTitleLength = 122	
	if len(news.Articles[i].Title) > kMaxTitleLength {
		news.Articles[i].Title = news.Articles[i].Title[0:kMaxTitleLength] + "..."
	}

	// Hide the hostname to save space if the title is long.
	if len(news.Articles[i].Title) > 90 {
		news.Articles[i].Host = ""
	}
	*/
	
	// Sort by category.
	// TODO: sort by category, then by rank.
	sort.Slice(articleArgs, func(i, j int) bool {
	  return articleArgs[i].Category < articleArgs[j].Category
	})

	numCategories := len(categoryOrder)
	
	articleGroups := make([]ArticleGroup, numCategories)
	
	const (
		kArticlesPerRow = 2
		kRowsPerCategory = 6
	)
	
	cat := 0
	headlineSide := 0 // The side that has the headline (large article).
	for ccc, category := range categoryOrder {
		row := 0
		col := 0
		filled := false
		
		articleGroups[cat].Category = category
		articleGroups[cat].BgColor = bgColors[category]
		articleGroups[cat].HeaderColor = headerColors[category]
		articleGroups[cat].HeadlineSide = headlineSide
		
		for _, articleArg := range articleArgs {
			// This should work since we've sorted by category.
			if articleArg.Category == category {
				if row == 0 {
					// Make room for new row
					articleGroups[cat].ArticleArgs = append(articleGroups[cat].ArticleArgs, 
														    make([]ArticleArg, kRowsPerCategory))
				}
				
				articleGroups[cat].ArticleArgs[col][row] = articleArg
				
				// First article on the "large side" is a headline (bigger), the remaining articles get skipped.
				if col == headlineSide && ccc == 0 {
					if row == 0 {
						articleGroups[cat].ArticleArgs[col][row].Size =  1 // 1 means large article
					} else {
						articleGroups[cat].ArticleArgs[col][row].Size = -1 // -1 means skip the article
					}
				}
				
				// Inc row, col
				col++
				if col == kArticlesPerRow {
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
				articleGroups[cat].ArticleArgs = append(articleGroups[cat].ArticleArgs, 
														make([]ArticleArg, kRowsPerCategory))
			}
			
			articleGroups[cat].ArticleArgs[col][row].Size = -1 // -1 means skip the article
			
			// Inc row, col
			col++
			if col == kArticlesPerRow {
				col = 0
				row++

				if row == kRowsPerCategory {
					filled = true
					break
				}
			}
		}
		
		cat++
		headlineSide = (headlineSide + 1) % 2 // The side with the headline switches each time, to look nice.
	}

	// Get the username.
	userId := GetSession(r)
	username := getUsername(userId)

	// Render the news articles.
	newsArgs := struct {
		PageArgs
		Username		string
		ArticleGroups	[]ArticleGroup
		LastColumnIdx	int
		NavMenu			[]string
		UrlPath			string
	}{
		PageArgs:		PageArgs{Title: "votezilla - News"},
		Username:		username,
		ArticleGroups:	articleGroups,
		NavMenu:		navMenu,
		UrlPath:		"news",
	}
	
	executeTemplate(w, "news", newsArgs)
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
	executeTemplate(w, "newsSources", newsSourcesArgs)	
}
*/