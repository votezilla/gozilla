package main

import (
	"fmt"
	"net/http"
	"sort"
)

// POSSIBLE ACTIVITY OUTPUT:
// Get polls voted on by user
//   Poll 'Favorite letter?' now has X votes
//   Poll 'What are some of your favorite comedy movies?' now has X votes
//   Poll 'What's your favorite "3 Stooges" stooge?' now has X votes
//   Poll 'Rock, paper, or scissors?' now has X votes
//   Poll '2 + 2 = __________' now has X votes
//   Poll 'Rank or file?' now has X votes
// Get articles posted by user
//   newish690 posted a new article 'Rock, paper, or scissors?'
//   newish690 posted a new article 'What is your favorite color?'
//   newish690 posted a new article 'Favorite letter?'
//   newish690 posted a new article 'What's your favorite "3 Stooges" stooge?'
//   newish690 posted a new article 'Is money good?'
//   newish690 posted a new article 'Is Communism good or bad?'
// Get articles commented on by user
//   the-huffington-post posted a new comment: 'XXX' about article: '...'
//   yae33333 posted a new comment: 'XXX' about article: 'Reallllllllllllllllllllllllllllllllllllllllllllllllly long poll post'
//   newish690 posted a new comment: 'XXX' about article: 'China disses the US'
//   al-jazeera-english posted a new comment: 'XXX' about article: 'UK: Police officer suspended after kneeling on Black man's neck'
//   newish690 posted a new comment: 'XXX' about article: 'China Slams U.S. Response to Hong Kong Security Law as 'Gangster Logic''
// Get articles voted on by user, and set their bucket accordingly.
//   Article 'What are some of your favorite comedy movies?' now has a ranking of 1
//   Article 'Rock, paper, or scissors?' now has a ranking of 1
//   Article 'Is money good?' now has a ranking of 1
//   Article 'What's your favorite "3 Stooges" stooge?' now has a ranking of 1
//   Article 'What is your favorite color?' now has a ranking of 1
//   Article 'In which order should we explore the Solar System?' now has a ranking of 1



///////////////////////////////////////////////////////////////////////////////
//
// Activity handler - notifications about your stuff that was replied to, or all content at the moment.
//
///////////////////////////////////////////////////////////////////////////////
func activityHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)

	pr("historyHandler")

	userId, username := GetSessionInfo(w, r)

	allArticles := []Article{}
	messages := []string{}

	addMessage := func(message string) { messages = append(messages, message) }

	//removeDupIds := func(articles []Article) (filteredArticles []Article) {
	//	for _, article := range articles {
//
	//		// If duplicate id exists, purge the article.
	//		_, found := dupIds[article.Id]
	//		if !found {
	//			filteredArticles = append(filteredArticles, article)
	//			dupIds[article.Id] = true
	//		}
	//	}
	//	return
	//}

	pr("Get polls voted on by user")
	{
		articles := fetchPollsVotedOnByUser(userId, kNumCols * kRowsPerCategory)
		//articles = removeDupIds(articles)

		// TODO: peel this out to activitiyHandler.
		for _, article := range articles {
			addMessage(fmt.Sprintf("  [Your] Poll '%s' now has X votes", article.Title))
		}

		allArticles = append(allArticles, articles...)

		for a := range articles {
			articles[a].Bucket = kVotedPolls
		}
	}

	pr("Get articles posted by user")
	{
		articles := fetchArticlesPostedByUser(userId, kNumCols * kRowsPerCategory)
		//articles = removeDupIds(articles)

		for _, article := range articles {
			addMessage(fmt.Sprintf("  %s posted a new article '%s'", article.Author, article.Title))
		}

		allArticles = append(allArticles, articles...)

		for a := range articles {
			articles[a].Bucket = kSubmittedPosts
		}
	}

	pr("Get articles commented on by user")
	{
		articles := fetchArticlesCommentedOnByUser(userId, kNumCols * kRowsPerCategory)
		//articles = removeDupIds(articles)

		for _, article := range articles {
			addMessage(fmt.Sprintf("  %s posted a new article '%s'", article.Author, article.Title))
		}

		allArticles = append(allArticles, articles...)

		for a := range articles {
			articles[a].Bucket = kCommentedPosts
		}
	}

	pr("Get articles voted on by user, and set their bucket accordingly.")
	{
		articles := fetchArticlesUpDownVotedOnByUser(userId, kNumCols * kRowsPerCategory)
		//articles = removeDupIds(articles)

		for _, article := range articles {
			addMessage(fmt.Sprintf("  Your Article '%s' now has a ranking of %d", article.Title, article.VoteTally)) // TODO: add names of users who upvoted.
		}

		allArticles = append(allArticles, articles...)

		for a := range articles {
			articles[a].Bucket = kVotedPosts
		}
	}

	upvotes, downvotes := deduceVotingArrows(allArticles)

	prVal("upvotes", upvotes)
	prVal("downvotes", downvotes)

	// Create a list order, and sort the activities by date, indirectly, via the list order.
	assert(len(allArticles) == len(messages))
	prVal("len(allArticles)", len(allArticles))

	var listOrder []int
	for j := 8; j < len(allArticles); j++ {
		prf("J = %d", j)

		listOrder = make([]int, j)//len(allArticles))  // <<<< IT WORKS UP TO 8, BUT AT 16 IT FAILS!!!
		for i := 0; i < len(listOrder); i++ {
			listOrder[i] = i
		}
//		prVal("<< SORT", listOrder)
//		pr("CREATED:")
//		for i := 0; i < len(listOrder); i++ {
//			prf("  %2d: %#v", i, allArticles[listOrder[i]].PublishedAt)
//		}

		// vvv
		sort.Slice(listOrder, func(i, j int) bool {
			return allArticles[listOrder[i]].PublishedAtUnix.After(
				   allArticles[listOrder[j]].PublishedAtUnix)
		})

/*		prVal(">> SORT", listOrder)
		pr("CREATED:")
		for i := 0; i < len(listOrder); i++ {
			prf("  %2d: %#v", i, allArticles[listOrder[i]].PublishedAt)
		}
		for i := 0; i < len(listOrder) - 1; i++ {
			assertMsg(allArticles[listOrder[i]].PublishedAtUnix.After(
					  allArticles[listOrder[i+1]].PublishedAtUnix) ||
					  allArticles[listOrder[i]].PublishedAtUnix.Equal(
					  allArticles[listOrder[i+1]].PublishedAtUnix),
				fmt.Sprintf("%d is before %d", i, i+1))
		}
*/
	}

	// Render the news articles.
	args := struct {
		FrameArgs
		Messages	[]string
		Articles	[]Article
		ListOrder	[]int
	}{
		FrameArgs:	makeFrameArgs2("votezilla - Activity", "", kActivity, userId, username, upvotes, downvotes),
		Messages:	messages,
		Articles:	allArticles,
		ListOrder:	listOrder,
	}
	executeTemplate(w, kActivity, args)
}
