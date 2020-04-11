package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

//////////////////////////////////////////////////////////////////////////////
//
// ajax poll vote
//
//////////////////////////////////////////////////////////////////////////////
func ajaxPollVoteHandler(w http.ResponseWriter, r *http.Request) {
	pr(vo_, "ajaxPollVoteHandler")
	prVal(vo_, "r.Method", r.Method)
	
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
    
    //parse request to struct
    var vote struct {
		PollId		int
		UserId		int
		VoteData	[]string
	}
	
    err := json.NewDecoder(r.Body).Decode(&vote)
    if err != nil {
		prVal(vo_, "Failed to decode json body", r.Body)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    prVal(vo_, "=======>>>>> vote", vote)
	
	/*
	if vote.Add {
    	DbExec( // sprintf necessary cause ::bool produces incorrect value in driver.
			`INSERT INTO $$PostVote(PostId, UserId, Up)
			 VALUES ($1::bigint, $2::bigint, $3::bool)
			 ON CONFLICT (PostId, UserId) DO UPDATE 
			 SET Up = $3::bool;`,
			vote.PostId,
			vote.UserId,
			vote.Up)
	} else { // remove
		DbExec(
			`DELETE FROM $$PostVote 
			 WHERE PostId = $1::bigint AND UserId = $2::bigint;`,
			vote.PostId,
			vote.UserId)
	}*/
    
    // create json response from struct
    a, err := json.Marshal(vote)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Write(a)
}

//////////////////////////////////////////////////////////////////////////////
//
// view poll results
//
//////////////////////////////////////////////////////////////////////////////
// TODO: This is just duplicate code, make it view the results.  (Same or different handler for adding the vote?)
func viewPollResultsHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)
	  
	prVal(nw_, "r.URL.Query()", r.URL.Query())

	reqPostId := parseUrlParam(r, "postId")
	postId, err := strconv.ParseInt(reqPostId, 10, 64)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // TODO: prettify error displaying - use dinosaurs.
		return
	}
	
	// Get the username.
	userId := GetSession(r)
	username := getUsername(userId)	
	
	// TODO_REFACTOR: unify articles and posts in database.
	article, err := fetchArticle(postId, userId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError) // TODO: prettify error displaying - use dinosaurs.
		return
	}
	
	upvotes, downvotes := deduceVotingArrows([]Article{article})
	
	comments := "TODO: NESTED COMMENTS!"
	
	// Render the news articles.
	commentsArgs := struct {
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
		PageArgs:		PageArgs{Title: "View Poll Results"},
		Username:		username,
		UserId:			userId,
		NavMenu:		navMenu,
		UrlPath:		"news",
		Article:		article,
		UpVotes:		upvotes, 
		DownVotes:		downvotes,
		Comments:		comments,
	}
	
	executeTemplate(w, "viewPollResults", commentsArgs)	
}