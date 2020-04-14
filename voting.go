package main

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"net/url"
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
	
	userId := GetSession(r);
	if userId == -1 { // Secure cookie not found.  Either session expired, or someone is hacking.
		// So go to the register page.
		pr(go_, "Must be logged in to vote.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	prVal(vo_, "userId", userId);
    
    //parse request to struct
    var vote struct {
		PollId		int
		VoteData	[]string
	}
	
    err := json.NewDecoder(r.Body).Decode(&vote)
    if err != nil {
		prVal(vo_, "Failed to decode json body", r.Body)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    prVal(vo_, "=======>>>>> vote", vote)
    
    // TODO: there is vote data validation on the client, but it may need to be added
    //       on the server eventually.
    
    voteDataJson, err := json.Marshal(vote.VoteData);
    if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
    }
    prVal(vo_, "voteDataJson", voteDataJson)
       
    // Send poll vote to the database.
	DbExec( 
		`INSERT INTO $$PollVote(PollId, UserId, Vote)
		 VALUES ($1::bigint, $2::bigint, $3)
		 ON CONFLICT (PollId, UserId) DO UPDATE
		 SET Vote = $3;`,
		vote.PollId,
		userId,
		voteDataJson);			
    
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
	
	reqVoteData := parseUrlParam(r, "voteData")
	prVal(vo_, "reqVoteData", reqVoteData)
	
	decodedVoteData, err := url.QueryUnescape(reqVoteData)
	check(err)
	prVal(vo_, "decodedVoteData", decodedVoteData)
	
	voteData := strings.Split(decodedVoteData, ",")

	reqPostId := parseUrlParam(r, "postId")
	
	postId, err := strconv.ParseInt(reqPostId, 10, 64) // Convert from string to int64.
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
	
	userVoteString := "" // userVoteString is a textual representation the user's vote(s)."
	for i, option := range(article.PollOptionData.Options) {
		userVoteString += ternary_str(voteData[i] != "",  // if the vote was checked:
			ternary_str(userVoteString != "", ", ", "") + //   concat with ", "
				option,                                   //   all votes that were checked
				"")
	}
	
	comments := "TODO: NESTED COMMENTS!"
	
	// Render the news articles.
	viewPollArgs := struct {
		PageArgs
		Username		string
		UserId			int64
		NavMenu			[]string
		UrlPath			string
		Article			Article
		UpVotes			[]int64
		DownVotes		[]int64
		Comments		string
		VoteData		[]string
		UserVoteString	string
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
		VoteData:		voteData,	// The way this user just voted.
		UserVoteString:	userVoteString, 
	}
	
	executeTemplate(w, "viewPollResults", viewPollArgs)	
}