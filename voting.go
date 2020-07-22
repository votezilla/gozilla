package main

import (
	"encoding/json"
	"github.com/lib/pq"
	"net/http"
	"strconv"
)

type PollTallyResult struct {
	Count		int
	Percentage	float32
}

type PollTallyResults []PollTallyResult

type VoteData []string

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
		VoteData	VoteData
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

	pollId := int64(vote.PollId)

	// Send poll vote to the database, removing any prior vote.
	DbExec(
		`INSERT INTO $$PollVote(PollId, UserId, VoteOptionIds, VoteAmounts)
		 VALUES ($1::bigint, $2::bigint, $3::int[], $4::int[])
		 ON CONFLICT (PollId, UserId) DO UPDATE
		 SET (VoteOptionIds, VoteAmounts) = ($3::int[], $4::int[])`,
		pollId,
		userId,
		pq.Array(voteOptionIds),
		pq.Array(voteAmounts))

	// Tally the poll tally results, and cache them in the db.

	article, err := fetchArticle(pollId, userId)
	check(err)

	pollTallyResults := calcPollTally(pollId, article.PollOptionData)
	prVal("pollTallyResults", pollTallyResults)

	pollTallyResultsJson, err := json.Marshal(pollTallyResults)
   	check(err)
   	prVal("pollTallyResultsJson", pollTallyResultsJson)

	DbExec(
		`UPDATE $$PollPost
		 SET PollTallyResults = $1
		 WHERE Id = $2::bigint`,
		pollTallyResultsJson,
		pollId)

    // create json response from struct
    a, err := json.Marshal(vote)
    //checkw(err)
    if err != nil {
        //http.Error(w, err.Error(), http.StatusInternalServerError)
        serveError(w, err.Error())
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
	prf("calcPollTally %d %v", pollId, pollOptionData)

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

		prVal(">>1 pollTallyResults", pollTallyResults)

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

		prVal("dividend", dividend)

		invDividendPercent := ternary_float32(dividend != 0, 100.0 / float32(dividend), 0.0) // calc scalar dividend, prevent div by zero.
		for i := range pollTallyResults {
			pollTallyResults[i].Percentage = float32(pollTallyResults[i].Count) * invDividendPercent
		}

		prVal(">>2 pollTallyResults", pollTallyResults)

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
			for option, pollTallyResult := range pollTallyResults {
				if contains_int64(eliminatedVoteOptions, int64(option)) {
					continue
				}

				prf("\tOption %d\tCount %d\tPercentage %f", option, pollTallyResult.Count, pollTallyResult.Percentage)
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


//////////////////////////////////////////////////////////////////////////////
//
// view poll results II - non-popup
//
//////////////////////////////////////////////////////////////////////////////
// TODO: This is just duplicate code, make it view the results.  (Same or different handler for adding the vote?)
func viewPollResultsHandler(w http.ResponseWriter, r *http.Request) {
	RefreshSession(w, r)

	pr("viewPollResultsHandler")

	prVal("r.URL.Query()", r.URL.Query())

	reqPostId := parseUrlParam(r, "postId")

	postId, err := strconv.ParseInt(reqPostId, 10, 64) // Convert from string to int64.
	if err != nil {
		pr("error 1")
		http.Error(w, err.Error(), http.StatusInternalServerError) // TODO: prettify error displaying - use dinosaurs.
		return
	}

	userId, username := GetSessionInfo(w, r)
	article, err := fetchArticle(postId, userId)
	if err != nil {
		pr("error 2")
		http.Error(w, err.Error(), http.StatusInternalServerError) // TODO: prettify error displaying - use dinosaurs.
		return
	}


	// Tally the votes
	pollTallyResults := calcPollTally(postId, article.PollOptionData)

	// Suggested polls for further voting - on the sidebar.
	polls := fetchSuggestedPolls(userId, postId)

	upvotes, downvotes := deduceVotingArrows(append(polls, article))

	headComment, upcommentvotes, downcommentvotes := ReadCommentsFromDB(article.Id, userId)

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
		HeadComment					Comment
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
		PollTallyResults:			pollTallyResults,
		HeadComment:				headComment,
		MoreArticlesFromThisSource: polls,
	}

	executeTemplate(w, kViewPollResults, viewPollArgs)
}