package main

import (
	"net/url"
	"time"
)


//////////////////////////////////////////////////////////////////////////////
//
// fetch news articles from database
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticles() (articleArgs []Article) {
	var title			string
	var description		string
	var linkUrl			string
	var urlToImage		string
	var publishedAt		time.Time
	var newsSourceId	string
	var category		string
	var language		string
	var country			string
	
	rows := DbQuery(
		`SELECT Title, Description, LinkUrl, COALESCE(UrlToImage, ''), COALESCE(PublishedAt, Created), NewsSourceId, 
				Category, Language, Country
		 FROM votezilla.NewsPost
		 LIMIT 600;`)
	
	for rows.Next() {
		check(rows.Scan(&title, &description, &linkUrl, &urlToImage, &publishedAt, &newsSourceId, 
					    &category, &language, &country))

		//prVal(po_, "title", title)		
		//prVal(po_, "description", description)	
		//prVal(po_, "linkUrl", linkUrl)		 
		//prVal(po_, "urlToImage", urlToImage)	 
		//prVal(po_, "publishedAt", publishedAt)	 
		//prVal(po_, "newsSourceId", newsSourceId)
		//prVal(po_, "category", category)	
		//prVal(po_, "language", language)	 
		//prVal(po_, "country", country)
		
		// Parse the hostname.  TODO: parse away the "www."
		host := ""
		u, err := url.Parse(linkUrl)
		if err != nil {
			host = "Error parsing hostname"
		} else {
			host = u.Host
		}

		// Set the article information
		articleArgs = append(articleArgs, Article{
			Author:			newsSourceId, // haha hijacking Author to be the poster
			Title:			title,
			Description:	description,
			Url:			linkUrl,
			UrlToImage:		urlToImage,
			PublishedAt:	publishedAt.Format(time.UnixDate),
			NewsSourceId:	newsSourceId,
			Host:			host,
			Category:		category,
			Language:		language,
			Country:		country,
		})
	}
	check(rows.Err())
	rows.Close()
	
	return articleArgs
}


//////////////////////////////////////////////////////////////////////////////
//
// fetch posts from database
//
//////////////////////////////////////////////////////////////////////////////
func fetchPosts() (articleArgs []Article) {
	var title		string
	var linkUrl		string
	var urlToImage	string
	var created		time.Time
	var username	string
	var country		string
	
	rows := DbQuery(
		`SELECT L.Title, L.LinkUrl, COALESCE(L.UrlToImage, ''), L.Created, U.Username, U.Country
		 FROM ONLY votezilla.LinkPost L 
		 JOIN votezilla.User U ON L.UserId = U.Id 
		 LIMIT 50;`)

	for rows.Next() {
		check(rows.Scan(&title, &linkUrl, &urlToImage, &created, &username, &country))
		
		//prVal(po_, "title", title)
		//prVal(po_, "linkUrl", linkUrl)
		//prVal(po_, "created", created)
		//prVal(po_, "username", username) // TODO: add username to some Article arg... so it can be displayed
		//prVal(po_, "country", country)
		
		// Parse the hostname.  TODO: parse away the "www."
		host := ""
		u, err := url.Parse(linkUrl)
		if err != nil {
			host = "Error parsing hostname"
		} else {
			host = u.Host
		}
		
		// Set the article information
		articleArgs = append(articleArgs, Article{
			Author:			username, // haha hijacking Author to be the poster
			Title:			title,
			Description:	"",
			Url:			linkUrl,
			UrlToImage:		urlToImage,
			PublishedAt:	created.Format(time.UnixDate), // <-- TODO: not exactly the same thing, but close enough?
			NewsSourceId:	"",
			Host:			host,
			Category:		"general",
			Language:		"EN",
			Country:		country,
		})
	}
	check(rows.Err())
	rows.Close()
	
	return articleArgs
}
