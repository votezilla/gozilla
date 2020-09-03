package main

import (
	"encoding/json"
	"github.com/lib/pq"
	"net/http"
	"strconv"
	"strings"
)

type PollTallyResult struct {
	Count		int
	Percentage	float32
}

type PollTallyResults []PollTallyResult

type PollTallyInfo struct {
	Stats		PollTallyResults

	Article		*Article  		// So "PollTallyResults" can read in Article values.
	GetArticle	func() Article

	Header		string
}

func (i *PollTallyInfo) SetArticle(pArticle *Article) {
	(*i).Article = pArticle
	(*i).GetArticle = func() Article {
		return *i.Article
	}
}

type ExtraTallyInfo	[]PollTallyInfo

type VoteData []string


//////////////////////////////////////////////////////////////////////////////
//
// ajax poll vote
//
//////////////////////////////////////////////////////////////////////////////
func ajaxPollVote(w http.ResponseWriter, r *http.Request) {
	pr("ajaxPollVote")
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
		PollId				int
		VoteData			[]string
		NewVoteData			[]string
		NewOptions			[]string
		NumOriginalOptions	int
	}

    err := json.NewDecoder(r.Body).Decode(&vote)
    if err != nil {
		prVal("Failed to decode json body", r.Body)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    prVal("=======>>>>> vote", vote)

    // Convert voteDataJson into parallel arrays for more compact database storage.
	// 1) Voting for existing options:
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
	// 2) Voting for new options the user just added, while remembering the new options added.
	newOptions := make([]string, 0)        // New options the user just added.
	newOptionId := vote.NumOriginalOptions // Index of the next option the user might vote on.
	for o, newOption := range vote.NewOptions {
		if newOption != "" { // Filter out empty options (they were probably mistakes), along with corresponding votes for them.
			newOptions = append(newOptions, newOption) // Add new user-submitted option.

			str := vote.NewVoteData[o]
			if str != "" {
				voteOptionIds = append(voteOptionIds, newOptionId)

				if str != "x" { // Ranked Voting
					voteAmount, err := strconv.Atoi(str)
					check(err)

					voteAmounts = append(voteAmounts, voteAmount)
				}
			}

			newOptionId++
		}
	}

	pollId := int64(vote.PollId)

	// Fetch the pollOptionData for the poll.
	var pollOptionData PollOptionData
	rows := DbQuery("SELECT PollOptionData FROM $$PollPost WHERE Id = $1::bigint", pollId)
	for rows.Next() {
		var pollOptionJson	string
		err := rows.Scan(&pollOptionJson)
		check(err)

		assert(len(pollOptionJson) > 0)
		check(json.Unmarshal([]byte(pollOptionJson), &pollOptionData))

		// If the user has added new options, add them to pollOptionData and the database.
		if len(newOptions) > 0 {
			pollOptionData.Options = append(pollOptionData.Options, newOptions...)

			a, err := json.Marshal(pollOptionData)
			check(err)

			// TODO: Fetching & updating PollOptionsData should be protected by a db transaction.
			DbExec("UPDATE $$PollPost SET PollOptionData = $1 WHERE Id = $2::bigint", a, pollId)
		}
	}
	check(rows.Err())

	// Send poll vote to the database, removing any prior vote.
	// TODO: make the code and database protect against duplicate names.
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
	pollTallyResults := calcPollTally(pollId, pollOptionData, false, false, "")

	//prVal("pollTallyResults", pollTallyResults)

	pollTallyResultsJson, err := json.Marshal(pollTallyResults)
   	check(err)
   	//prVal("pollTallyResultsJson", pollTallyResultsJson)

	DbExec(
		`UPDATE $$PollPost
		 SET PollTallyResults = $1
		 WHERE Id = $2::bigint`,
		pollTallyResultsJson,
		pollId)

    // create json response from struct
    a, err := json.Marshal(vote)

    if err != nil {
        serveError(w, err)
        return
    }
    w.Write(a)
}

