package main

import (
	"errors"
	"fmt"
//	"math/rand"
	"net/url"
	"strconv"
	"strings"
	"time"

	//"database/sql/driver"
	_ "github.com/lib/pq"
	//"github.com/jmoiron/sqlx"
	"encoding/json"
)

const (
	kDefaultImage       = "/static/mozilla dinosaur head.png"
	kDefaultThumbnail   = "/static/mozilla dinosaur thumbnail.png"
	kApproxCharsPerLine = 30
)


//////////////////////////////////////////////////////////////////////////////
//
// query news articles and user posts from database, with condition test on the
// id, category, and optional partitioning per category.
// If articlesPerCategory <= 0, no category partitioning takes place.
//
//////////////////////////////////////////////////////////////////////////////
func _queryArticles(idCondition string, userIdCondition string, categoryCondition string, newsSourceIdCondition string,
					articlesPerCategory int, maxArticles int, fetchVotesForUserId int64, onlyPolls bool) (articles []Article) {
	pr("_queryArticles")
	prVal("idCondition", idCondition)
	prVal("userIdCondition", userIdCondition)
	prVal("categoryCondition", categoryCondition)
	prVal("newsSourceIdCondition", newsSourceIdCondition)
	prVal("articlesPerCategory", articlesPerCategory)
	prVal("maxArticles", maxArticles)
	prVal("fetchVotesForUserId", fetchVotesForUserId)

	bRandomizeTime := true
	//bRandomizeTime := (fetchVotesForUserId == -1)

	// Union of NewsPosts (News API) and LinkPosts (user articles).
	newsPostQuery := fmt.Sprintf(
	   `SELECT Id,
	   		   NewsSourceId AS Author,
	   		   Title,
	   		   COALESCE(Description, '') AS Description,
	   		   LinkUrl,
	   		   COALESCE(UrlToImage, '') AS UrlToImage,
	   		   COALESCE(PublishedAt, Created) AS PublishedAt,
	   		   NewsSourceId,
	   		   $$GetCategory(Category, Country) AS Category,
	   		   Language,
			   Country,
			   '' AS PollOptionData,
			   NumComments,
			   ThumbnailStatus,
			   'N' AS Source
		FROM $$NewsPost
		WHERE ThumbnailStatus <> -1 AND (Id %s) AND (NewsSourceId %s) AND ($$GetCategory(Category, Country) %s)
		      AND UrlToImage != ''`, // << TODO: fetch og:Image for news posts with '' UrlToImage in newsService, as this is filtering out some news!
		idCondition,
		newsSourceIdCondition,
		categoryCondition)

	linkPostQuery := fmt.Sprintf(
	   `SELECT P.Id,
		       U.Username AS Author,
		       P.Title,
		       '' AS Description,
		       P.LinkUrl,
			   COALESCE(P.UrlToImage, '') AS UrlToImage,
			   P.Created AS PublishedAt,
			   '' AS NewsSourceId,
			   $$GetCategory(Category, U.Country) AS Category,
			   'EN' AS Language,
			   COALESCE(U.Country, ''),
			   '' AS PollOptionData,
			   NumComments,
			   ThumbnailStatus,
			   'L' AS Source
		FROM $$LinkPost P
		JOIN $$User U ON P.UserId = U.Id
		WHERE ThumbnailStatus <> -1 AND (P.Id %s) AND (U.Id %s) AND ($$GetCategory(Category, U.Country) %s)`,
		idCondition,
		userIdCondition,
		categoryCondition)

	pollPostQueryBuilder := func(pollsCategory bool) string {
		return fmt.Sprintf(
		   `SELECT P.Id,
				   U.Username AS Author,
				   P.Title,
				   '' AS Description,
				   FORMAT('/poll/?postId=%%s', P.Id) AS LinkUrl,
				   COALESCE(P.UrlToImage, '') AS UrlToImage,
				   P.Created AS PublishedAt,
				   '' AS NewsSourceId,
				   %s AS Category,
				   'EN' AS Language,
				   COALESCE(U.Country, ''),
				   PollOptionData,
				   NumComments,
				   ThumbnailStatus,
				   'P' AS Source
			FROM $$PollPost P
			JOIN $$User U ON P.UserId = U.Id
			WHERE (P.Id %s) AND (U.Id %s) AND ($$GetCategory(Category, U.Country) %s)`,	// Removed: 'ThumbnailStatus = 1 AND' because all polls currently use same thumbnail status
			ternary_str(pollsCategory, "'polls'", "$$GetCategory(Category, U.Country)"),
			idCondition,
			userIdCondition,
			categoryCondition)
	}

	pollPostQuery := pollPostQueryBuilder(false)

	pollCatQuery := pollPostQueryBuilder(true)

	// TODO: Optimize queries so we only create strings that we will actually use.
	query := ""
	if userIdCondition != "IS NOT NULL" {  // Looking up posts that target a user - so there can be no news posts.
		query = strings.Join([]string{linkPostQuery, pollPostQuery}, "\nUNION ALL\n")
	} else if newsSourceIdCondition != "IS NOT NULL" {  // We're just querying news posts.
		query = newsPostQuery
	} else if onlyPolls {
		query = pollCatQuery
	// Removing this since pollCatQuery alongside pollPostQuery causes duplicate polls, which causes voting bugs!
	//} else if articlesPerCategory > 0 {
	//	query = strings.Join([]string{newsPostQuery, linkPostQuery, pollPostQuery, pollCatQuery}, "\nUNION ALL\n")
	} else {
		query = strings.Join([]string{newsPostQuery, linkPostQuery, pollPostQuery}, "\nUNION ALL\n")
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
		SELECT posts.*,
			COALESCE(votes.VoteTally, 0) AS VoteTally,
			posts.PublishedAt +
				interval '24 hours' *
				(
					3 * COALESCE(votes.VoteTally, 0) +
					0.5 * posts.NumComments +
					5 * (%s)
				) AS OrderBy
		FROM posts
		LEFT JOIN votes ON posts.Id = votes.PostId
		ORDER BY OrderBy DESC`,
		query,
		ternary_str(bRandomizeTime, "RANDOM()", "0"))

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
				NumComments,
				ThumbnailStatus,
				Source,
				VoteTally,
				OrderBy
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
			ORDER BY x.OrderBy DESC`,
			query,
			fetchVotesForUserId)
	}

	if maxArticles > 0 {
		query += "\nLIMIT " + strconv.Itoa(maxArticles)
	}

	query += `;`

	checkForDupId := map[int64]bool{}

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
							&publishedAt, &newsSourceId, &category, &language, &country, &pollOptionJson, &numComments, &thumbnailStatus, &source,
							&voteTally, &orderBy, &upvoted))
		} else {
			check(rows.Scan(&id, &author, &title, &description, &linkUrl, &urlToImage,
							&publishedAt, &newsSourceId, &category, &language, &country, &pollOptionJson, &numComments, &thumbnailStatus, &source,
							&voteTally, &orderBy))
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
			years := days / 365.0

			if years > 20.0 {
				timeSinceStr = "old"
			} else if years >= 1.0 {
				timeSinceStr = strconv.FormatFloat(years, 'f', 0, 32) + "y"
			} else if weeks >= 1.0 {
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

		// Map the category to one that makes sense.
		category, found := newsCategoryRemapping[category]
		if !found {
			category = "other"
		}

		// Set the article information.
		newArticle := Article{
			Id:				id,
			Author:			author, // haha hijacking Author to be the poster
			Title:			title,
			Description:	description,
			Url:			linkUrl,
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
			ThumbnailStatus:thumbnailStatus,
		}

		// Handle polls.
		if len(pollOptionJson) > 0 {
			//prf("pollId %d category %s", newArticle.Id, category)

			newArticle.IsPoll 		  = true
			newArticle.Title 		  = "POLL: " + newArticle.Title

			check(json.Unmarshal([]byte(pollOptionJson), &newArticle.PollOptionData))

			// Tally title characters.
			numLinesApprox := ceil_div(len(newArticle.Title), kApproxCharsPerLine)

			// Tally poll options separately (because divided by a newline).
			numCharsApprox := 2 * len(newArticle.PollOptionData.Options)  // Treat checkbox as 2 characters.
			for _, option := range newArticle.PollOptionData.Options {
				numCharsApprox += len(option)
			}
			numLinesApprox += ceil_div(numCharsApprox, kApproxCharsPerLine)

			//prf("numCharsApprox: %d numLinesApprox: %d", numCharsApprox, numLinesApprox)

			//newArticle.UrlToImage 	  = fmt.Sprintf("/static/ballotboxes/%d.jpg", rand.Intn(17)) // Pick a random ballotbox image.
			//newArticle.UrlToThumbnail = newArticle.UrlToImage
			newArticle.UrlToImage 	  = "/static/ballotboxes/ballotbox 3dinos.jpg"
			newArticle.UrlToThumbnail = ternary_str(numLinesApprox <= 1,//2,
												    "/static/ballotboxes/ballotbox 3dinos small.jpg",
													"/static/ballotboxes/ballotbox 3dinos small 160x180.jpg")

			//prVal("newArticle.PollOptionData", newArticle.PollOptionData)

			newArticle.Url = fmt.Sprintf("/article/?postId=%d", id)
			//newArticle.Url = fmt.Sprintf("/viewPollResults2/?postId=%d", id)  // This would take the user to directly viewing the results.  This is a design problem to figure out later.

			newArticle.NumLines = numLinesApprox
		} else { // Handle non-polls.
			numCharsApprox := len(newArticle.Title)

			numLinesApprox := ceil_div(numCharsApprox, kApproxCharsPerLine)

			//prf("numLines: %d title: %s", numLinesApprox, newArticle.Title)

			newArticle.NumLines = numLinesApprox

			newArticle.UrlToImage = coalesce_str(urlToImage, kDefaultImage)		  // Full-size image.  Uses a default image (mozilla head) as backup.

			if urlToImage == ""	{
				// Dropback if no image.  (TODO: replace licensed art.)
				newArticle.UrlToThumbnail = kDefaultThumbnail
				newArticle.IsThumbnail = true
			} else if thumbnailStatus == image_Unprocessed {
				// Uses full-size image as backup if thumbnail isn't processed yet, or default thumbnail (tiny mozilla head) as backup if image is missing.
				newArticle.UrlToThumbnail = urlToImage
			} else if thumbnailStatus == image_DownsampledV2 {
				// Downsamples into two version of the thumbnail, different heights depending on the height of the article.  (New version 2 of the thumbnail.)  TODO: maybe we can pick a or b ahead of time?
				newArticle.UrlToThumbnail = ternary_str(numLinesApprox <= 2,
					"/static/thumbnails/" + strconv.FormatInt(id, 10) + "a.jpeg", // a - 160 x 116 - thumbnail
					"/static/thumbnails/" + strconv.FormatInt(id, 10) + "b.jpeg") // b - 160 x 150
			} else if thumbnailStatus == image_Downsampled {
				// Old version of the thumbnail.
				newArticle.UrlToThumbnail =
					"/static/thumbnails/" + strconv.FormatInt(id, 10) + ".jpeg"
			} else {
				//prVal("image_DownsampledV2", image_DownsampledV2)
				panic(fmt.Sprintf("Unexpected thumbnail status: %d", thumbnailStatus))
			}
		}

		// Check for articles with duplicate id's.  When polls have duplicate id's, it causes voting bugs!!!
		_, found = checkForDupId[id]
		assertMsg(!found, fmt.Sprintf("Found post with duplicate id: %d", id))
		checkForDupId[id] = true

		articles = append(articles, newArticle)
	}
	check(rows.Err())
	rows.Close()

	prVal("checkForDupId", checkForDupId)
	//prVal("len(articles)", len(articles))

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
		userId,					 		  // fetchVotesForUserId 	int64
		false)							  // onlyPolls				bool

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
func fetchArticlesPartitionedByCategory(articlesPerCategory int, userId int64, maxArticles int) ([]Article) {
	return _queryArticles(
		"IS NOT NULL",        // idCondition
		"IS NOT NULL",        // userIdCondition
		"IS NOT NULL",        // categoryCondition
		"IS NOT NULL",		  // newsSourceIdCondition	string
		articlesPerCategory,  // articlesPerCategory 	int
		maxArticles,		  // maxArticles 			int
		userId,				  // fetchVotesForUserId 	int64
		false)				  // onlyPolls				bool
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
		userId,																					  // fetchVotesForUserId 	int64
		false)							 														  // onlyPolls				bool
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
		userId,																						// fetchVotesForUserId 	int64
		false)							 															// onlyPolls				bool
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch articles within a particular category, up to maxArticles total,
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticlesWithinCategory(category string, userId int64, maxArticles int) ([]Article) {
	_, foundCategory := newsCategoryInfo.HeaderColors[category]  // Ensure we have a valid category (to prevent SQL injection).

	if foundCategory {
		return _queryArticles(
			"IS NOT NULL",                         // idCondition
			"IS NOT NULL",	                       // userIdCondition
			"= '" + category + "'",                // categoryCondition
			"IS NOT NULL",						   // newsSourceIdCondition	string
			-1,									   // articlesPerCategory 	int
			maxArticles,						   // maxArticles 			int
			userId,								   // fetchVotesForUserId 	int64
			false)							 	   // onlyPolls				bool
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
		userId,								   // fetchVotesForUserId 	int64
		false)							 	   // onlyPolls				bool
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch articles from a news source
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticlesFromThisNewsSource(newsSourceId string, userId, skipArticleId int64) (articles []Article) {
	// TODO_SECURITY: add additional check for newsSourceId within known news sources.

	return _queryArticles(
		ternary_str(skipArticleId > 0,
			"!= " + strconv.FormatInt(skipArticleId, 10),  // idCondition 		string
			"IS NOT NULL"), 				               // idCondition 		string
		"IS NOT NULL",   			                 // userIdCondition 		string
		"IS NOT NULL",   			                 // categoryCondition 	    string
		"= '" + sqlEscapeString(newsSourceId) + "'", // newsSourceIdCondition	string
		-1,              			                 // articlesPerCategory 	int
		10,     					                 // maxArticles 			int
		userId,            			                 // fetchVotesForUserId 	int64
		false)										 // onlyPolls				bool
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch polls
//
//////////////////////////////////////////////////////////////////////////////
func fetchPolls(userId int64, maxArticles int) (articles []Article) {
	return _queryArticles(
		"IS NOT NULL",	// idCondition				string
		"IS NOT NULL",  // userIdCondition 			string
		"IS NOT NULL",  // categoryCondition 	    string
		"IS NOT NULL",	// newsSourceIdCondition	string
		-1,             // articlesPerCategory 		int
		maxArticles,    // maxArticles 				int
		userId,         // fetchVotesForUserId 		int64
		true)			// onlyPolls				bool
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch suggested polls
//
//////////////////////////////////////////////////////////////////////////////
func fetchSuggestedPolls(userId, skipArticleId int64) (articles []Article) {
	return _queryArticles(
		ternary_str(skipArticleId > 0,                     // idCondition 		string
			"!= " + strconv.FormatInt(skipArticleId, 10),
			"IS NOT NULL"),
		"IS NOT NULL",  // userIdCondition 			string
		"IS NOT NULL",  // categoryCondition 	    string
		"IS NOT NULL",	// newsSourceIdCondition	string
		-1,             // articlesPerCategory 		int
		5,     			// maxArticles 				int
		userId,         // fetchVotesForUserId 		int64
		true)			// onlyPolls				bool
}
