package main

import (
	"time"
)

//////////////////////////////////////////////////////////////////////////////
//
// fetch posts from database
//
//////////////////////////////////////////////////////////////////////////////
func fetchPosts() (articleArgs []Article) {
	var title		string
	var linkUrl		string
	var created		time.Time
	var username	string
	var country		string
	
	rows := DbQuery(`
		SELECT L.Title, L.LinkUrl, L.Created, U.Username, U.Country
		FROM votezilla.LinkPost L 
		JOIN votezilla.User U ON L.UserId = U.Id 
		LIMIT 50;`)
	defer rows.Close()

	for rows.Next() {
		check(rows.Scan(&title, &linkUrl, &created, &username, &country))

		prVal(po_, "title", title)
		prVal(po_, "linkUrl", linkUrl)
		prVal(po_, "created", created)
		prVal(po_, "username", username)
		prVal(po_, "country", country)

		// Set the article information
		articleArgs = append(articleArgs, Article{
			Author:			"",
			Title:			title,
			Description:	"",
			Url:			linkUrl,
			UrlToImage:		"",
			PublishedAt:	"",
			NewsSourceId:	"",
			Host:			"",
			Category:		"general",
			Language:		"EN",
			Country:		country,
		})
	}
	check(rows.Err())
	
	return articleArgs
}
