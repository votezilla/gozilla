package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)


///////////////////////////////////////////////////////////////////////////////
//
// AJAX - Check for notifications
//
///////////////////////////////////////////////////////////////////////////////
func ajaxCheckForNotifications(w http.ResponseWriter, r *http.Request) {
	if !flags.checkForNotifications{
		serveHtml(w, "Flag CheckForNotifications disabled")
		return
	}

	pr("ajaxCheckForNotifications")
	prVal("r.Method", r.Method)

	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	userId := GetSession(w, r);
/*	if userId == -1 { // Secure cookie not found.  Either session expired, or someone is hacking.
		// So go to the register page.
		pr("Must be logged in to get notifications.")
		serveErrorMsg(w, "Must be logged in to get notifications.")
		return
	}
*/

    //parse request to struct
    var request struct {
		NotificationType		string
		ElapsedMilliseconds		int
	}

    err := json.NewDecoder(r.Body).Decode(&request)
    if err != nil {
		prVal("Failed to decode json body", r.Body)
        serveError(w, err)
        return
    }

	var response struct {
		NumNotifications	int
	}

	switch request.NotificationType[0] {
		case 'n': // "news"
			pr("  request.NotificationType = 'news'")
			elapsedSeconds := request.ElapsedMilliseconds / 1000
			withinTimeInterval := "now() - " + int_to_str(elapsedSeconds) + " * (interval '1 second')"

			response.NumNotifications = DbQueryCount(
			   `SELECT COUNT(*) FROM $$Post
				WHERE Created > `+ withinTimeInterval + ";")

			break;
		case 'a': // "activity"
			pr("  request.NotificationType = 'activity'")
			_, _, _, unvisited := fetchActivity(userId, request.ElapsedMilliseconds) // CHECKIN_TODO: only fetch activity within the last ___ seconds!

			prVal("  unvisited", unvisited)

			// Add up the unvisited notifications - this is the update number.
			response.NumNotifications = 0
			for _, b := range unvisited {
				if b {
					response.NumNotifications++
				}
			}

			prVal("  response.NumNotifications", response.NumNotifications)

			break;
		default:
			assert(false)
	}


    prVal("response", response)

    // create json response from struct
    a, err := json.Marshal(response)
    if err != nil {
        serveError(w, err)
        return
    }
    w.Write(a)
}

/* POSSIBLE ACTIVITY OUTPUT:
   Get polls voted on by user
     Poll 'Favorite letter?' now has X votes
     Poll 'What are some of your favorite comedy movies?' now has X votes
     Poll 'What's your favorite "3 Stooges" stooge?' now has X votes
     Poll 'Rock, paper, or scissors?' now has X votes
     Poll '2 + 2 = __________' now has X votes
     Poll 'Rank or file?' now has X votes
   Get articles posted by user
     newish690 posted a new article 'Rock, paper, or scissors?'
     newish690 posted a new article 'What is your favorite color?'
     newish690 posted a new article 'Favorite letter?'
     newish690 posted a new article 'What's your favorite "3 Stooges" stooge?'
     newish690 posted a new article 'Is money good?'
     newish690 posted a new article 'Is Communism good or bad?'
   Get articles commented on by user
     the-huffington-post posted a new comment: 'XXX' about article: '...'
     yae33333 posted a new comment: 'XXX' about article: 'Reallllllllllllllllllllllllllllllllllllllllllllllllly long poll post'
     newish690 posted a new comment: 'XXX' about article: 'China disses the US'
     al-jazeera-english posted a new comment: 'XXX' about article: 'UK: Police officer suspended after kneeling on Black man's neck'
     newish690 posted a new comment: 'XXX' about article: 'China Slams U.S. Response to Hong Kong Security Law as 'Gangster Logic''
   Get articles voted on by user, and set their bucket accordingly.
     Article 'What are some of your favorite comedy movies?' now has a ranking of 1
     Article 'Rock, paper, or scissors?' now has a ranking of 1
     Article 'Is money good?' now has a ranking of 1
     Article 'What's your favorite "3 Stooges" stooge?' now has a ranking of 1
     Article 'What is your favorite color?' now has a ranking of 1
     Article 'In which order should we explore the Solar System?' now has a ranking of 1

   Example FB notifications:
     Currently relevant to vz:
   * ___ commented on [your/___'s] post.
   * ___ likes your comment: "___"
   * ___ posted a link ___.
   * ___ posted in ___.
   * ___ replied to [your/___'s] comment on your link.
   * ___, ___, and X people reacted to [your video: "___"/a link you shared]
	 Not relevant yet to vz:
   * ___ (group) has new posts.
   * ___ and ___ mentioned you in their comments.
   * ___ invited you to ___.

   Example Reddit messages:
   * post/comment reply: [link]
     from /u/___ via /r/___ sent ___ ago
     [The message]
*/
type CommentResult struct {
	Comment string
	Id		int64
}