//////////////////////////////////////////////////////////////////////////////
//
// calc poll tally
//
//////////////////////////////////////////////////////////////////////////////
func calcSimpleVoting(pollId int64, numOptions int, viewDemographics, viewRankedVoteRunoff bool, condition string) PollTallyResults {
	pollTallyResults := make(PollTallyResults, numOptions)

	// Get the votes from the database.
	joinStr := ternary_str(viewDemographics, " JOIN $$User u ON v.UserId = u.Id ", "")

	rows := DbQuery("SELECT v.VoteOptionIds FROM $$PollVote v" + joinStr + " WHERE PollId = $1::bigint" + condition, pollId)
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
	sum := 0
	for i := range pollTallyResults {
		sum += pollTallyResults[i].Count
	}
	dividend = sum

	invDividendPercent := ternary_float32(dividend != 0, 100.0 / float32(dividend), 0.0) // calc scalar dividend, prevent div by zero.
	for i := range pollTallyResults {
		pollTallyResults[i].Percentage = float32(pollTallyResults[i].Count) * invDividendPercent
	}

	return pollTallyResults
}

func calcRankedChoiceVoting(pollId int64, numOptions int, viewDemographics, viewRankedVoteRunoff bool,
							condition string) PollTallyResults {
	pollTallyResults := make(PollTallyResults, numOptions)

	type UserRankedVotes struct {
		VoteOptions	[]int64
		VoteRanks	[]int64

		BestOption	int64
	}
	userRankedVotes := make([]UserRankedVotes, 0)

	// Get the votes from the database.
	joinStr := ternary_str(viewDemographics, " JOIN $$User u ON v.UserId = u.Id ", "")

	rows := DbQuery("SELECT v.VoteOptionIds, v.VoteAmounts FROM $$PollVote v " + joinStr + " WHERE PollId = $1::bigint" + condition, pollId)
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
			// If the user's best option is valid (i.e. all their candidates weren't eliminated), add it to the tally.
			if userRankedVote.BestOption >= 0 {
				pollTallyResults[userRankedVote.BestOption].Count++
				sum++
			}
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
		for i := range pollTallyResults {
			if pollTallyResults[i].Percentage > 50.0 {
				break rankedVotingLoop
			}
		}

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
		// Eliminate the worst option... without deleting anything :)
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

func calcPollTally(pollId int64, pollOptionData PollOptionData, viewDemographics, viewRankedVoteRunoff bool, condition string) PollTallyResults {
	prf("calcPollTally %d %v", pollId, pollOptionData)

	numOptions := len(pollOptionData.Options)

	pollTallyResults := make(PollTallyResults, numOptions)

	if (!pollOptionData.RankedChoiceVoting) { // Regular single or multi-select voting
		pollTallyResults = calcSimpleVoting(pollId, numOptions, viewDemographics, viewRankedVoteRunoff, condition)

		assert(len(pollOptionData.Options) == len(pollTallyResults))

	} else { // RankedChoiceVoting
		pollTallyResults = calcRankedChoiceVoting(pollId, numOptions, viewDemographics, viewRankedVoteRunoff, condition)
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

	pr("======================================================================")
	pr("viewPollResultsHandler")
	pr("======================================================================")

	prVal("r.URL.Query()", r.URL.Query())

	reqPostId 			:= parseUrlParam(r, "postId")
	splitByDemographic	:= parseUrlParam(r, "splitByDemographic")
	viewDemographics	:= splitByDemographic != ""
	viewRankedVoteRunoff:= str_to_bool(parseUrlParam(r, "viewRankedVoteRunoff"))

	prVal("reqPostId", reqPostId)
	prVal("viewDemographics", viewDemographics)
	prVal("viewRankedVoteRunoff", viewRankedVoteRunoff)
	prVal("splitByDemographic", splitByDemographic)

	postId, err := strconv.ParseInt(reqPostId, 10, 64) // Convert from string to int64.
	if err != nil {
		pr("error 1")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	userId, username := GetSessionInfo(w, r)
	article, err := fetchArticle(postId, userId)
	if err != nil {
		pr("error 2")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}


	// Tally the vote stats
	if !viewDemographics && !viewRankedVoteRunoff {
		article.PollTallyInfo.Stats = calcPollTally(postId, article.PollOptionData, false, false, "")
	}
	article.PollTallyInfo.SetArticle(&article)

	// Tally the demographic vote stats
	var extraTallyInfo ExtraTallyInfo
	if viewDemographics {
		pr("viewDemographics")

		demoOptions, found := demographicOptions[splitByDemographic]
		if !found {
			http.Error(w, "Invalid demographic", http.StatusInternalServerError)
		}

		demoOptions = append(demoOptions, [2]string{"OTHER", "OTHER"}, [2]string{"SKIP", "SKIP"})

		column := demographicColumns[splitByDemographic]

		extraTallyInfo = make([]PollTallyInfo, len(demoOptions))
		for o, option := range demoOptions {
			extraTallyInfo[o].Stats = calcPollTally(postId, article.PollOptionData, viewDemographics, false,
				ternary_str(option[0] == "SKIP",
					" AND (u." + column + " = '" + option[0] + "' OR u." + column + " IS NULL)",
					" AND u." + column + " = '" + option[0] + "' "))
			extraTallyInfo[o].Header = option[1]
		}

		// Trim demographics that have no votes.
		prVal("extraTallyInfo", extraTallyInfo)
		prVal("len(extraTallyInfo)", len(extraTallyInfo))
		for i := len(extraTallyInfo) - 1; i >= 0; i-- {
			prVal("  i", i)
			totalCount := 0

			prVal("  extraTallyInfo[i].Stats", extraTallyInfo[i].Stats)
			prVal("  len(extraTallyInfo[i].Stats)", len(extraTallyInfo[i].Stats))
			for j := 0; j < len(extraTallyInfo[i].Stats); j++ {
				prVal("    j", j)
				totalCount += extraTallyInfo[i].Stats[j].Count
			}
			if totalCount == 0 {
				extraTallyInfo = append(extraTallyInfo[:i], extraTallyInfo[i+1:]...)
			}
		}

		for i := 0; i < len(extraTallyInfo); i++ {
			extraTallyInfo[i].SetArticle(&article)
		}
	}

	assert(len(article.PollOptionData.Options) == len(article.PollTallyInfo.Stats))

	prVal("article.PollTallyInfo", article.PollTallyInfo)
	prVal("extraTallyInfo", extraTallyInfo)

	// Deduce userVoteString.
	//{{ range $o, $option := $poll.Options }}
	//	{{ if (index $article.VoteData $o) }}
	userVoteString := ""
	{
		userVotedOptions := []string{}
		for o, option := range article.PollOptionData.Options {
			if o < len(article.VoteData) && article.VoteData[o] {
				userVotedOptions = append(userVotedOptions, `"` + option + `"`)
			}
		}
		userVoteString = strings.Join(userVotedOptions, ", ")
	}

	// Suggested polls for further voting - on the sidebar.
	polls := fetchSuggestedPolls(userId, postId)

	upvotes, downvotes := deduceVotingArrows(append(polls, article))

	headComment, upcommentvotes, downcommentvotes := ReadCommentsFromDB(article.Id, userId)

	// Render the news articles.

	pa := makePageArgs(r, "View Poll Results", "", article.Description)

	viewPollArgs := struct {
		PageArgs
		Username					string
		UserId						int64
		NavMenu						[]string
		UrlPath						string
		UpVotes						[]int64
		DownVotes					[]int64

		Article						Article
		UpCommentVotes				[]int64
		DownCommentVotes 			[]int64
		UserVoteString				string
		HeadComment					Comment
		MoreArticlesFromThisSource	[]Article
		CommentPrompt				string
		DemographicLabels			map[string]string
		ViewDemographics			bool
		ViewRankedVoteRunoff		bool
		ExtraTallyInfo				ExtraTallyInfo
	}{
		PageArgs:					pa,
		Username:					username,
		UserId:						userId,
		NavMenu:					navMenu,
		UrlPath:					"news",
		Article:					article,
		UpVotes:					upvotes,
		DownVotes:					downvotes,

		UpCommentVotes:				upcommentvotes,
		DownCommentVotes: 			downcommentvotes,
		HeadComment:				headComment,
		MoreArticlesFromThisSource: polls,
		UserVoteString:				userVoteString,
		CommentPrompt:				"Explain why you vote for " + userVoteString,
		DemographicLabels:			demographicLabels,
		ViewDemographics:			viewDemographics,
		ViewRankedVoteRunoff:		viewRankedVoteRunoff,
		ExtraTallyInfo:				extraTallyInfo,
	}

	executeTemplate(w, kViewPollResults, viewPollArgs)
}