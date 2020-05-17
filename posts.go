package main

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	//"database/sql/driver"
	_ "github.com/lib/pq"
	//"github.com/jmoiron/sqlx"
	"encoding/json"
)


//////////////////////////////////////////////////////////////////////////////
//
// query news articles and user posts from database, with condition test on the
// id, category, and optional partitioning per category.
// If articlesPerCategory <= 0, no category partitioning takes place.
//
//////////////////////////////////////////////////////////////////////////////
func _queryArticles(idCondition string, userIdCondition string, categoryCondition string, newsSourceIdCondition string,
					articlesPerCategory int, maxArticles int, fetchVotesForUserId int64) (articles []Article) {
	pr("_queryArticles")
	prVal("idCondition", idCondition)
	prVal("userIdCondition", userIdCondition)
	prVal("categoryCondition", categoryCondition)
	prVal("newsSourceIdCondition", newsSourceIdCondition)
	prVal("articlesPerCategory", articlesPerCategory)
	prVal("maxArticles", maxArticles)
	prVal("fetchVotesForUserId", fetchVotesForUserId)

	bRandomizeTime := false  // REVERT!!!
	//bRandomizeTime := (fetchVotesForUserId == -1)

	// Union of NewsPosts (News API) and LinkPosts (user articles).
	newsPostQuery := fmt.Sprintf(
	   `SELECT Id, NewsSourceId AS Author, Title, Description, LinkUrl,
	   		   COALESCE(UrlToImage, '') AS UrlToImage, COALESCE(PublishedAt, Created) AS PublishedAt,
	   		   NewsSourceId,
	   		   $$GetCategory(Category, Country) AS Category,
	   		   Language, Country,
			   '' AS PollOptionData,
			   COALESCE(PublishedAt, Created) %s AS OrderBy,
			   NumComments,
			   ThumbnailStatus,
			   'N' AS Source
		FROM $$NewsPost
		WHERE ThumbnailStatus <> -1 AND (Id %s) AND (NewsSourceId %s) AND ($$GetCategory(Category, Country) %s)`,
		ternary_str(bRandomizeTime, "- RANDOM() * '3:00:00'::INTERVAL", ""), // Make it randomly up to 3 hours later.
		idCondition,
		newsSourceIdCondition,
		categoryCondition)

	linkPostQuery := fmt.Sprintf(
	   `SELECT P.Id, U.Username AS Author, P.Title, '' AS Description, P.LinkUrl,
			   COALESCE(P.UrlToImage, '') AS UrlToImage, P.Created AS PublishedAt,
			   '' AS NewsSourceId,
			   $$GetCategory(Category, U.Country) AS Category,
			   'EN' AS Language, U.Country,
			   '' AS PollOptionData,
			   P.Created %s AS OrderBy,
			   NumComments,
			   ThumbnailStatus,
			   'L' AS Source
		FROM $$LinkPost P
		JOIN $$User U ON P.UserId = U.Id
		WHERE ThumbnailStatus <> -1 AND (P.Id %s) AND (U.Id %s) AND ($$GetCategory(Category, U.Country) %s)`,
		"", //ternary_str(bRandomizeTime, "- RANDOM() * '1:00:00'::INTERVAL", ""),
		idCondition,
		userIdCondition,
		categoryCondition)

	pollPostQuery := fmt.Sprintf(
	   `SELECT P.Id, U.Username AS Author, P.Title, '' AS Description, FORMAT('/poll/?postId=%%s', P.Id),
			   COALESCE(P.UrlToImage, '') AS UrlToImage, P.Created AS PublishedAt,
			   '' AS NewsSourceId,
			   $$GetCategory(Category, U.Country) AS Category,
			   'EN' AS Language, U.Country,
			   PollOptionData,
			   P.Created %s AS OrderBy,
			   NumComments,
			   ThumbnailStatus,
			   'P' AS Source
		FROM $$PollPost P
		JOIN $$User U ON P.UserId = U.Id
		WHERE (P.Id %s) AND (U.Id %s) AND ($$GetCategory(Category, U.Country) %s)`,	// Removed: 'ThumbnailStatus = 1 AND' because all polls currently use same thumbnail status
		"", //ternary_str(bRandomizeTime, "+ RANDOM() * '1:00:00'::INTERVAL", ""),
		idCondition,
		userIdCondition,
		categoryCondition)

	orderByClause := "\nORDER BY OrderBy DESC\n" // TODO: Use a Reddit-style ranking algorithm

	query := ""
	if userIdCondition != "IS NOT NULL" {  // Looking up posts that target a user - so there can be no news posts.
		query = strings.Join([]string{linkPostQuery, pollPostQuery}, "\nUNION ALL\n") + orderByClause
	} else if newsSourceIdCondition != "IS NOT NULL" {  // We're just querying news posts.
		query = newsPostQuery
	} else {
		query = strings.Join([]string{newsPostQuery, linkPostQuery, pollPostQuery}, "\nUNION ALL\n") + orderByClause
	}

	if articlesPerCategory > 0 {
		// Select 5 articles of each category
		query = fmt.Sprintf(`
			SELECT Id,
				Author,
				Title,
				Description,
				LinkUrl,
				UrlToImage,
				PublishedAt,
				NewsSourceId,
				Category,
				Language,
				Country,
				PollOptionData,
				OrderBy,
				NumComments,
				ThumbnailStatus,
				Source
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
	} else {
		query = fmt.Sprintf(`
			SELECT
				*
			FROM (%s) x
			ORDER BY x.OrderBy DESC`,
			query)
	}

	if fetchVotesForUserId >= 0 {
		// Join query to post votes table.
		query = fmt.Sprintf(`
			SELECT x.*,
				   CASE WHEN v.Up IS NULL THEN 0
						WHEN v.Up THEN 1
						ELSE -1
				   END AS Upvoted
			FROM (%s) x
			LEFT JOIN $$PostVote v ON x.Id = v.PostId AND (v.UserId = %d)
			ORDER BY v.Created DESC`,
			query,
			fetchVotesForUserId)
	}

	if maxArticles > 0 {
		query += "\nLIMIT " + strconv.Itoa(maxArticles)
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
		LEFT JOIN votes ON posts.Id = votes.PostId
		ORDER BY posts.OrderBy DESC`,
		query)
	query += `;`

	rows := DbQuery(query)
	for rows.Next() {
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
		var pollOptionJson	string
		var orderBy			time.Time
		var upvoted			int
		var voteTally		int
		var numComments		int
		var thumbnailStatus	int
		var source			string

		if fetchVotesForUserId >= 0 {
			check(rows.Scan(&id, &author, &title, &description, &linkUrl, &urlToImage,
							&publishedAt, &newsSourceId, &category, &language, &country, &pollOptionJson, &orderBy, &numComments, &thumbnailStatus, &source,
							&upvoted, &voteTally))
		} else {
			check(rows.Scan(&id, &author, &title, &description, &linkUrl, &urlToImage,
							&publishedAt, &newsSourceId, &category, &language, &country, &pollOptionJson, &orderBy, &numComments, &thumbnailStatus, &source,
							&voteTally))
		}
		//prVal("id", id)
		//prVal("author", author)
		//prVal("title", title)
		//prVal("description", description)
		//prVal("linkUrl", linkUrl)
		//prVal("urlToImage", urlToImage)
		//prVal("publishedAt", publishedAt)
		//prVal("newsSourceId", newsSourceId)
		//prVal("category", category)
		//prVal("language", language)
		//prVal("country", country)
		//prVal("pollOptionJson", pollOptionJson)
		//prVal("orderBy", orderBy)
		//prVal("upvoted", upvoted)
		//prVal("voteTally", voteTally)
		//prVal("numComments", numComments)
		//prVal("thumbnailStatus", thumbnailStatus)
		//prVal("source", source)

		// Parse the hostname.  TODO: parse away the "www."
		host := ""
		if source == "L" {  // Only show the host url when it's a user-submitted post, for security.
			u, err := url.Parse(linkUrl)
			if err != nil {
				host = "Error parsing hostname"
			} else {
				host = u.Host
			}

			if len(host) > 4 {
				if host[0:4] == "www." {
					host = host[4:]
				}
			}

			if host == "" {
				continue;  // Bad link - exclude it.  TODO: add this check when creating posts.
			}
		}
		//prVal("host", host)

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

		if urlToImage == "" {
			urlToImage = "/static/mozilla dinosaur head.png"
		}

		// Set the article information.
		newArticle := Article{
			Id:				id,
			Author:			author, // haha hijacking Author to be the poster
			Title:			title,
			Description:	description,
			Url:			linkUrl,
			UrlToImage:		urlToImage,
			UrlToThumbnail:
				ternary_str(thumbnailStatus == image_Unprocessed,					  // When thumnail is unprecessed, use...
					urlToImage,                                                       // Full-size image.
					ternary_str(urlToImage != "",
						"/static/thumbnails/" + strconv.FormatInt(id, 10) + ".jpeg",  // Thumbnail image processed.
						"/static/mozilla dinosaur thumbnail.png")),					  // Dropback if no image.  (TODO: replace licensed art.)
			PublishedAtUnix:publishedAt,
			PublishedAt:	publishedAt.Format(time.UnixDate),
			NewsSourceId:	newsSourceId,
			Host:			host,
			Category:		category,
			Language:		language,
			Country:		country,
			TimeSince:		timeSinceStr,
			AuthorIconUrl:
				ternary_str(newsSourceId != "",
					"/static/newsSourceIcons/" + newsSourceId + ".png",  // News source icon.
					"/static/mozilla dinosaur head.png"),                // TODO: we need real dinosaur icon art for users.
			Upvoted:		upvoted,
			VoteTally:		voteTally,
			NumComments:	numComments,
		}

		// Handle polls.
		if len(pollOptionJson) > 0 {
			newArticle.IsPoll 		  = true
			newArticle.Title 		  = "POLL: " + newArticle.Title
			newArticle.UrlToImage 	  = "/static/ballotbox.png"
			newArticle.UrlToThumbnail = "/static/ballotbox small.png"

			err = json.Unmarshal([]byte(pollOptionJson), &newArticle.PollOptionData)
			check(err)

			prVal("newArticle.PollOptionData", newArticle.PollOptionData)

			newArticle.Url = fmt.Sprintf("/article/?postId=%d", id) // "/comments" is synonymous with clicking on a post (or poll) to see more info.
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
		"= " + strconv.FormatInt(id, 10), // idCondition
		"IS NOT NULL",					  // userIdCondition
		"IS NOT NULL",					  // categoryCondition
		"IS NOT NULL",		  			  // newsSourceIdCondition	string
		-1,							  	  // articlesPerCategory 	int
		2,								  // 2, so we could potentially catch duplicate articles.
		userId)					 		  // fetchVotesForUserId 	int64

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
		"IS NOT NULL",        // idCondition
		"IS NOT NULL",        // userIdCondition
		"IS NOT NULL",        // categoryCondition
		"IS NOT NULL",		  // newsSourceIdCondition	string
		articlesPerCategory,  // articlesPerCategory 	int
		maxArticles,		  // maxArticles 			int
		-1)					  // fetchVotesForUserId 	int64
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch articles comented on by a user.
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticlesCommentedOnByUser(userId int64, maxArticles int) ([]Article) {
	return _queryArticles(
		"IN (SELECT PostId FROM $$Comment WHERE UserId = " + strconv.FormatInt(userId, 10) + ")", // idCondition
		"IS NOT NULL",																			  // userIdCondition
		"IS NOT NULL",																			  // categoryCondition
		"IS NOT NULL",						                                                      // newsSourceIdCondition	string
		-1,																						  // articlesPerCategory 	int
		maxArticles,																			  // maxArticles 			int
		-1)																						  // fetchVotesForUserId 	int64
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch articles within a particular category, up to maxArticles total,
// which userId voted on.
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticlesUpDownVotedOnByUser(userId int64, maxArticles int) ([]Article) {
	return _queryArticles(
		"IN (SELECT PostId FROM $$PostVote WHERE UserId = " + strconv.FormatInt(userId, 10) + ")",  // idCondition
		"IS NOT NULL",                                                                              // userIdCondition
		"IS NOT NULL",                                                                              // categoryCondition
		"IS NOT NULL",						                                                        // newsSourceIdCondition	string
		-1,																							// articlesPerCategory 	int
		maxArticles,																				// maxArticles 			int
		userId)																						// fetchVotesForUserId 	int64
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch articles within a particular category, up to maxArticles total,
// which excludeUserId did not vote on.
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticlesWithinCategory(category string, excludeUserId int64, maxArticles int) ([]Article) {
	_, foundCategory := newsCategoryInfo.HeaderColors[category]

	if foundCategory {
		return _queryArticles(
			"IS NOT NULL",                         // idCondition
			"IS NOT NULL",	                       // userIdCondition
			"= '" + category + "'",                // categoryCondition
			"IS NOT NULL",						   // newsSourceIdCondition	string
			-1,									   // articlesPerCategory 	int
			maxArticles,						   // maxArticles 			int
			-1)									   // fetchVotesForUserId 	int64
	} else {
		prVal("Unknown category", category)
		return []Article{}
	}
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch articles posted by a user.
//   category - optional, can provide "" to skip.
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticlesPostedByUser(userId int64, maxArticles int) ([]Article) {
	return _queryArticles(
		"IS NOT NULL", 						   // idCondition
		"= " + strconv.FormatInt(userId, 10),  // userIdCondition
		"IS NOT NULL",                         // categoryCondition
		"IS NOT NULL",						   // newsSourceIdCondition	string
		-1,									   // articlesPerCategory 	int
		maxArticles,						   // maxArticles 			int
		-1)									   // fetchVotesForUserId 	int64
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch articles from a news source.
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticlesFromThisNewsSource(newsSourceId string) (articles []Article) {
	// TODO_SECURITY: add additional check for newsSourceId within known news sources.

	return _queryArticles(
		"IS NOT NULL", 				                 // idCondition 			string
		"IS NOT NULL",   			                 // userIdCondition 		string
		"IS NOT NULL",   			                 // categoryCondition 	    string
		"= '" + sqlEscapeString(newsSourceId) + "'", // newsSourceIdCondition	string
		-1,              			                 // articlesPerCategory 	int
		20,     					                 // maxArticles 			int
		-1)             			                 // fetchVotesForUserId 	int64
}