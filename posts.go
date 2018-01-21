package main

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"
)


//////////////////////////////////////////////////////////////////////////////
//
// query news articles and user posts from database, with condition test on the id.
//
//////////////////////////////////////////////////////////////////////////////
func _queryArticles(idCondition string) (articles []Article) {
	var id				string
	var author			string
	var title			string
	var description		string
	var linkUrl			string
	var urlToImage		string
	var publishedAt		time.Time
	var newsSourceId	string
	var category		string
	var language		string
	var country			string
	
	query := fmt.Sprintf(`SELECT Id, NewsSourceId AS Author, Title, Description, LinkUrl, COALESCE(UrlToImage, ''),
				COALESCE(PublishedAt, Created) AS PublishedAt, NewsSourceId, Category, Language, Country
		 FROM votezilla.NewsPost
		 WHERE ThumbnailStatus = 1 AND (Id %s)
		 UNION
		 SELECT P.Id, U.Username AS Author, P.Title, '' AS Description, P.LinkUrl, COALESCE(P.UrlToImage, ''),
		 		P.Created AS PublishedAt, '' AS NewsSourceId, 'EN' AS Category, 'EN' AS Language, U.Country
		 FROM ONLY votezilla.LinkPost P 
		 JOIN votezilla.User U ON P.UserId = U.Id
		 WHERE (P.Id %s)
		 ORDER BY PublishedAt DESC
		 LIMIT 600;`, idCondition, idCondition)
		 
	//prVal(po_, "query", query)
	
	// Union of NewsPosts (News API) and LinkPosts (user articles)
	rows := DbQuery(query)
	
	for rows.Next() {
		check(rows.Scan(&id, &author, &title, &description, &linkUrl, &urlToImage, 
						&publishedAt, &newsSourceId, &category, &language, &country))

		//prVal(po_, "id", id)
		//prVal(po_, "author", author)
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
		
		// Combine "politics" and "general" into "news>
		if category == "politics" || category == "general" {
			category = "news"
		}

		// Set the article information
		articles = append(articles, Article{
			Id:				id,
			Author:			author, // haha hijacking Author to be the poster
			Title:			title,
			Description:	description,
			Url:			linkUrl,
			UrlToImage:		coalesce_str(urlToImage, "/static/mozilla dinosaur head.png"),
			UrlToThumbnail:	"/static/thumbnails/" + id + ".jpeg",
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
	
	return articles
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch a single article from database
// TODO: doesn't work with user-submitted posts.  Will need a database refactor for that!
// id - article id
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticle(id int64) (Article, error) {
	articles := _queryArticles(" = " + strconv.FormatInt(id, 10))
	
	len := len(articles)
	
	if len == 1 {
		return articles[0], nil
	} else if len == 0 {
		return Article{}, errors.New("Article not found")
	} else {
		return Article{}, errors.New("Duplicate articles found")
	}
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch news articles from database
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticles() (articles []Article) {
	return _queryArticles("IS NOT NULL")
}