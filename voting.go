package main

import (
	"encoding/json"
	"github.com/lib/pq"
	"net/http"
	"net/url"
	"strconv"
	"strings"	
)

type PollTallyResult struct {
	Count		int
	Percentage	float32
}

type PollTallyResults []PollTallyResult

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
 /*   
    voteDataJson, err := json.Marshal(vote.VoteData);
    if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
    }
    prVal(vo_, "voteDataJson", voteDataJson) 
 */   
    // Convert voteDataJson into parallel arrays for more compact database storage.
    voteOptionIds := make([]int, 0)
    voteAmounts := make([]int, 0)
    
    for optionId, str := range vote.VoteData {
		if str != "" {
			voteOptionIds = append(voteOptionIds, optionId)
			
			if str != "x" { // Ranked Voting
				voteAmount, err := strconv.Atoi(str)
				check(err)
				
				voteAmounts = append(voteAmounts, voteAmount)
			}
		}
	}
   
    // Send poll vote to the database, removing any prior vote.
	DbExec( 
		`INSERT INTO $$PollVote(PollId, UserId, VoteOptionIds, VoteAmounts)
		 VALUES ($1::bigint, $2::bigint, $3::int[], $4::int[])
		 ON CONFLICT (PollId, UserId) DO UPDATE
		 SET (VoteOptionIds, VoteAmounts) = ($3::int[], $4::int[])`,
		vote.PollId,
		userId,
		pq.Array(voteOptionIds),
		pq.Array(voteAmounts));			
    
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
// calc poll tally
//
//////////////////////////////////////////////////////////////////////////////
func calcPollTally(pollId int64, pollOptionData PollOptionData) PollTallyResults {	
	pollTallyResults := make(PollTallyResults, len(pollOptionData.Options))
	
	if (!pollOptionData.RankedChoiceVoting) { // Regular single or multi-select voting
		rows := DbQuery("SELECT VoteOptionIds FROM $$PollVote WHERE PollId = $1::bigint", pollId)
	
		defer rows.Close()
		for rows.Next() {
			var voteOptionIds []int64	// This is the only type possible for scanning into an array of ints.

			err := rows.Scan(pq.Array(&voteOptionIds))
			check(err)
			
			// Sum the votes.
			for _, voteOption := range voteOptionIds {
				pollTallyResults[voteOption].Count++
			}
			
			dividend := 0
			
			if !pollOptionData.CanSelectMultipleOptions { // Single select - basic survey.
				sum := 0
				for i := range pollTallyResults {
					sum += pollTallyResults[i].Count
				} 
				dividend = sum
			} else {                       // Multi-select survey.
				greatest := 0
				for i := range pollTallyResults {
					greatest = max_int(greatest, pollTallyResults[i].Count)
				}
				dividend = greatest
			}
			
			invDividend := float32(1.0 / dividend)
			for i := range pollTallyResults {
				pollTallyResults[i].Percentage = float32(100.0) * float32(pollTallyResults[i].Count) * invDividend
			}
		}
		check(rows.Err())	
		
		return pollTallyResults
	} else { // RankedChoiceVoting
		nyi()
		/* TODO: implement vote tallying for RankedChoiceVoting
		voteRankings := make([][]int, len(pollOptionData.Options))

		rows := DbQuery("SELECT VoteOptionIds, VoteAmounts FROM $$PollVote WHERE PollId = $1::bigint", pollId)
	
		defer rows.Close()
		for rows.Next() {
			var voteOptionIds, voteAmounts []int

			err := rows.Scan(pq.Array(&voteOptionIds), pq.Array(&voteAmounts))
			check(err)
			
			voteTally := make([]int, len(pollOptionData.Options))
			for {
				for v, rank := range voteAmounts {
					//if rank == "1"
				}
				
				nyi()
			
				// Clear voteTally back to 0.
				for i := range voteTally {
					voteTally[i] = 0
				}
			}
			
			//for _, voteOption := range voteOptionIds {
			//	pollTallyResults = append(pollTallyResults, voteOption)
			//}
			
			return pollTallyResults
		}
		check(rows.Err())	
		*/
	}
	
	return pollTallyResults
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
	
	// Tally the votes
	pollTallyResults := calcPollTally(postId, article.PollOptionData)
	
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
		Username			string
		UserId				int64
		NavMenu				[]string
		UrlPath				string
		Article				Article
		UpVotes				[]int64
		DownVotes			[]int64
		Comments			string
		VoteData			[]string
		UserVoteString		string
		PollTallyResults	PollTallyResults
	}{
		PageArgs:			PageArgs{Title: "View Poll Results"},
		Username:			username,
		UserId:				userId,
		NavMenu:			navMenu,
		UrlPath:			"news",
		Article:			article,
		UpVotes:			upvotes, 
		DownVotes:			downvotes,
		Comments:			comments,
		VoteData:			voteData,	// The way this user just voted.
		UserVoteString:		userVoteString, 
		PollTallyResults:	pollTallyResults,
	}
	
	executeTemplate(w, "viewPollResults", viewPollArgs)	
}