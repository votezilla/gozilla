// Database cache service
//
// Currently just materializing the expensive (and non-user) part of the posts / articles query, to hopefully
// speed up the main query by 500ms.  Just an optimization for /news.
//
// To view materialized views: SELECT relname, relkind FROM pg_class WHERE relkind = 'm';
package main

import (
//	"encoding/json"
//	"fmt"
//	"io/ioutil"
//	"strings"
	"time"
)

const (
	kLinkedPollsView = "linkedpollsview"
)

func materializeNewsView(newsCycle int) {
	pr("Materializing News Query")

	qp := defaultArticleQueryParams()
	qp.createMaterializedView 	= true
	qp.articlesPerCategory 		= kRowsPerCategory + 1
	qp.maxArticles		   		= kMaxArticles
	qp.addSemicolon				= true
	qp.newsCycle				= newsCycle

	DbExec(qp.createArticleQuery())
}

func viewExists(viewName string) bool {
	return DbExists(
	   `SELECT relname, relkind
		FROM pg_class
		WHERE relname = '` + viewName + `'
		AND relkind = 'm';`,
		)
}

func refreshNewsView(newsCycle int) {
	viewExists := viewExists(kMaterializedNewsView + int_to_str(newsCycle))

	if viewExists {
		query := "REFRESH MATERIALIZED VIEW " + kMaterializedNewsView + int_to_str(newsCycle)
		pr(query)

		DbExec(query)
	} else {
		prVal("Materialize news cycle", newsCycle)

		materializeNewsView(newsCycle)
	}
}

//////////////////////////////////////////////////////////////////////////////
func refreshLinkedPolls() {
	viewExists := viewExists(kLinkedPollsView)

	if viewExists {
		DbExec("REFRESH MATERIALIZED VIEW " + kLinkedPollsView)
	} else {
		DbExec(
		   `CREATE MATERIALIZED VIEW ` + kLinkedPollsView + ` AS
				SELECT
					a.pollid AS PollIdA,
					b.pollid AS PollIdB,
					c.title AS TitleA,
					d.title AS TitleB,
					COUNT(a.userid)
				FROM $$PollVote a
				JOIN $$PollVote b ON a.userid = b.userid and a.pollid <> b.pollid
				JOIN $$PollPost c ON a.pollid = c.id
				JOIN $$PollPost d ON b.pollid = d.id
				GROUP BY 1, 2, 3, 4
				HAVING COUNT(a.userid) >= 5
				ORDER BY 1, 2;`)
	}
}

//////////////////////////////////////////////////////////////////////////////
//
// caching service - On startup, and every 1 minute, materializes the expensive
//                (and non-user) part of the posts / articles query, and stores
//                it in one of 3 rotating slots, so users can have randomness
//                when reading /news.
//
//////////////////////////////////////////////////////////////////////////////
func CachingService() {
	pr("========================================")
	pr("====== STARTING DB CACHE SERVICE =======")
	pr("========================================\n")

	newsCycle := 0

	numRepetitions := 0
	for {
		refreshNewsView(newsCycle)  // Trying this to fix the hanging issue we get sometimes.

		// Rotate the slot
		newsCycle = (newsCycle + 1) % 3

		DbTrackOpenConnections()

		if numRepetitions >= 3 {
			refreshLinkedPolls()

			pr("Sleeping 1 minute...")
			time.Sleep(1 * time.Minute)
		}

		numRepetitions++
	}
}
