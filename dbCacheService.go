// Database cache service
//
// Currently just materializing the expensive (and non-user) part of the posts / articles query, to hopefully
// speed up the main query by 500ms.  Just an optimization for /news.
package main

import (
//	"encoding/json"
//	"fmt"
//	"io/ioutil"
//	"strings"
	"time"
)






//////////////////////////////////////////////////////////////////////////////
//
// news service - On startup, and every 1 minute, materializes the expensive
//                (and non-user) part of the posts / articles query, and stores
//                it in one of 3 rotating slots, so users can have randomness
//                when reading /news.
//
//////////////////////////////////////////////////////////////////////////////
func DbCacheService() {
	pr("========================================")
	pr("====== STARTING DB CACHE SERVICE =======")
	pr("========================================\n")

	slot := 0

	for {
		slot = 0 // GET IT WORKING WITHOUT THE SLOT LOGIC FIRST!

		//query :=

		// Rotate the slot
		slot = (slot + 1) / 3

		pr("Sleeping 1 minute...")
		time.Sleep(1 * time.Minute)
	}
}