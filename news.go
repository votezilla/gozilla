package main

import (
	"net/http"
	"math/rand"
	"sort"
	//"strconv"
	//"net/url"
)

// For rendering the news article information.
type ArticleArg struct {
	Article
	Size			int		// 0=normal, 1=large (headline)
	AuthorIconUrl	string
}

type ArticleGroup struct {
	ArticleArgs		[][]ArticleArg // Arrow of rows, each row has 2 articles.
	Category		string
	HeaderColor		string
	BgColor			string
	HeadlineSide	int		// 0=left, 1=right (On large, i.e. non-mobile, devices...)
	More			string	// category if "More..." appears at end of group.
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
	articles []Article
	
	// Custom-written data from https://newsapi.org/v1/sources?language=en query
	newsSources NewsSources
	
	categoryOrder = []string{
		"news", 			
		"business", 			
		"sport", 			
		"entertainment", 	
		"science-and-nature",
		"technology",		
		"gaming",			
		"music", 			
	}
	
	headerColors map[string]string = map[string]string{
		"news" 			 	: "#ccc",
		"business" 			: "#8e8",
		"sport" 			: "#88f",
		"entertainment" 	: "#e85be4",
		"science-and-nature": "#8cf",
		"technology" 		: "#8ff",
		"gaming" 			: "#58d858",
		"music" 			: "#fd8",
	}

	bgColors map[string]string = map[string]string{
		"news"	 			: "#ddd",
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
	  
	prVal(nw_, "r.URL.Query()", r.URL.Query())

	reqCategory		:= parseUrlParam(r, "category")
	reqNoHeadlines	:= parseUrlParam(r, "noHeadlines")

	// TODO: cache this, fetch every minute?
	articles := fetchArticles()
	
	prf(ns_, "Fetched %d articles", len(articles))
	
	numArticlesToDisplay := len(articles)
	prVal(nw_, "numArticlesToDisplay", numArticlesToDisplay)
	
	prf(ns_, "There are now %d articles total", len(articles))
	
	articleArgs := make([]ArticleArg, numArticlesToDisplay)
	
	perm := rand.Perm(len(articles))
	perm[0] = 0 // HACK!!
	
	//prVal(nw_, "perm", perm)
	
	// TODO: change type ArticleArgs to just be []Article
	for i := 0; i < numArticlesToDisplay; i++ {
		article := articles[perm[i]] // shuffle the article order (to mix between sources)

		// Truncate the title if it's too long.
		const kMaxTitleLength = 122	
		if len(article.Title) > kMaxTitleLength {
			article.Title = article.Title[0:kMaxTitleLength] + "..."
		}

		// Hide the hostname to save space if the title is long.
		if len(article.Title) > 90 {
			article.Host = ""
		}

		// Copy the article information.
		articleArgs[i].Article	= article
		articleArgs[i].Size		= 0 // normal size
		
		if article.NewsSourceId != "" {
			articleArgs[i].AuthorIconUrl = "/static/newsSourceIcons/" + article.NewsSourceId + ".png"
		} else {
			articleArgs[i].AuthorIconUrl = "/static/mozilla dinosaur head.png" // TODO: we need real dinosaur icons for users.
		}
	}
	
	// Sort by category.
	// TODO: sort by category, then by rank.
	sort.Slice(articleArgs, func(i, j int) bool {
	  return articleArgs[i].Category < articleArgs[j].Category
	})

	numCategories := len(categoryOrder)
	
	articleGroups := make([]ArticleGroup, numCategories)
	
	var (
		kArticlesPerRow = 2
		kRowsPerCategory = 4//5
	)
	
	cat := 0
	headlineSide := 0 // The side that has the headline (large article).
	currArticle := 0
	for ccc, category := range categoryOrder {
		row := 0
		col := 0
		filled := false
	
		if reqCategory == "" { // Mixed categories
			articleGroups[cat].Category = category
			articleGroups[cat].More = category
		} else { 			   // Single category
			category = reqCategory // Make all categories the same
			// Only the first articleGroup has a category name, the rest have "",
			// which is a flag to have no category header.
			if ccc == 0 {
				articleGroups[cat].Category = category
			} else {
				articleGroups[cat].Category = ""
			}
			articleGroups[cat].More = ""
		} 
		articleGroups[cat].BgColor = bgColors[category]
		articleGroups[cat].HeaderColor = headerColors[category]
		articleGroups[cat].HeadlineSide = headlineSide
		
		if reqCategory == "" {
			currArticle = 0
		}
		
		// TODO: if a single category, without noHeadlines, either large image should be set to always 
		// 4 article height, or all articles should stack verticlally in each column.  
		// (I prefer the second idea, because it might look nicer.)
		
		for ; currArticle < len(articleArgs); currArticle++ {
			articleArg := articleArgs[currArticle]
		
			// This works since we've sorted by category.
			if articleArg.Category == category {
				if row == 0 {
					// Make room for new row
					articleGroups[cat].ArticleArgs = append(articleGroups[cat].ArticleArgs, 
														    make([]ArticleArg, kRowsPerCategory))
				}
				
				// The first article is always the headline.  Articles after the headline get skipped.
				size := 0
				
				if reqNoHeadlines == "" {
					if col == 0 {
						if row == 0 { // first article is the headline, i.e. big
							size =  1 // 1 means large article (headline)
						} else {      // the rest of the articles get skipped, since the headline takes all the space.
							size = -1 // -1 means skip the article
						}
					}
				}
			
				// Assign this slot the next article, as long as this is not an empty slot.  Make sure size gets assigned!
				if size == -1 {
					articleGroups[cat].ArticleArgs[col][row].Size = -1
				} else {
					articleGroups[cat].ArticleArgs[col][row] = articleArg
					articleGroups[cat].ArticleArgs[col][row].Size = size
					
					//articleArg.Article.Title = articleArg.Article.Title[0:9] + " " + strconv.Itoa(row) + " " + strconv.Itoa(col) + " " + strconv.Itoa(headlineSide)
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
		
		if reqNoHeadlines == "" {
			headlineSide = (headlineSide + 1) % 2 // The side with the headline switches each time, to look nice.
		}
	}
	
	// If a single category, only the last articleGroup should have a "More..." link.
	if reqCategory != "" { 
		articleGroups[cat - 1].More = reqCategory
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
	
	//prVal(nw_, "newsArgs", newsArgs)
	
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