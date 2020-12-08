//TODO: rename this to postQueries.go

package main

import (
	"errors"
	"fmt"
	"github.com/lib/pq"
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

type ArticleQueryParams struct {
	idCondition				string
	userIdCondition			string
	categoryCondition		string
	newsSourceIdCondition	string
	articlesPerCategory		int
	maxArticles				int
	fetchVotesForUserId		int64
	onlyPolls				bool
	noPolls					bool
	bRandomizeTime			bool
	addSemicolon			bool
	withinElapsedMilliseconds	int

	// Materialized view for /news.
	createMaterializedView	bool
	useMaterializedView		bool
	newsCycle				int		// Which materialized view to read
}

const (
	kMaterializedNewsView 	= "materializednewsview"  // Must be lowercase!

	// TODO: we need real dino art.
	// Leave most of these as PNG, as PNG can be smaller than JPG!  (And should be lossless.)
	kDefaultAuthorIcon		= "/static/dino-head-24x24.png"
	kDefaultThumbnail   	= "/static/dino-head-160x96.png"
	kDefaultImage       	= "/static/dino-head-546x386.png"

	// TODO: conver to PNG.
	kDefaultPollImage       	= "/static/ballotboxes/ballotbox-3dinos.jpg"
	kDefaultPollThumbnail 		= "/static/ballotboxes/ballotbox-3dinos-small.jpg"
	kDefaultPollThumbnailTall 	= "/static/ballotboxes/ballotbox-3dinos-small-160x180.jpg"


	kApproxCharsPerLine 	= 30
)


func defaultArticleQueryParams() (qp ArticleQueryParams) {
	qp.idCondition 				= "IS NOT NULL"
	qp.userIdCondition 	  		= "IS NOT NULL"
	qp.categoryCondition 	  	= "IS NOT NULL"
	qp.newsSourceIdCondition 	= "IS NOT NULL"
	qp.articlesPerCategory		= -1
	qp.maxArticles				= -1
	qp.fetchVotesForUserId		= int64(-1)
	qp.bRandomizeTime 			= flags.randomizeTime == "true"
	qp.addSemicolon				= false
	return
}

func (qp ArticleQueryParams) print() {
	/*
	prVal("idCondition", qp.idCondition)
	prVal("userIdCondition", qp.userIdCondition)
	prVal("categoryCondition", qp.categoryCondition)
	prVal("newsSourceIdCondition", qp.newsSourceIdCondition)
	prVal("articlesPerCategory", qp.articlesPerCategory)
	prVal("maxArticles", qp.maxArticles)
	prVal("fetchVotesForUserId", qp.fetchVotesForUserId)
	prVal("onlyPolls", qp.onlyPolls)
	prVal("noPolls", qp.noPolls)
	prVal("bRandomizeTime", qp.bRandomizeTime)
	prVal("createMaterializedView", qp.createMaterializedView)
	prVal("useMaterializedView", qp.useMaterializedView)
	*/
}

func (qp ArticleQueryParams) validate() {
	assertMsg(!(qp.createMaterializedView && qp.useMaterializedView),
		"Cannot create and user the materialized view at the same time.")

	assertMsg(ifthen(qp.createMaterializedView, qp.fetchVotesForUserId == -1),
		"Cannot materialize a table with anything tied to a specific user.")

	// Check we're creating and using the materialized queries with the same values.
	bMaterializable :=
	   qp.idCondition 			== "IS NOT NULL" &&
	   qp.userIdCondition 		== "IS NOT NULL" &&
	   qp.categoryCondition 	== "IS NOT NULL" &&
	   qp.newsSourceIdCondition == "IS NOT NULL" &&
	   qp.articlesPerCategory   == (kRowsPerCategory + 1) &&
	   qp.maxArticles 			== kMaxArticles &&
	   qp.onlyPolls			    == false &&
	   qp.noPolls			    == false
	assertMsg(ifthen(qp.createMaterializedView || qp.useMaterializedView, bMaterializable),
		`Can only create or user a materialized query if settings are correct,
		 and if settings are correct, the query should be materialized.`)
}

func (qp ArticleQueryParams) createBaseQuery() string {

	// Union of NewsPosts (News API) and LinkPosts (user articles).
	// OPT_TODO: don't test for all this NOT NULL stuff, suck that out of the query when building it.
	newsPostQuery := fmt.Sprintf(
	   `SELECT Id,
			   NewsSourceId AS Author,
			   -1::bigint AS UserId,
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
			   '' AS PollTallyResults,
			   NumComments,
			   ThumbnailStatus,
			   'N' AS Source
		FROM $$NewsPost
		WHERE ThumbnailStatus <> -1 AND (Id %s) AND (NewsSourceId %s) AND ($$GetCategory(Category, Country) %s)
			  AND UrlToImage != ''`, // << TODO: fetch og:Image for news posts with '' UrlToImage in newsService, as this is filtering out some news!
		qp.idCondition,
		qp.newsSourceIdCondition,
		qp.categoryCondition)

	linkPostQuery := fmt.Sprintf(
	   `SELECT P.Id,
			   U.Username AS Author,
			   UserId,
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
			   '' AS PollTallyResults,
			   NumComments,
			   ThumbnailStatus,
			   'L' AS Source
		FROM $$LinkPost P
		JOIN $$User U ON P.UserId = U.Id
		WHERE ThumbnailStatus <> -1 AND (P.Id %s) AND (U.Id %s) AND ($$GetCategory(Category, U.Country) %s)`,
		qp.idCondition,
		qp.userIdCondition,
		qp.categoryCondition)

	pollPostQuery := fmt.Sprintf(
	   `SELECT P.Id,
			   U.Username AS Author,
			   UserId,
			   P.Title,
			   '' AS Description,
			   FORMAT('/poll/?postId=%%s', P.Id) AS LinkUrl,
			   COALESCE(P.UrlToImage, '') AS UrlToImage,
			   P.Created AS PublishedAt,
			   '' AS NewsSourceId,
			   $$GetCategory(Category, U.Country) AS Category,
			   'EN' AS Language,
			   COALESCE(U.Country, ''),
			   PollOptionData,
			   COALESCE(PollTallyResults, ''),
			   NumComments,
			   ThumbnailStatus,
			   'P' AS Source
		FROM $$PollPost P
		JOIN $$User U ON P.UserId = U.Id
		WHERE (P.Id %s) AND (U.Id %s) AND ($$GetCategory(Category, U.Country) %s)`,	// Removed: 'ThumbnailStatus = 1 AND' because all polls currently use same thumbnail status
		qp.idCondition,
		qp.userIdCondition,
		qp.categoryCondition)

	if qp.withinElapsedMilliseconds > 0 {
		elapsedSeconds := qp.withinElapsedMilliseconds / 1000
		withinTimeInterval := "now() - " + int_to_str(elapsedSeconds) + " * (interval '1 second')"

		newsPostQuery += " AND COALESCE(PublishedAt, Created) > " + withinTimeInterval
		linkPostQuery += " AND P.Created > " + withinTimeInterval
		pollPostQuery += " AND P.Created > " + withinTimeInterval
	}

	// TODO: Optimize queries so we only create strings that we will actually use.
	query := ""
	if qp.onlyPolls {
		query = pollPostQuery
	} else if qp.userIdCondition != "IS NOT NULL" {  // Looking up posts that target a user - so there can be no news posts.
		query = strings.Join([]string{linkPostQuery, pollPostQuery}, "\nUNION ALL\n")
	} else if qp.newsSourceIdCondition != "IS NOT NULL" {  // We're just querying news posts.
		query = newsPostQuery
	} else if qp.noPolls {
		query = strings.Join([]string{newsPostQuery, linkPostQuery}, "\nUNION ALL\n")
	} else {
		query = strings.Join([]string{newsPostQuery, linkPostQuery, pollPostQuery}, "\nUNION ALL\n")
	}

	// Add vote tally and sort quality per article.
	// Sorting heuristic:
	//   (3 * NumVotes + 3 * NumUpVotes + .5 * NumComments) * (1 if you didn't vote yet, .04 if you have voted on it) + (random number from 0 to 8)
	// Testing decreased heuristic, TODO: test it out.
	query = fmt.Sprintf(`
		WITH posts AS (%s),
			 votes AS (
				SELECT PostId,
					   SUM(CASE WHEN Up THEN 1 ELSE -1 END) AS VoteTally
				FROM $$PostVote
				WHERE PostId IN (SELECT Id FROM posts)
				GROUP BY PostId
			 ),
			 pollVotes AS (
				SELECT PollId,
					   COUNT(*) AS VoteTally,
					   SUM(CASE WHEN UserId = (%d) THEN 1 ELSE 0 END) AS MyVotes
				FROM $$PollVote
				WHERE PollId IN (SELECT Id FROM posts WHERE Source = 'P')
				GROUP BY PollId
			 )
		SELECT posts.*,
			COALESCE(votes.VoteTally, 0) AS VoteTally,
			posts.PublishedAt +
				interval '24 hours' *
				(
					(
						2.5 * COALESCE(votes.VoteTally, 0) +
					 	2.5 * COALESCE(pollVotes.VoteTally, 0) +
						0.42 * posts.NumComments
					) * (1 - .96 * COALESCE(pollVotes.MyVotes, 0)) +
					10 * (%s)
				) AS OrderBy
		FROM posts
		LEFT JOIN votes ON posts.Id = votes.PostId
		LEFT JOIN pollVotes ON posts.Id = pollVotes.PollId
		ORDER BY OrderBy DESC`,
		query,
		qp.fetchVotesForUserId,
		ternary_str(qp.bRandomizeTime, "RANDOM()", "0"))

	if qp.articlesPerCategory > 0 {
		// Select 5 articles of each category
		query = fmt.Sprintf(`
			SELECT Id,
				Author,
				UserId,
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
				PollTallyResults,
				NumComments,
				ThumbnailStatus,
				Source,
				VoteTally,
				OrderBy
			FROM (
				SELECT
					*,
					ROW_NUMBER() OVER (
						PARTITION BY Category
						ORDER BY OrderBy DESC) AS r
				FROM (%s) x
			) x
			WHERE x.r <= %d`,
			query,
			qp.articlesPerCategory) // DEAD HACK: OR (x.Category = 'polls' AND x.r <= %d)`, // Hack: polls can fit 2 more because none are headlines.
	} else {
		query = fmt.Sprintf(`
			SELECT
				*
			FROM (%s) x
			ORDER BY x.OrderBy DESC`,
			query)
	}

	if qp.maxArticles > 0 {
		query += "\nLIMIT " + strconv.Itoa(qp.maxArticles)
	}

	if qp.addSemicolon {
		query += `;`
	}

	return query
}
//////////////////////////////////////////////////////////////////////////////
//
// Create the materializable query for articles - which can either be use to either
// materialize this expensive query (~509ms), or to join it later to the user-specific
// data, the non-materialized way.
//
//////////////////////////////////////////////////////////////////////////////
func (qp ArticleQueryParams) createArticleQuery() string {

	startTimer("createArticleQuery")

	pr("createArticleQuery")
	qp.print()
	qp.validate()

	query := ""
	if qp.createMaterializedView {
		// Create materialized view
		query = "CREATE MATERIALIZED VIEW " + kMaterializedNewsView + int_to_str(qp.newsCycle) + " AS " + qp.createBaseQuery()
	} else if qp.useMaterializedView {
		// Use materialized view in query
		query = kMaterializedNewsView + int_to_str(qp.newsCycle)
	} else {
		// Oldschool, unoptimized query - not materialized view.
		query = qp.createBaseQuery()
	}

	// This part is not materializable, since it joins to a specific user!!!
	if qp.fetchVotesForUserId >= 0 {
		// Join query to post vote and poll vote tables.
		query = fmt.Sprintf(`
			SELECT x.*,
			   CASE WHEN v.Up IS NULL THEN 0
					WHEN v.Up THEN 1
					ELSE -1
			   END AS Upvoted,
			   w.VoteOptionIds
			FROM %s x
			LEFT JOIN $$PostVote v ON x.Id = v.PostId AND (v.UserId = %d)
			LEFT JOIN $$PollVote w ON x.Id = w.PollId AND (w.UserId = %d)
			ORDER BY x.OrderBy DESC`,
			ternary_str(qp.useMaterializedView, query, "(\n" + query + "\n)"),
			qp.fetchVotesForUserId,
			qp.fetchVotesForUserId)
	} else if qp.useMaterializedView {
		query = fmt.Sprintf("SELECT * FROM %s ORDER BY OrderBy DESC", query)
	}

	if qp.addSemicolon {
		query += `;`
	}

	endTimer("createArticleQuery")

	return query
}

//////////////////////////////////////////////////////////////////////////////
//
// query news articles and user posts from database, with condition test on the
// id, category, and optional partitioning per category.
// If articlesPerCategory <= 0, no category partitioning takes place.
//
//////////////////////////////////////////////////////////////////////////////
func queryArticles(qp ArticleQueryParams) (articles []Article) {
	startTimer("queryArticles")
	startTimer("doQuery")

	pr("queryArticles")

	query := qp.createArticleQuery()

	qp.print()

	rows := DbQuery(query)

	endTimer("doQuery")

	startTimer("scanRows")
	checkForDupId := map[int64]bool{}
	for rows.Next() {
		var id				int64
		var author			string
		var userId			int64
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
		var pollTallyResultsJson	string
		var orderBy			time.Time
		var upvoted			int
		var voteTally		int
		var numComments		int
		var thumbnailStatus	int
		var source			string
		var voteOptionIds 	[]int64

		if qp.fetchVotesForUserId >= 0 {
			check(rows.Scan(&id, &author, &userId, &title, &description, &linkUrl, &urlToImage,
							&publishedAt, &newsSourceId, &category, &language, &country,
							&pollOptionJson, &pollTallyResultsJson, &numComments, &thumbnailStatus, &source,
							&voteTally, &orderBy, &upvoted, pq.Array(&voteOptionIds)))

		} else {
			check(rows.Scan(&id, &author, &userId, &title, &description, &linkUrl, &urlToImage,
							&publishedAt, &newsSourceId, &category, &language, &country,
							&pollOptionJson, &pollTallyResultsJson, &numComments, &thumbnailStatus, &source,
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
		//prVal("pollTallyResultsJson", pollTallyResultsJson)
		//prVal("orderBy", orderBy)
		//prVal("upvoted", upvoted)
		//prVal("voteOptionIds", voteOptionIds)
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

		// Map the category to one that makes sense.
		category, found := newsCategoryRemapping[category]
		if !found {
			category = "other"
		}

		// Set the article information.
		newArticle := Article{
			Id:				id,
			Author:			author, // haha hijacking Author to be the poster
			UserId:			userId,
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
			TimeSince:		getTimeSinceString(publishedAt, false),
			AuthorIconUrl:
				ternary_str(newsSourceId != "",
					"/static/newsSourceIcons/" + newsSourceId + ".png",  // News source icon.
					kDefaultAuthorIcon),
			Upvoted:		upvoted,
			VoteOptionIds:	voteOptionIds,
			VoteTally:		voteTally,
			NumComments:	numComments,
			ThumbnailStatus:thumbnailStatus,
		}

		// Handle polls.
		if len(pollOptionJson) > 0 {
			//prf("pollId %d category %s", newArticle.Id, category)

			newArticle.IsPoll  = true

			check(json.Unmarshal([]byte(pollOptionJson), &newArticle.PollOptionData))

			// If this poll has already been voted on...
			if len(pollTallyResultsJson) > 0 {
				check(json.Unmarshal([]byte(pollTallyResultsJson), &newArticle.PollTallyInfo.Stats))

				newArticle.PollTallyInfo.SetArticle(&newArticle)

				// Force AnyoneCanAddOptions to be true, otherwise ppl make closed polls that don't get everyone's opinion.
				newArticle.PollOptionData.AnyoneCanAddOptions = true

				//prVal("newArticle.PollTallyResults", newArticle.PollTallyResults)

				// vv Keep this code here!!! It must always be called, particularly when user is logged out, or there'll be a crash
				// in /viewPollResults.
				numOptions := len(newArticle.PollOptionData.Options)
				newArticle.VoteData = make([]bool, numOptions)

				if len(voteOptionIds) > 0 {
					newArticle.WeVoted = true

					// Convert voteOptionIds to map to make it easily lookupable by the html template.
					//newArticle.VoteOptionsMap = make(map[int64]bool)
					//for _, optionId := range voteOptionIds {
					//	newArticle.VoteOptionsMap[optionId] = true
					//}
					//prVal("voteOptionIds", voteOptionIds)

					for _, optionId := range voteOptionIds {
						// Since /news it cached and could be 3 mins old, we have to do this check.
						if optionId < int64(len(newArticle.VoteData)) {
							newArticle.VoteData[optionId] = true
						}
					}
				}
			}

			//newArticle.Title   = ternary_str(newArticle.WeVoted, "RESULTS: ", "POLL: ") + newArticle.Title

			// Tally title characters.
			numLinesApprox := ceil_div(len(newArticle.Title), kApproxCharsPerLine)

			// Tally poll options separately (because divided by a newline).
			longestItem := 0 // Calc the longest item length.
			numCharsApprox := 2 * len(newArticle.PollOptionData.Options)  // Treat checkbox as 2 characters.
			for _, option := range newArticle.PollOptionData.Options {
				length := len(option)
				numCharsApprox += length

				longestItem = max_int(longestItem, length)
			}
			numLinesApprox += ceil_div(numCharsApprox, kApproxCharsPerLine)
			newArticle.LongestItem = longestItem

			//prf("numCharsApprox: %d numLinesApprox: %d", numCharsApprox, numLinesApprox)

			//newArticle.UrlToImage 	  = fmt.Sprintf("/static/ballotboxes/%d.jpg", rand.Intn(17)) // Pick a random ballotbox image.
			//newArticle.UrlToThumbnail = newArticle.UrlToImage
			newArticle.UrlToImage 	  = kDefaultPollImage
			newArticle.UrlToThumbnail = ternary_str(numLinesApprox <= 1,
												    kDefaultPollThumbnail,
													kDefaultPollThumbnailTall)

			// TODO: this is a dup of the code below it.
			if thumbnailStatus >= image_DownsampledV2 {
				thumbnailBasePath := "/static/thumbnails/" + strconv.FormatInt(id, 10)

				// v2+ - Downsamples into two version of the thumbnail, different heights depending on the height of the article.  (New version 2 of the thumbnail.)  TODO: maybe we can pick a or b ahead of time?
				newArticle.UrlToThumbnail = ternary_str(numLinesApprox <= 2,
					thumbnailBasePath + "a.jpeg", // a - 160 x 116 - thumbnail
					thumbnailBasePath + "b.jpeg") // b - 160 x 150

				// v3  - Point full-size image to large thumbnail.
				if thumbnailStatus >= image_DownsampledV3 {
					newArticle.UrlToImage = thumbnailBasePath + "c.jpeg" // c - 570 x _ [large thumbnail]
				}
			}

			//prVal("newArticle.PollOptionData", newArticle.PollOptionData)

			newArticle.Url = fmt.Sprintf("/article/?postId=%d#vote", id)
			//newArticle.Url = fmt.Sprintf("/viewPollResults/?postId=%d", id)  // This would take the user to directly viewing the results.  This is a design problem to figure out later.

			newArticle.NumLines = numLinesApprox
		} else { // Handle non-polls.
			numCharsApprox := len(newArticle.Title)

			numLinesApprox := ceil_div(numCharsApprox, kApproxCharsPerLine)

			//prf("numLines: %d title: %s", numLinesApprox, newArticle.Title)

			newArticle.NumLines = numLinesApprox

			newArticle.UrlToImage = coalesce_str(urlToImage, kDefaultImage)  // Full-size image.

			if urlToImage == ""	{
				// Dropback if no image.  (TODO: replace licensed art.)
				newArticle.UrlToThumbnail = kDefaultThumbnail
				newArticle.IsThumbnail = true
			} else if thumbnailStatus == image_Unprocessed {
				// Uses full-size image as backup if thumbnail isn't processed yet.
				newArticle.UrlToThumbnail = urlToImage
			} else if thumbnailStatus >= image_DownsampledV2 {
				thumbnailBasePath := "/static/thumbnails/" + strconv.FormatInt(id, 10)

				// v2+ - Downsamples into two version of the thumbnail, different heights depending on the height of the article.  (New version 2 of the thumbnail.)  TODO: maybe we can pick a or b ahead of time?
				newArticle.UrlToThumbnail = ternary_str(numLinesApprox <= 2,
					thumbnailBasePath + "a.jpeg", // a - 160 x 116 - thumbnail
					thumbnailBasePath + "b.jpeg") // b - 160 x 150

				// v3  - Point full-size image to large thumbnail.
				if thumbnailStatus >= image_DownsampledV3 {
					newArticle.UrlToImage = thumbnailBasePath + "c.jpeg" // c - 570 x _ [large thumbnail]
				}
			} else if thumbnailStatus == image_Downsampled {
				// Old version of the thumbnail.
				newArticle.UrlToThumbnail =
					"/static/thumbnails/" + strconv.FormatInt(id, 10) + ".jpeg"
			} else {
				//prVal("image_DownsampledV2", image_DownsampledV2)
				panic(fmt.Sprintf("Unexpected thumbnail status: %d", thumbnailStatus))
			}
		}

		//prVal("newArticle.UrlToImage",newArticle.UrlToImage)
		//prVal("newArticle.UrlToThumbnail", newArticle.UrlToThumbnail)

		newArticle.Ellipsify = func(text string, maxLength int) string { return ellipsify(text, maxLength); }


		// Check for articles with duplicate id's.  When polls have duplicate id's, it causes voting bugs!!!
		_, found = checkForDupId[id]
		if found {
			prf("Found post with duplicate id: %d!!!", id)
			continue
		}
		checkForDupId[id] = true

		articles = append(articles, newArticle)
	}
	check(rows.Err())
	rows.Close()

	endTimer("scanRows")
	endTimer("queryArticles")

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
	qp := defaultArticleQueryParams()
	qp.idCondition 			= "= " + strconv.FormatInt(id, 10)
	qp.maxArticles 			= 2			// so we could potentially catch duplicate articles.
	qp.fetchVotesForUserId 	= userId	// int64

	articles := queryArticles(qp)

	len := len(articles)

	if len == 1 {
		articles[0].Size = 2  // 2 means full-page article in /article or /viewPollResults.
		return articles[0], nil
	} else if len == 0 {
		return Article{}, errors.New("Article not found")
	} else {
		return Article{}, errors.New("Duplicate articles found")
	}
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch news articles.  We'll use this to help build the activity / notifications.
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticles(articlesPerCategory int, userId int64, maxArticles int) ([]Article) {
	qp := defaultArticleQueryParams()
	qp.maxArticles 			= maxArticles
	qp.fetchVotesForUserId	= userId
	return queryArticles(qp)
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch news articles for /news -
//   partitioned by category, up to articlesPerCategory articles per category, up to maxArticles total.
//
//////////////////////////////////////////////////////////////////////////////
func fetchNews(articlesPerCategory int, userId int64, maxArticles, newsCycle int, noPolls, isCacheValid bool) ([]Article) {
	qp := defaultArticleQueryParams()
	qp.useMaterializedView = isCacheValid
	qp.articlesPerCategory = articlesPerCategory
	qp.maxArticles		   = maxArticles
	qp.fetchVotesForUserId = userId
	qp.newsCycle		   = newsCycle
	qp.noPolls			   = noPolls
	return queryArticles(qp)
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch articles comented on by a user.
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticlesCommentedOnByUser(creatorUserId, voterUserId int64, maxArticles int) ([]Article) {
	qp := defaultArticleQueryParams()
	qp.idCondition 			= "IN (SELECT PostId FROM $$Comment WHERE UserId = " + strconv.FormatInt(creatorUserId, 10) + ")"
	qp.maxArticles 			= maxArticles
	qp.fetchVotesForUserId 	= voterUserId
	return queryArticles(qp)
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch polls voted on by a user.
//
//////////////////////////////////////////////////////////////////////////////
func fetchPollsVotedOnByUser(creatorUserId, voterUserId int64, maxArticles int) ([]Article) {
	qp := defaultArticleQueryParams()
	qp.idCondition 			= "IN (SELECT PollId FROM $$PollVote WHERE UserId = " + strconv.FormatInt(creatorUserId, 10) + ")"
	qp.maxArticles 			= maxArticles
	qp.fetchVotesForUserId 	= voterUserId
	return queryArticles(qp)
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch articles within a particular category, up to maxArticles total,
// which userId voted on.
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticlesUpDownVotedOnByUser(creatorUserId, voterUserId int64, maxArticles int) ([]Article) {
	qp := defaultArticleQueryParams()
	qp.idCondition 			= "IN (SELECT PostId FROM $$PostVote WHERE UserId = " + strconv.FormatInt(creatorUserId, 10) + ")"
	qp.maxArticles 			= maxArticles
	qp.fetchVotesForUserId 	= voterUserId
	return queryArticles(qp)
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch articles within a particular category, up to maxArticles total,
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticlesWithinCategory(category string, userId int64, maxArticles int) ([]Article) {
	qp := defaultArticleQueryParams()
	_, foundCategory := newsCategoryInfo.HeaderColors[category]  // Ensure we have a valid category (to prevent SQL injection).

	if foundCategory {
		qp.categoryCondition 	= "= '" + sqlEscapeString(category) + "'"
		qp.maxArticles 			= maxArticles
		qp.fetchVotesForUserId 	= userId
		return queryArticles(qp)
	} else {
		//prVal("Unknown category", category)
		return []Article{}
	}
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch articles posted by a user.
//   category - optional, can provide "" to skip.
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticlesPostedByUser(creatorUserId, voterUserId int64, maxArticles int) ([]Article) {
	qp := defaultArticleQueryParams()
	qp.userIdCondition		= "= " + strconv.FormatInt(creatorUserId, 10)
	qp.maxArticles			= maxArticles
	qp.fetchVotesForUserId	= voterUserId
	return queryArticles(qp)
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch articles not posted by a user.
//   category - optional, can provide "" to skip.
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticlesNotPostedByUser(userId int64, maxArticles, withinElapsedMilliseconds int) ([]Article) {
	qp := defaultArticleQueryParams()
	qp.userIdCondition			 = "<> " + strconv.FormatInt(userId, 10)
	qp.maxArticles				 = maxArticles
	qp.fetchVotesForUserId		 = userId
	qp.withinElapsedMilliseconds = withinElapsedMilliseconds
	return queryArticles(qp)
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch articles from a news source
//
//////////////////////////////////////////////////////////////////////////////
func fetchArticlesFromThisNewsSource(newsSourceId string, userId, skipArticleId int64,
									 maxArticles int) (articles []Article) {
	_, isNewsSource := newsSourceList[newsSourceId]
	if isNewsSource {

		qp := defaultArticleQueryParams()
		if skipArticleId >= 0 {
			qp.idCondition = "!= " + strconv.FormatInt(skipArticleId, 10)
		}
		qp.newsSourceIdCondition = "= '" + sqlEscapeString(newsSourceId) + "'"
		qp.maxArticles           = maxArticles
		qp.fetchVotesForUserId   = userId
		return queryArticles(qp)
	} else {
		pr("Invalid news source!")
		return []Article{}
	}
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch polls
//
//////////////////////////////////////////////////////////////////////////////
func fetchPolls(userId int64, maxArticles int) (articles []Article) {
	qp := defaultArticleQueryParams()
	qp.maxArticles			= maxArticles
	qp.fetchVotesForUserId	= userId
	qp.onlyPolls			= true
	return queryArticles(qp)
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch suggested polls
//
//////////////////////////////////////////////////////////////////////////////
func fetchSuggestedPolls(userId, skipArticleId int64) (articles []Article) {
	qp := defaultArticleQueryParams()
	if skipArticleId > 0 {
		qp.idCondition = "!= " + strconv.FormatInt(skipArticleId, 10)
	}
	qp.maxArticles			= 5
	qp.fetchVotesForUserId	= userId
	qp.onlyPolls			= true
	return queryArticles(qp)
}

//////////////////////////////////////////////////////////////////////////////
//
// fetch polls posted by user
//
//////////////////////////////////////////////////////////////////////////////
func fetchPollsPostedByUser(userId int64, maxArticles int) (articles []Article) {
	qp := defaultArticleQueryParams()
	qp.userIdCondition		= "= " + strconv.FormatInt(userId, 10)
	qp.maxArticles			= maxArticles
	qp.fetchVotesForUserId	= userId
	qp.onlyPolls			= true
	return queryArticles(qp)
}
