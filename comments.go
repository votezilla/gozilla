package main

import (
	"net/http"
	"strconv"
)

/*
CREATE TABLE vz.Comment (
    Id                  BIGSERIAL   PRIMARY KEY,

    PostId              BIGINT NOT NULL, -- Which post this is a comment on.
    UserId              BIGINT NOT NULL, -- The user who left this comment.
    ParentId            BIGINT,          -- The parent comment, hierarchially.

    Text                VARCHAR(40000) NOT NULL, -- 40,000 is Reddit's maximum text length.
    PrevRevisions       VARCHAR(40000)[],

    -- Materialized path - this sorts it into a tree hierarchy for us, though
    --                     we'll still need to sort by aye/nay vote later...
    --                     think if this is the right algorithm.
    MaterializedPath    BIGINT NOT NULL, -- int64, bits used as follows:
                                         --    16b, 16b, 8b, 8b, 4b, 4b, 4b, 4b
                                         --     L0   L1  L2  L3  L4  L5  L6, L7
    -- Note: for now we'll fetch all comments for a post,
    --       then SORT BY MaterializedPath.
    --       in Go, we'll parse these into a hierarchy, will need pointers.
    --              Then sort each list by aye/nay vote.
    --       Eventually... we'll need to optimize this.

    Created             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    Deleted             BOOL DEFAULT false
);
CREATE UNIQUE INDEX comment_path_index ON vz.Comment (Id, MaterializedPath);  -- UNIQUE index is a btree, so sorting by MaterializedPath should be fast.
*/


//////////////////////////////////////////////////////////////////////////////
//
// display comments
//
//////////////////////////////////////////////////////////////////////////////
func articleHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)

	prVal(co_, "r.URL.Query()", r.URL.Query())

	reqPostId := parseUrlParam(r, "postId")
	postId, err := strconv.ParseInt(reqPostId, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // TODO: prettify error displaying - use dinosaurs.
		return
	}

	// Get the username.
	userId := GetSession(r)
	username := getUsername(userId)

	prVal(co_, "userId", userId);
	prVal(co_, "username", username);

	// TODO_REFACTOR: unify articles and posts in database.
	article, err := fetchArticle(postId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // TODO: prettify error displaying - use dinosaurs.
		return
	}

	upvotes, downvotes := deduceVotingArrows([]Article{article})

	comments := "TODO: NESTED COMMENTS!"

	// Render the news articles.
	articleArgs := struct {
		PageArgs
		Username		string
		UserId			int64
		NavMenu			[]string
		UrlPath			string
		Article			Article
		UpVotes			[]int64
		DownVotes		[]int64
		Comments		string
	}{
		PageArgs:		PageArgs{Title: "votezilla - Article"},
		Username:		username,
		UserId:			userId,
		NavMenu:		navMenu,
		UrlPath:		"news",
		Article:		article,
		UpVotes:		upvotes,
		DownVotes:		downvotes,
		Comments:		comments,
	}

	executeTemplate(w, "article", articleArgs)
}
