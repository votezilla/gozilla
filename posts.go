package main

import (
	"net/http"
	"time"
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

	pr(po_, "postsHandler")
	
	var title		string
	var linkUrl		string
	var created		time.Time
	var username	string
	rows := DbQuery(`
		SELECT L.Title, L.LinkUrl, L.Created, U.Username 
		FROM votezilla.LinkPost L 
		JOIN votezilla.User U ON L.UserId= U.Id 
		LIMIT 50;`)
	defer rows.Close()
	
	var articleArgs []ArticleArg //make([]ArticleArg, len(rows))
	
	for rows.Next() {
		check(rows.Scan(&title, &linkUrl, &created, &username))
		
		prVal(po_, "title", title)
		prVal(po_, "linkUrl", linkUrl)
		prVal(po_, "created", created)
		prVal(po_, "username", username)
		
		// Set the article information
		articleArgs = append(articleArgs, ArticleArg{
			Article:	Article{
				Author:			"",
				Title:			title,
				Description:	"",
				Url:			linkUrl,
				UrlToImage:		"",
				PublishedAt:	"",
				NewsSourceId:	"",
				Host:			username, // TODO: this is not the host
				Category:		"",
				Language:		"EN",
				Country:		"US",
			},
			Index: 		len(articleArgs) + 1,
		})
	}
	check(rows.Err())

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