func fetchRecentComments(notUserId int64, numComments int, withinElapsedMilliseconds int) (articles []Article, comments []CommentResult) {

	elapsedSeconds := withinElapsedMilliseconds / 1000
	withinTimeInterval := "now() - " + int_to_str(elapsedSeconds) + " * (interval '1 second')"

	rows := DbQuery(`
		SELECT
			c.Id,
			c.Text,
			p.UserId,
			u.Username,
			c.PostId,
			c.Created,
			p.Title
		FROM $$Comment c
		JOIN $$User u ON c.UserId = u.Id
		JOIN $$Post p ON c.PostId = p.Id
		WHERE c.UserId <> $1
		  AND c.Created > ` + withinTimeInterval + `
		ORDER BY c.Created DESC
		LIMIT $2;`,
		notUserId,
		numComments)

	for rows.Next() {
		var commentId	int64
		var comment		string
		var userId		int64
		var username 	string
		var postId		int64
		var created		time.Time
		var title		string
		check(rows.Scan(&commentId, &comment, &userId, &username, &postId, &created, &title))

		//prf("CommentId %d ; Comment %s ; userId %d ; postId %d ; created %#v ; title: %s\n",
		//	commentId, comment, userId, postId, created, title)

		articles = append(articles, Article {
			AuthorIconUrl: 	kDefaultAuthorIcon,  // TODO: we need real dinosaur icon art for users.
			Author: 		username,
			Id: 			postId,
			UserId:			userId,	// For the activity "your" nomenclature, this has to be the UserId of the article poster, not the commenter.
			PublishedAtUnix: created,
			TimeSince: 		getTimeSinceString(created, true),
			Title: 			title,
		})
		comments = append(comments, CommentResult {
			Comment: comment,
			Id:		 commentId,
		})
	}
	check(rows.Err())
	rows.Close()

	return
}

///////////////////////////////////////////////////////////////////////////////
//
// Fetch latest activity
//
///////////////////////////////////////////////////////////////////////////////
func fetchActivity(userId int64, withinElapsedMilliseconds int) (allArticles []Article, messages []string, links []string, unvisited []bool) {

	pr("Get articles shared by user")
	{
		articles := fetchArticlesNotPostedByUser(userId, 50, withinElapsedMilliseconds)

		prVal("  len(articles)", len(articles))

		for a, article := range articles {
			//prVal("article.UserId", article.UserId)
			messages = append(messages, fmt.Sprintf(
				"  shared %s %s: '%s'",
				ternary_str(article.UserId == userId,
					"your",
					ternary_str(article.IsPoll, "a", "an")),
				ternary_str(article.IsPoll, "poll", "article"),
				article.Title))

			links = append(links, fmt.Sprintf("/article/?postId=%d", article.Id))

			articles[a].TimeSince = getTimeSinceString(article.PublishedAtUnix, true)
		}

		allArticles = append(allArticles, articles...)
	}

	pr("Get articles commented on by user")
	{
		articles, comments := fetchRecentComments(userId, 50, withinElapsedMilliseconds)

		for a, article := range articles {
			//prVal("article.UserId", article.UserId)
			comment := comments[a]
			messages = append(messages, fmt.Sprintf(
				"  commented on %s %s '%s': '%s'",
				ternary_str(article.UserId == userId, "your", "the"),
				ternary_str(article.IsPoll, "poll", "article"),
				article.Title,
				ellipsify(comment.Comment, 42)))

			links = append(links, fmt.Sprintf("/article/?postId=%d#comment_%d", article.Id, comment.Id))
		}

		allArticles = append(allArticles, articles...)
	}
	// TODO: add replied to your comment
	//       add when poll gets more votes

	// Look up which links have been visited
	unvisited = make([]bool, len(allArticles))
	{
		// Copy this user's visited links for from database to a hash.
		visitedLinks := map[string]bool{}
		rows := DbQuery(`SELECT PathQuery FROM $$HasVisited WHERE UserId=$1::bigint`, userId)
		for	rows.Next() {
			pathQuery := ""
			rows.Scan(&pathQuery)

			visitedLinks[pathQuery] = true
		}
		check(rows.Err())
		rows.Close()

		//prVal("visitedLinks", visitedLinks)

		// Look up whether each link is in the hash, copy results to unvisited.
		for a, link := range links {
			// Strip off the hash for comparison, because the hash can never reach a server.
			cleanLink := strings.Split(link, "#")[0]

			//prVal("cleanLink", cleanLink)

			_, found := visitedLinks[cleanLink]
			unvisited[a] = !found

			//prf("  link %s %s found", cleanLink, ternary_str(found, "is", "is not"))
		}
	}

	return
}

///////////////////////////////////////////////////////////////////////////////
//
// Activity handler - notifications about your stuff that was replied to, or all content at the moment.
//
///////////////////////////////////////////////////////////////////////////////
func activityHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)

	pr("historyHandler")

	userId, username := GetSessionInfo(w, r)

	const oneMonth = 1000 * 60 * 60 * 24 * 30  // Calc milliseconds per month

	allArticles, messages, links, unvisited := fetchActivity(userId, oneMonth)

	// Create a list order, and sort the activities by date, indirectly, via the list order.
	assert(len(allArticles) == len(messages))
	prVal("len(allArticles)", len(allArticles))

	listOrder := make([]int, len(allArticles))
	for i := 0; i < len(listOrder); i++ {
		listOrder[i] = i
	}

	sort.Slice(listOrder, func(i, j int) bool {
		return allArticles[listOrder[i]].PublishedAtUnix.After(
			   allArticles[listOrder[j]].PublishedAtUnix)
	})

	prVal("links", links)


	// Render the news articles.
	args := struct {
		FrameArgs
		Messages	[]string
		Articles	[]Article
		ListOrder	[]int
		Links		[]string
		Unvisited	[]bool
	}{
		FrameArgs:	makeFrameArgs(r, "votezilla - Activity", "", kActivity, userId, username),
		Messages:	messages,
		Articles:	allArticles,
		ListOrder:	listOrder,
		Links:		links,
		Unvisited:	unvisited,
	}
	executeTemplate(w, kActivity, args)
}




/*POSSIBLE CODE TO USE:
removeDupIds := func(articles []Article) (filteredArticles []Article) {
	for _, article := range articles {

		// If duplicate id exists, purge the article.
		_, found := dupIds[article.Id]
		if !found {
			filteredArticles = append(filteredArticles, article)
			dupIds[article.Id] = true
		}
	}
	return
}*/
