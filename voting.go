package main

import (
	"encoding/json"
//	"fmt"
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
	pr("ajaxPollVoteHandler")
	prVal("r.Method", r.Method)

	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	userId := GetSession(r)
	if userId == -1 { // Secure cookie not found.  Either session expired, or someone is hacking.
		// So go to the register page.
		pr("Must be logged in to vote.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	prVal("userId", userId)

    //parse request to struct
    var vote struct {
		PollId		int
		VoteData	[]string
	}

    err := json.NewDecoder(r.Body).Decode(&vote)
    if err != nil {
		prVal("Failed to decode json body", r.Body)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    prVal("=======>>>>> vote", vote)

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
		pq.Array(voteAmounts))

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
	numOptions := len(pollOptionData.Options)

	pollTallyResults := make(PollTallyResults, numOptions)

	if (!pollOptionData.RankedChoiceVoting) { // Regular single or multi-select voting

		// Get the votes from the database.
		rows := DbQuery("SELECT VoteOptionIds FROM $$PollVote WHERE PollId = $1::bigint", pollId)
		defer rows.Close()
		for rows.Next() {
			var voteOptions []int64	// This is the only type possible for scanning into an array of ints.

			err := rows.Scan(pq.Array(&voteOptions))
			check(err)

			// Tally the votes.
			for _, voteOption := range voteOptions {
				pollTallyResults[voteOption].Count++
			}
		}
		check(rows.Err())

		dividend := 0
		if !pollOptionData.CanSelectMultipleOptions { // Single select - basic survey - get the sum.
			sum := 0
			for i := range pollTallyResults {
				sum += pollTallyResults[i].Count
			}
			dividend = sum
		} else {                                      // Multi-select survey - get the greatest value.
			greatest := 0
			for i := range pollTallyResults {
				greatest = max_int(greatest, pollTallyResults[i].Count)
			}
			dividend = greatest
		}

		invDividendPercent := 100.0 / float32(dividend)
		for i := range pollTallyResults {
			pollTallyResults[i].Percentage = float32(pollTallyResults[i].Count) * invDividendPercent
		}

		return pollTallyResults
	} else { // RankedChoiceVoting

		type UserRankedVotes struct {
			VoteOptions	[]int64
			VoteRanks	[]int64

			BestOption	int64
		}
		userRankedVotes := make([]UserRankedVotes, 0)

		// Get the votes from the database.
		rows := DbQuery("SELECT VoteOptionIds, VoteAmounts FROM $$PollVote WHERE PollId = $1::bigint", pollId)
		defer rows.Close()
		for rows.Next() {
			var voteOptions, voteRanks []int64	// []int64 is the only type possible for scanning into an array of ints.

			err := rows.Scan(pq.Array(&voteOptions), pq.Array(&voteRanks))
			check(err)

			assert(len(voteOptions) == len(voteRanks))

			userRankedVotes = append(userRankedVotes,
								     UserRankedVotes {
										 VoteOptions:	voteOptions,
										 VoteRanks:		voteRanks })
		}
		check(rows.Err())

		// Do the ranked voting algorithm.
		eliminatedVoteOptions := make([]int64, 0)
		round := 1
		rankedVotingLoop: for {
			// For each user...
			for u, userRankedVote := range(userRankedVotes) {

				// ...Find the best option for the user...
				userRankedVotes[u].BestOption = -1
				minRank	  					 := MaxInt64
				for r, rank := range(userRankedVote.VoteRanks) {
					option := userRankedVote.VoteOptions[r]

					// ...Based on the candidates still available.
					if contains_int64(eliminatedVoteOptions, option) {
						continue
					}

					if rank < minRank { // The best option has the lowest rank (closest to "1").
						minRank 	   				  = rank
						userRankedVotes[u].BestOption = option
					}
				}
			}

			// Clear the tally results
			for i := range pollTallyResults {
				pollTallyResults[i].Count = 0
			}
			sum := 0

			// Tally the votes.
			for _, userRankedVote := range(userRankedVotes) {
				pollTallyResults[userRankedVote.BestOption].Count++
				sum++
			}

			prVal("sum", sum)

			// Calculate the percentage.
			invDividendPercent := 100.0 / float32(sum)
			for i := range pollTallyResults {
				pollTallyResults[i].Percentage = float32(pollTallyResults[i].Count) * invDividendPercent
			}

			prf("Round %d results:", round)
			for i, pollTallyResult := range pollTallyResults {
				prf("\tOption %d\tCount %d\tPercentage %f", i, pollTallyResult.Count, pollTallyResult.Percentage)
			}

			// Once a vote option has the majority, we have found a winner.  (Should we skip this?  Yes, I think!  Just a dumb hand-counting optimization to save time.)
			//for i := range pollTallyResults {
			//	if pollTallyResults[i].Percentage > 50.0 {
			//		break rankedVotingLoop
			//	}
			//}

			// Otherwise, eliminate the remaining vote option with the fewest votes and recount the votes.
			leastVotes  := MaxInt
			worstOption := -1
			for option, pollTallyResult := range pollTallyResults {
				// It must be from one of the options remaining.
				if contains_int64(eliminatedVoteOptions, int64(option)) {
					continue
				}

				if pollTallyResult.Count < leastVotes {
					leastVotes = pollTallyResult.Count
					worstOption = option
				}
			}
			eliminatedVoteOptions = append(eliminatedVoteOptions, int64(worstOption))

			prf("Eliminated vote option %d, it had the lowest vote count: %d", worstOption, leastVotes)

			// Stop when we have one candidate remaining.
			if round == numOptions - 1 {
				assert(len(eliminatedVoteOptions) == numOptions - 1)

				break rankedVotingLoop
			}

			round++
		}

		return pollTallyResults
	}

	return pollTallyResults
}


func testPopupHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)

	pr("testPopupHandler")

	// Get the username.
	userId := GetSession(r)
	username := getUsername(userId)

	// Render the news articles.
	testPopupArgs := struct {
		PageArgs
		Username			string
		UserId				int64
		NavMenu				[]string
		UrlPath				string
		UpVotes				[]int64
		DownVotes			[]int64
	}{
		PageArgs:			PageArgs{Title: "Test popup"},
		Username:			username,
		UserId:				userId,
		NavMenu:			navMenu,
		UrlPath:			"testPopup",
		UpVotes:			[]int64{},
		DownVotes:			[]int64{},
	}

	executeTemplate(w, kTestPopup, testPopupArgs)
}

