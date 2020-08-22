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

func materializeNewsView() {
	pr("Materializing News Query")

	qp := defaultArticleQueryParams()
	qp.createMaterializedView 	= true
	qp.articlesPerCategory 		= kRowsPerCategory + 1
	qp.maxArticles		   		= kMaxArticles
	qp.addSemicolon				= true

	DbExec(qp.createArticleQuery())
}

func refreshNewsView() {
	viewExists := DbExists(
	   `SELECT relname, relkind
		FROM pg_class
		WHERE relname = '` + kMaterializedNewsView + `'
		AND relkind = 'm';`,
		)

	if viewExists {
		pr("AAAAAA")
		DbExec("REFRESH MATERIALIZED VIEW " + kMaterializedNewsView)
	} else {
		pr("BBBBBB")
		materializeNewsView()
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

	slot := 0

	for {
		slot = 0 // GET IT WORKING WITHOUT THE SLOT LOGIC FIRST!


		refreshNewsView()


		// Rotate the slot
		slot = (slot + 1) / 3

		pr("Sleeping 1 minute...")
		time.Sleep(1 * time.Minute)
	}
}