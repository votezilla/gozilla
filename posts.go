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
	Id				int64
	UrlToThumbnail	string
	NewsSourceId	string
	Host			string
	Category		string
	Language		string
	Country			string
	PublishedAtUnix	time.Time
	TimeSince		string
	Size			int		// 0=normal, 1=large (headline)
	AuthorIconUrl	string
	Bucket			string  // "" by default, but can override Category as a way to categorize articles
	Upvoted			int
	VoteTally		int
	
	IsPoll			bool
	PollOptions		[]string
}

//////////////////////////////////////////////////////////////////////////////
//
// query news articles and user posts from database, with condition test on the
// id, category, and optional partitioning per category.
// If articlesPerCategory <= 0, no category partitioning takes place.
//
//////////////////////////////////////////////////////////////////////////////
func _queryArticles(idCondition string, categoryCondition string, articlesPerCategory int, maxArticles int,
				    fetchVotesForUserId int64) (articles []Article) {
	var id				int64
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
	var upvoted			int
	var voteTally		int

	bRandomizeTime := (fetchVotesForUserId == -1)

	// Union of NewsPosts (News API) and LinkPosts (user articles).
	query := fmt.Sprintf(
	   `SELECT Id, NewsSourceId AS Author, Title, Description, LinkUrl,
	   		   COALESCE(UrlToImage, '') AS UrlToImage, COALESCE(PublishedAt, Created) AS PublishedAt,
	   		   NewsSourceId,
	   		   $$GetCategory(Category, Country) AS Category,
	   		   Language, Country,
			   COALESCE(PublishedAt, Created) %s AS OrderBy
		FROM $$NewsPost
		WHERE ThumbnailStatus = 1 AND (Id %s) AND ($$GetCategory(Category, Country) %s)
		UNION
		SELECT P.Id, U.Username AS Author, P.Title, '' AS Description, P.LinkUrl,
			   COALESCE(P.UrlToImage, '') AS UrlToImage, P.Created AS PublishedAt,
			   '' AS NewsSourceId,
			   $$GetCategory('news', U.Country) AS Category,
			   'EN' AS Language, U.Country,
			   P.Created %s AS OrderBy
		FROM ONLY $$LinkPost P
		JOIN $$User U ON P.UserId = U.Id
		WHERE (P.Id %s) AND ($$GetCategory('news', U.Country) %s)
		ORDER BY OrderBy DESC`,
		ternary_str(bRandomizeTime, "+ RANDOM() * '3:00:00'::INTERVAL", ""),
		idCondition,
		categoryCondition,
		ternary_str(bRandomizeTime, "+ RANDOM() * '1:00:00'::INTERVAL", ""),
		idCondition,
		categoryCondition,
	)
	if articlesPerCategory > 0 {
		// Select 5 articles of each category
		query = fmt.Sprintf(`
			SELECT Id, Author, Title, Description, LinkUrl, UrlToImage,
				   PublishedAt, NewsSourceId, Category, Language, Country, OrderBy
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
	} else if fetchVotesForUserId >= 0 {
		// Join query to post votes table.
		query = fmt.Sprintf(`
			SELECT x.*,
				   CASE WHEN v.Up IS NULL THEN 0
				        WHEN v.Up THEN 1
				        ELSE -1
				   END AS Upvoted
			FROM (%s) x
			LEFT JOIN $$PostVote v ON x.Id = v.PostId AND (v.UserId = %d)
			ORDER BY v.Created`,
			query,
			fetchVotesForUserId)
	}
	if maxArticles > 0 {
		query += ` LIMIT ` + strconv.Itoa(maxArticles)
	}
	// Add vote tally per article.
	query = fmt.Sprintf(`
		WITH posts AS (%s),
	  		 votes AS (
				SELECT PostId,
					   SUM(CASE WHEN Up THEN 1 ELSE -1 END) AS VoteTally
				FROM $$PostVote
				WHERE PostId IN (SELECT Id FROM posts)
				GROUP BY PostId
			 )
		SELECT posts.*, COALESCE(votes.VoteTally, 0) AS VoteTally
		FROM posts
		LEFT JOIN votes ON posts.Id = votes.PostId`,
		query)
	query += `;`

	rows := DbQuery(query)

	for rows.Next() {
		if fetchVotesForUserId >= 0 {
			check(rows.Scan(&id, &author, &title, &description, &linkUrl, &urlToImage,
							&publishedAt, &newsSourceId, &category, &language, &country, &orderBy, &upvoted, &voteTally))
		} else {
			check(rows.Scan(&id, &author, &title, &description, &linkUrl, &urlToImage,
							&publishedAt, &newsSourceId, &category, &language, &country, &orderBy, &voteTally))
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

		// Author icon URL: either the news source's, or the user's.  (TODO: let users pick their dino head / upload a photo.)
		authorIconUrl := ""
		if newsSourceId != "" {
			authorIconUrl = "/static/newsSourceIcons/" + newsSourceId + ".png"
		} else {
			authorIconUrl = "/static/mozilla dinosaur head.png" // TODO: we need real dinosaur icons for users.
		}

		// Set the article information
		newArticle := Article{
			Id:				id,
			Author:			author, // haha hijacking Author to be the poster
			Title:			title,
			Description:	description,
			Url:			linkUrl,
			UrlToImage:		coalesce_str(urlToImage, "/static/mozilla dinosaur head.png"),
			UrlToThumbnail:	"/static/thumbnails/" + strconv.FormatInt(id, 10) + ".jpeg",
			PublishedAtUnix:publishedAt,
			PublishedAt:	publishedAt.Format(time.UnixDate),
			NewsSourceId:	newsSourceId,
			Host:			host,
			Category:		category,
			Language:		language,
			Country:		country,
			TimeSince:		timeSinceStr,
			AuthorIconUrl:	authorIconUrl,
			Upvoted:		upvoted,
			VoteTally:		voteTally,
		}	
		
		// Hack in polls for now
		if id % 6 == 0 {
			newArticle.IsPoll = true
			newArticle.Title = "Poll: Who should be president in 2020?"
			newArticle.PollOptions = []string{"Trump", "Clinton", "Sanders"}
		} else if id % 6 == 3 {
			newArticle.IsPoll = true
			newArticle.Title = "Poll: Was Jeffrey Epstein murdered?"
			newArticle.PollOptions = []string{"Yes", "No", "Maybe", "Not sure"}
		}

		if newArticle.IsPoll {
			newArticle.UrlToImage     = "/static/ballotbox.png"
			newArticle.UrlToThumbnail = "/static/ballotbox small.png"
			newArticle.Url = fmt.Sprintf("/comments/?postId=%d", id) // "/comments" is synonymous with clicking on a post (or poll) to see more info.
		}
		
		articles = append(articles, newArticle)
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
func fetchArticle(id int64, userId int64) (Article, error) {
	articles := _queryArticles(
		"= " + strconv.FormatInt(id, 10),
		"IS NOT NULL",
		-1,
		2,	// 2, so we could potentially catch duplicate articles.
		userId)

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
// articles per category, up to maxArticles total, which excludeUserId did not vote on.
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticlesPartitionedByCategory(articlesPerCategory int, excludeUserId int64, maxArticles int) ([]Article) {
	return _queryArticles(
		"NOT IN (SELECT PostId FROM $$PostVote WHERE UserId = " + strconv.FormatInt(excludeUserId, 10) + ")",
		"IS NOT NULL",
		articlesPerCategory,
		maxArticles,
		-1)
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch news articles within a particular category, up to maxArticles total,
// which userId voted on.
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticlesVotedOnByUser(userId int64, maxArticles int) ([]Article) {
	return _queryArticles(
		"IN (SELECT PostId FROM $$PostVote WHERE UserId = " + strconv.FormatInt(userId, 10) + ")",
		"IS NOT NULL",
		-1,
		maxArticles,
		userId)
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch news articles within a particular category, up to maxArticles total,
// which excludeUserId did not vote on.
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticlesWithinCategory(category string, excludeUserId int64, maxArticles int) ([]Article) {
	return _queryArticles(
		"NOT IN (SELECT PostId FROM $$PostVote WHERE UserId = " + strconv.FormatInt(excludeUserId, 10) + ")",
		"= '" + category + "'",
		-1,
		maxArticles,
		-1)
}