//////////////////////////////////////////////////////////////////////////////
//
// view poll results II - non-popup
//
//////////////////////////////////////////////////////////////////////////////
// TODO: This is just duplicate code, make it view the results.  (Same or different handler for adding the vote?)
func viewPollResultsHandler2(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)

	pr("viewPollResultsHandler2")

	prVal("r.URL.Query()", r.URL.Query())

	reqVoteData := parseUrlParam(r, "voteData")
	prVal("reqVoteData", reqVoteData)

	decodedVoteData, err := url.QueryUnescape(reqVoteData)
	check(err)
	prVal("decodedVoteData", decodedVoteData)

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

	// Tally the votes
	pollTallyResults := calcPollTally(postId, article.PollOptionData)

	userVoteString := "" // userVoteString is a textual representation the user's vote(s)."
	for i, option := range(article.PollOptionData.Options) {
		userVoteString += ternary_str(voteData[i] != "",  // if the vote was checked:
			ternary_str(userVoteString != "", ", ", "") + //   concat with ", "
				option,                                   //   all votes that were checked
				"")
	}

	polls := fetchPolls(userId, postId)

	upvotes, downvotes := deduceVotingArrows(append(polls, article))

	comments, upcommentvotes, downcommentvotes := ReadCommentTagsFromDB(article.Id, userId)

	prVal("upvotes", upvotes)
	prVal("downvotes", downvotes)
	prVal("upcommentvotes", upcommentvotes)
	prVal("downcommentvotes", downcommentvotes)

	// Render the news articles.
	viewPollArgs := struct {
		PageArgs
		Username					string
		UserId						int64
		NavMenu						[]string
		UrlPath						string
		Article						Article
		UpVotes						[]int64
		DownVotes					[]int64
		UpCommentVotes				[]int64
		DownCommentVotes 			[]int64
		VoteData					[]string
		UserVoteString				string
		PollTallyResults			PollTallyResults
		Comments					[]CommentTag
		MoreArticlesFromThisSource	[] Article
	}{
		PageArgs:					PageArgs{Title: "View Poll Results"},
		Username:					username,
		UserId:						userId,
		NavMenu:					navMenu,
		UrlPath:					"news",
		Article:					article,
		UpVotes:					upvotes,
		DownVotes:					downvotes,
		UpCommentVotes:				upcommentvotes,
		DownCommentVotes: 			downcommentvotes,
		VoteData:					voteData,	// The way this user just voted.
		UserVoteString:				userVoteString,
		PollTallyResults:			pollTallyResults,
		Comments:					comments,
		MoreArticlesFromThisSource: polls,
	}

	executeTemplate(w, kViewPollResults2, viewPollArgs)
}