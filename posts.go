package main

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"
)


// JSON-parsed format of an article.
type Article struct {
	Author			string
	Title			string
	Description		string
	Url				string
	UrlToImage		string
	PublishedAt		string
	// Custom parameters:
	Id				string
	UrlToThumbnail	string
	NewsSourceId	string
	Host			string
	Category		string
	Language		string
	Country			string
	PublishedAtUnix	time.Time
	TimeSince		string
}

//////////////////////////////////////////////////////////////////////////////
//
// query news articles and user posts from database, with condition test on the
// id, category, and optional partitioning per category.
// If articlesPerCategory <= 0, no category partitioning takes place.
//
//////////////////////////////////////////////////////////////////////////////
func _queryArticles(idCondition string, categoryCondition string, articlesPerCategory int, 
					maxArticles int) (articles []Article) {
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
	var orderBy			time.Time
	var rowNumber		int
	
	// Union of NewsPosts (News API) and LinkPosts (user articles).
	query := fmt.Sprintf(`
		SELECT Id, NewsSourceId AS Author, Title, Description, LinkUrl, COALESCE(UrlToImage, ''),
			   COALESCE(PublishedAt, Created) AS PublishedAt, NewsSourceId, Category, Language, Country,
			   COALESCE(PublishedAt, Created) + RANDOM() * '3:00:00'::INTERVAL AS OrderBy
		FROM votezilla.NewsPost
		WHERE ThumbnailStatus = 1 AND (Id %s) AND (Category %s)
		UNION
		SELECT P.Id, U.Username AS Author, P.Title, '' AS Description, P.LinkUrl, COALESCE(P.UrlToImage, ''),
			   P.Created AS PublishedAt, '' AS NewsSourceId, 'news' AS Category, 'EN' AS Language, U.Country,
			   P.Created + RANDOM() * '1:00:00'::INTERVAL AS OrderBy
		FROM ONLY votezilla.LinkPost P 
		JOIN votezilla.User U ON P.UserId = U.Id
		WHERE (P.Id %s) AND ('news' %s)
		ORDER BY OrderBy DESC`, 
		idCondition, categoryCondition,
		idCondition, categoryCondition)
	if articlesPerCategory > 0 {
		// Select 5 articles of each category
		query = fmt.Sprintf(`
			SELECT * 
			FROM (		
				SELECT 
					*,
					ROW_NUMBER() OVER (PARTITION BY Category ORDER BY 
						OrderBy DESC) AS r 
				FROM (%s) x
			) x
			WHERE x.r <= %d`, 
			query, 
			articlesPerCategory)
	}
	if maxArticles > 0 {
		query += ` LIMIT ` + strconv.Itoa(maxArticles)
	}
	query += `;`
	
	prf(po_, "query: %s", query)
	rows := DbQuery(query)
	
	for rows.Next() {
		if articlesPerCategory > 0 {
			check(rows.Scan(&id, &author, &title, &description, &linkUrl, &urlToImage, 
							&publishedAt, &newsSourceId, &category, &language, &country, &orderBy, &rowNumber))
		} else {
			check(rows.Scan(&id, &author, &title, &description, &linkUrl, &urlToImage, 
							&publishedAt, &newsSourceId, &category, &language, &country, &orderBy))
		}
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
		
		// Format time since article was posted to a short format, e.g. "2h" for 2 hours.
		var timeSinceStr string
		{
			timeSince := time.Since(publishedAt)
			seconds := timeSince.Seconds()
			minutes := timeSince.Minutes()
			hours := timeSince.Hours()
			days := hours / 24.0
			weeks := days / 7.0
			
			if weeks >= 1.0 {
				timeSinceStr = strconv.FormatFloat(weeks, 'f', 0, 32) + "w"
			} else if days >= 1.0 {
				timeSinceStr = strconv.FormatFloat(days, 'f', 0, 32) + "d"
			} else if hours >= 1.0 {
				timeSinceStr = strconv.FormatFloat(hours, 'f', 0, 32) + "h"
			} else if minutes >= 1.0 {
				timeSinceStr = strconv.FormatFloat(minutes, 'f', 0, 32) + "m"
			} else {
				timeSinceStr = strconv.FormatFloat(seconds, 'f', 0, 32) + "s"
			}
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
			PublishedAtUnix:publishedAt,
			PublishedAt:	publishedAt.Format(time.UnixDate),
			NewsSourceId:	newsSourceId,
			Host:			host,
			Category:		category,
			Language:		language,
			Country:		country,
			TimeSince:		timeSinceStr,
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
	articles := _queryArticles(
		"= " + strconv.FormatInt(id, 10), 
		"IS NOT NULL",
		-1,
		2) // 2, so we could potentially catch duplicate articles.
	
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
// fetch news articles partitioned by category, up to articlesPerCategory
// articles per category, up to maxArticles total.
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticlesPartitionedByCategory(articlesPerCategory int, maxArticles int) ([]Article) {
	return _queryArticles(
		"IS NOT NULL",
		"IS NOT NULL",
		articlesPerCategory,
		maxArticles)
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch news articles within a particular category, up to maxArticles total.
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticlesWithinCategory(category string, maxArticles int) ([]Article, error) {
	// Validate we have a valid category; otherwise, this could be SQL injection!
	if _, ok := headerColors[category]; !ok {
	    return []Article{}, errors.New("Invalid category")
	}

	return _queryArticles(
		"IS NOT NULL", 
		"= '" + category + "'",
		-1,
		maxArticles), nil
}