package main

import (
	"net/http"
	"math/rand"
)

//////////////////////////////////////////////////////////////////////////////
//
// display posts
// TODO: santize (html- and url-escape the arguments).  (Make sure URL's don't point back to votezilla.)
// TODO: use a caching, resizing image proxy for the images.
//
//////////////////////////////////////////////////////////////////////////////
func postsHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)

	numArticlesToDisplay := min(50, len(articles))
	
	articleArgs := make([]ArticleArg, numArticlesToDisplay)
	
	perm := rand.Perm(len(articles))
	
	mutex.RLock()
	for i := 0; i < numArticlesToDisplay; i++ {
		article := articles[perm[i]] // shuffle the article order (to mix between sources)

		// Copy the article information.
		articleArgs[i].Article = article

		// Set the index
		articleArgs[i].Index = i + 1
	}
	mutex.RUnlock()

	// Get the username.
	userId := GetSession(r)
	username := getUsername(userId)

	// Render the news articles.
	newsArgs := struct {
		PageArgs
		Username	string
		Articles	[]ArticleArg
		NavMenu		[]string
		UrlPath		string
	}{
		PageArgs:	PageArgs{Title: "votezilla - News"},
		Username:	username,
		Articles:	articleArgs,
		NavMenu:	navMenu,
		UrlPath:	"posts",
	}
	
	executeTemplate(w, "news", newsArgs)
}
