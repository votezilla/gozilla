package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/lib/pq"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type PollTallyResult struct {
	Count		int
	Percentage	float32
	Skip		bool		// Option has already been eliminated
	Worst		bool		// Option is currently being eliminated
}

type PollTallyResults []PollTallyResult

type PollTallyInfo struct {
	Stats		PollTallyResults
	TotalVotes	int
	Header		string
	Footer		string

	Article		*Article  		// So "PollTallyResults" can read in Article values.
	GetArticle	func() Article
}

type RawRankedVotes map[string]int

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

	userId := GetSession(w, r)
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
	rows.Close()

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
	pollTallyResults , _, _ := calcPollTally(pollId, pollOptionData, false, false, false, "", Article{})

	//prVal("pollTallyResults", pollTallyResults)

	pollTallyResultsJson, err := json.Marshal(pollTallyResults)
   	check(err)
   	//prVal("pollTallyResultsJson", pollTallyResultsJson)

   	InvalidateCache(userId)

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
func calcSimpleVoting(pollId int64, numOptions int, viewDemographics, viewRankedVoteRunoff, viewPollComparison, canSelectMultipleOptions bool, condition string) PollTallyResults {
	pollTallyResults := make(PollTallyResults, numOptions)

	// Get the votes from the database.
	rows := DbQuery(
		"SELECT v.VoteOptionIds FROM $$PollVote v" +
		ternary_str(viewDemographics, " JOIN $$User u ON v.UserId = u.Id ", "") +
		ternary_str(viewPollComparison, " JOIN $$PollVote v2 ON v.UserId = v2.UserId ", "") +
		" WHERE v.PollId = $1::bigint" + condition, pollId)
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
	TallyMultipleOptionsDifferently := false // TODO: revisit this later.
	if canSelectMultipleOptions && TallyMultipleOptionsDifferently {
		for i := range pollTallyResults {
			sum = max_int(sum, pollTallyResults[i].Count)
		}
	} else {
		for i := range pollTallyResults {
			sum += pollTallyResults[i].Count
		}
	}
	dividend = sum

	invDividendPercent := ternary_float32(dividend != 0, 100.0 / float32(dividend), 0.0) // calc scalar dividend, prevent div by zero.
	for i := range pollTallyResults {
		pollTallyResults[i].Percentage = float32(pollTallyResults[i].Count) * invDividendPercent
	}

	return pollTallyResults
}

func calcRankedChoiceVoting(pollId int64, numOptions int, viewDemographics, viewRankedVoteRunoff, viewPollComparison bool,
							condition string, article Article) (PollTallyResults, ExtraTallyInfo, RawRankedVotes) {
	pr("calcRankedChoiceVoting")

	pollTallyResults   := make(PollTallyResults, numOptions)
	extraTallyInfo	   := make(ExtraTallyInfo, 0)
	mostVotesForOption := make(PollTallyResults, numOptions)

	type UserRankedVotes struct {
		VoteOptions	[]int64
		VoteRanks	[]int64

		BestOption	int64
	}
	userRankedVotes := make([]UserRankedVotes, 0)

	// Get the votes from the database.
	rows := DbQuery("SELECT v.VoteOptionIds, v.VoteAmounts FROM $$PollVote v " +
		ternary_str(viewDemographics, " JOIN $$User u ON v.UserId = u.Id ", "") +
		ternary_str(viewPollComparison, " JOIN $$PollVote v2 ON v.UserId = v2.UserId ", "") +
		" WHERE v.PollId = $1::bigint" + condition, pollId)
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



	// TODO: sort, return, and display userRankedVotes - # of each ranking.


	//prVal("  userRankedVotes", userRankedVotes)

	rawRankedVotes := make(map[string]int)
/*	if len(article.PollOptionData.Options) > 0 {
		pr("==================================================================================")
		pr("  Calculating raw ranked votes:")
		prVal("  article.PollOptionData.Options", article.PollOptionData.Options)
		for u, userRankedVote := range userRankedVotes {
			prVal("  u", u)
			prVal("  userRankedVote", userRankedVote)

			// Turn ranked vote into a string description
			rankedVoteDescription := ""
			for r := int64(1); r < int64(10); r++ {
				prVal("  r", r)
				prVal("  userRankedVote.VoteRanks", userRankedVote.VoteRanks)

				found := false
				for voteIdx, rank := range userRankedVote.VoteRanks {
					if rank == r {
						rankedVoteDescription = rankedVoteDescription + "#" + int64_to_str(r) + "=" + article.PollOptionData.Options[voteIdx] + ", "
						found = true
						break
					}
				}

				if !found {
					break
				}
			}
			rankedVoteDescription = rankedVoteDescription[:len(rankedVoteDescription)-2]  // Remove last ", "
			prVal("rankedVoteDescription", rankedVoteDescription)

			rawRankedVotes[rankedVoteDescription]++
		}
	}
	pr("  >>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>>. ")
	pr("  rawRankedVotes:")
	for k, v := range rawRankedVotes {
		prf("  %40s -> %d", k, v)
	}
*/
	// Do the ranked voting algorithm.
	eliminatedVoteOptions := make([]int64, 0)
	round := 1
	done := false

	rankedVotingLoop: for {
		message := ""

		// For each user...
		for u, userRankedVote := range userRankedVotes {

			// ...Find the best option for the user...
			userRankedVotes[u].BestOption = -1
			minRank	  					 := MaxInt64
			for r, rank := range userRankedVote.VoteRanks {
				option := userRankedVote.VoteOptions[r]

				// ...(Skip eliminated options).
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
		for _, userRankedVote := range userRankedVotes {
			// If the user's best option is valid (i.e. all their candidates weren't eliminated), add it to the tally.
			if userRankedVote.BestOption >= 0 {
				pollTallyResults[userRankedVote.BestOption].Count++
				sum++
			}
		}

		prVal("sum", sum)

		// Calculate the percentage.
		if sum > 0 {
			invDividendPercent := 100.0 / float32(sum)
			for i := range pollTallyResults {
				pollTallyResults[i].Percentage = float32(pollTallyResults[i].Count) * invDividendPercent
			}
		} else {
			for i := range pollTallyResults {
				pollTallyResults[i].Percentage = 0.0
			}
		}

		// Accumulate the most votes for each option (based on all the ranked vote runoff rounds).
		for i, pollTallyResult := range pollTallyResults {
			mostVotesForOption[i].Count = max_int(mostVotesForOption[i].Count, pollTallyResult.Count)
		}

		prf("Round %d results:", round)
		for option, pollTallyResult := range pollTallyResults {
			if contains_int64(eliminatedVoteOptions, int64(option)) {
				continue
			}

			prf("\tOption %d\tCount %d\tPercentage %f", option, pollTallyResult.Count, pollTallyResult.Percentage)
		}

		// Once a vote option has the majority, we have found a winner.  (We need this check, so we know when we're done.)
		for i := range pollTallyResults {
			if pollTallyResults[i].Percentage > 50.0 /*&& !viewRankedVoteRunoff*/ {
				if viewRankedVoteRunoff {
					message += "Found a winner, with a majority of the vote!: '" + article.PollOptionData.Options[i] + "'"

					pr(message)
				}
				done = true
			}
		}

		// Otherwise, eliminate the remaining vote option with the fewest votes and recount the votes.
		var worstOptions []int64

		if !done {
			leastVotes  := MaxInt
			mostVotes   := 0
			for option, pollTallyResult := range pollTallyResults {
				// It must be from one of the options remaining.
				if contains_int64(eliminatedVoteOptions, int64(option)) {
					continue
				}

				if pollTallyResult.Count < leastVotes {
					leastVotes = pollTallyResult.Count
				}
				if pollTallyResult.Count > mostVotes {
					mostVotes = pollTallyResult.Count
				}
			}
			// Eliminate the worst options... without deleting anything.
			if leastVotes < mostVotes {
				for option, pollTallyResult := range pollTallyResults {
					// It must be from one of the options remaining.
					if contains_int64(eliminatedVoteOptions, int64(option)) {
						continue
					}

					if pollTallyResult.Count == leastVotes {
						worstOptions = append(worstOptions, int64(option))
					}
				}

				if viewRankedVoteRunoff && len(worstOptions) >= 1 {
					message += "Eliminated " + ternary_str(len(worstOptions) > 1, "options", "option") + ": "

					for _, worstOption := range worstOptions  {
						message = message + "'" + article.PollOptionData.Options[worstOption] + "', "
					}
					message = message[:len(message)-2] // strip the last ", "

					message = message + ", as " + ternary_str(len(worstOptions) > 1, "they", "it") + " had the lowest vote."

					pr(message)
				}

				// Stop when we have one candidate remaining.
				if round == numOptions - 1 {
					//assert(len(eliminatedVoteOptions) == numOptions - 1)
					if viewRankedVoteRunoff {
						numOptionsRemaining := len(pollTallyResults) - len(eliminatedVoteOptions) - len(worstOptions)
						message += fmt.Sprintf("We'll stop now since we only have %d candidates remaining.", numOptionsRemaining)
					}
					done = true
				}
			} else {
				assert(leastVotes == mostVotes)

				message += fmt.Sprintf("We'll stop now since it's a tie.")

				done = true
			}
		}

		pr("************************************************************************")
		if viewRankedVoteRunoff {
			var pollTallyInfo PollTallyInfo
			pollTallyInfo.Stats = make(PollTallyResults, len(pollTallyResults))

			copy(pollTallyInfo.Stats, pollTallyResults)  // Copy pollTallyResults into Stats.

			pollTallyInfo.Header = "Ranked Vote Runoff - Pass " + int_to_str(round)
			pollTallyInfo.Footer = message

			for option, _ := range pollTallyInfo.Stats {
				if contains_int64(worstOptions, int64(option)) {
					pollTallyInfo.Stats[option].Worst = true

					prVal("  WORST OPTION", option)
				}
			}

			for option, _ := range pollTallyInfo.Stats {
				if contains_int64(eliminatedVoteOptions, int64(option)) {
					pollTallyInfo.Stats[option].Skip = true

					prVal("  SKIPPING OPTION", option)
				}
			}

			pr("Check 2")
			for option, _ := range pollTallyInfo.Stats {
				if pollTallyInfo.Stats[option].Skip {
					prVal("  SKIPPING OPTION", option)
				}
			}

			extraTallyInfo = append(extraTallyInfo, pollTallyInfo)

			prVal("  Appending pollTallyInfo", pollTallyInfo)
			pr("    Now extraTallyInfo =")
			for x, _ := range extraTallyInfo {
				prf("      extraTallyInfo[%d]=%#v", x, extraTallyInfo[x])
			}
		}

		eliminatedVoteOptions = append(eliminatedVoteOptions, worstOptions...)

		if done {
			break rankedVotingLoop
		}

		round++
	}

	if flags.debug != "" {
		pr("Check 3")
		for x, _ := range extraTallyInfo {
			for option, _ := range extraTallyInfo[x].Stats {
				if extraTallyInfo[x].Stats[option].Skip {
					prVal("  SKIPPING OPTION", option)
				} else {
					prVal("  Count is ", extraTallyInfo[x].Stats[option].Count)
				}
			}
		}
	}

	sum := 0
	for i := range mostVotesForOption {
		sum += mostVotesForOption[i].Count
	}
	if sum > 0 {
		invDividendPercent := 100.0 / float32(sum)
		for i := range mostVotesForOption {
			mostVotesForOption[i].Percentage = float32(mostVotesForOption[i].Count) * invDividendPercent
		}
	} // otherwise, should default to 0.0

	return mostVotesForOption, extraTallyInfo, rawRankedVotes  // We return mostVotesForOption as pollTallyResults, because that summarizes the votes the best.  Winners still win, 3rd party votes are displayed.
}

// Trim zeros from pollTallyResults.
func trimZeroStats(pollTallyResults PollTallyResults) {
	prVal("<< trimZeroStats pollTallyResults", pollTallyResults)
	for i, _ := range pollTallyResults {
		// Trim zero, unless it's one of the first 2 options or we're showing an option being eliminated during a rankedVoteRunoff.
		if i > 1 && pollTallyResults[i].Count == 0 && !pollTallyResults[i].Worst {
			pollTallyResults[i].Skip = true
		}
	}
	prVal(">> trimZeroStats pollTallyResults", pollTallyResults)
}

// Calcs the poll tally.  If it's a ranked vote with viewRankedVoteRunoff, return the extraTallyInfo as well.
func calcPollTally(pollId int64, pollOptionData PollOptionData, viewDemographics, viewRankedVoteRunoff, viewPollComprison bool, condition string, article Article) (PollTallyResults, ExtraTallyInfo, RawRankedVotes) {
	prf("calcPollTally %d %v", pollId, pollOptionData)

	numOptions := len(pollOptionData.Options)

	pollTallyResults := make(PollTallyResults, numOptions)
	extraTallyInfo := make(ExtraTallyInfo, 0)
	rawRankedVotes := make(RawRankedVotes, 0)

	if (!pollOptionData.RankedChoiceVoting) { // Regular single or multi-select voting
		pollTallyResults = calcSimpleVoting(pollId, numOptions, viewDemographics, viewRankedVoteRunoff, viewPollComprison, pollOptionData.CanSelectMultipleOptions, condition)

		assert(len(pollOptionData.Options) == len(pollTallyResults))

	} else { // RankedChoiceVoting
		pollTallyResults, extraTallyInfo, rawRankedVotes = calcRankedChoiceVoting(pollId, numOptions, viewDemographics, viewRankedVoteRunoff, viewPollComprison, condition, article)
	}

	trimZeroStats(pollTallyResults)
	for i, _ := range extraTallyInfo {
		trimZeroStats(extraTallyInfo[i].Stats)
	}

	return pollTallyResults, extraTallyInfo, rawRankedVotes
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
	compareToPoll		:= parseUrlParam(r, "compareToPoll")
	viewPollComparison	:= compareToPoll != ""

	prVal("reqPostId", reqPostId)
	prVal("viewDemographics", viewDemographics)
	prVal("viewRankedVoteRunoff", viewRankedVoteRunoff)
	prVal("splitByDemographic", splitByDemographic)
	prVal("compareToPoll", compareToPoll)
	prVal("viewPollComparison", viewPollComparison)

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

	pr("XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")

	// Tally the vote stats
	var extraTallyInfo ExtraTallyInfo
	var rawRankedVotes RawRankedVotes
	article.PollTallyInfo.Stats, extraTallyInfo, rawRankedVotes =
		calcPollTally(postId, article.PollOptionData, false, viewRankedVoteRunoff, false, "", article)

	article.PollTallyInfo.TotalVotes = 0
	for i := 0; i < len(article.PollTallyInfo.Stats); i++ {
		article.PollTallyInfo.TotalVotes += article.PollTallyInfo.Stats[i].Count
	}
	prVal("article.PollTallyInfo.Stats", article.PollTallyInfo.Stats)
	prVal("article.PollTallyInfo.TotalVotes", article.PollTallyInfo.TotalVotes)

	article.PollTallyInfo.SetArticle(&article)

	pollIdB := int64(-1)
	articleB := Article{}

	// Tally the demographic vote stats
	prVal("XXX viewDemographics", viewDemographics)
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
			var condition string

			if splitByDemographic == "age" {
				var minAge, maxAge int
				otherCase := false

				prVal("option[0]", option[0])

				// Min and max age ranges - TODO: update data in ageRanges in demographics.go.
				switch option[0][0] {
					case '0': minAge =  0; maxAge = 17; break
					case '1': minAge = 18; maxAge = 33; break
					case '2': minAge = 34; maxAge = 49; break
					case '3': minAge = 50; maxAge = 65; break
					case '4': minAge = 66; maxAge = 99999; break
					default: otherCase = true
				}

				prVal("  minAge", minAge)
				prVal("  maxAge", maxAge)
				prVal("  otherCase", otherCase)

				if otherCase {
					condition = " AND u.BirthYear IS NULL "
				} else {
					currentYear := time.Now().Year()
					minYear := currentYear - maxAge
					maxYear := currentYear - minAge

					condition = " AND (" + int_to_str(minYear) + " <= u.BirthYear AND u.BirthYear <= " + int_to_str(maxYear) + ") "
				}
			} else if splitByDemographic == "country" && option[0] == "US" {
				condition = " AND u.Country = 'US' "
			} else if splitByDemographic == "country" && option[0] == "OUTSIDE" {
				condition = " AND (u.Country NOT IN ('US', 'SKIP', 'OTHER') AND u.Country IS NOT NULL) "
			} else if option[0] == "SKIP" {
				condition = " AND (u." + column + " = '" + option[0] + "' OR u." + column + " IS NULL)"
			} else {
				condition = " AND u." + column + " = '" + option[0] + "' "
			}
			extraTallyInfo[o].Header = option[1]
			extraTallyInfo[o].Stats, _, _ = calcPollTally(postId, article.PollOptionData, viewDemographics, false, false, condition, article)
		}
	} else if viewPollComparison {
		if compareToPoll != "" {
			pollIdB, err = strconv.ParseInt(compareToPoll, 10, 64) // Convert from string to int64.
			if err != nil {
				pr("error B")
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		articleB, err = fetchArticle(pollIdB, userId)
		if err != nil {
			pr("error C")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		prVal("articleB", articleB)

		// Fetch the pollOptionData for poll B.
		var pollOptionDataB PollOptionData
		DoQuery(
			func(rows *sql.Rows) {
				var pollOptionJson	string
				err := rows.Scan(&pollOptionJson)
				check(err)

				assert(len(pollOptionJson) > 0)
				check(json.Unmarshal([]byte(pollOptionJson), &pollOptionDataB))
			},
			"SELECT PollOptionData FROM $$PollPost WHERE Id = $1::bigint",
				pollIdB,
		)
		prVal("  pollOptionDataB", pollOptionDataB)
		prVal("  PollOptionDataB.Options", pollOptionDataB.Options)

		extraTallyInfo = make([]PollTallyInfo, len(pollOptionDataB.Options))
		for i, optionB := range pollOptionDataB.Options {
			extraTallyInfo[i].Header = "Those who voted '" + optionB + "' also voted:"
			extraTallyInfo[i].Stats, _, _ = calcPollTally(
				postId, article.PollOptionData,
				false, false, true,
				" AND v2.PollId = " + int64_to_str(pollIdB) + " AND " + int_to_str(i) + " = ANY(v2.VoteOptionIds)",
				article,
			)
		}
	}

	// Trim demographics / poll comparision options that have no votes.
	for i := len(extraTallyInfo) - 1; i >= 0; i-- {
		totalCount := 0

		for j := 0; j < len(extraTallyInfo[i].Stats); j++ {
			totalCount += extraTallyInfo[i].Stats[j].Count
		}
		if totalCount == 0 {
			extraTallyInfo = append(extraTallyInfo[:i], extraTallyInfo[i+1:]...)
		}
	}

	for i := 0; i < len(extraTallyInfo); i++ {
		extraTallyInfo[i].SetArticle(&article)
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

	// Make comparable polls labels:
	comparablePollId := ternary_int64(pollIdB != -1, pollIdB, postId)  // Because of the row-col switching, this fixes what to compare to next.

	comparablePolls := make(map[int64]string, 0)
	DoQuery(
		func(rows *sql.Rows) {
			var pollIdB		int64
			var pollTitleB	string
			var count		int

			err := rows.Scan(&pollIdB, &pollTitleB, &count)
			check(err)

			comparablePolls[pollIdB] = pollTitleB + " (" + int_to_str(count) + ")"
		},
		"SELECT PollIdB, TitleB, Count FROM " + kLinkedPollsView + " WHERE PollidA = $1 ORDER BY Count DESC;",
			comparablePollId,
	)
	prVal("comparablePolls", comparablePolls)

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
		FocusOnTopComment			bool

		DemographicLabels			map[string]string
		ComparablePolls				map[int64]string
		ViewDemographics			bool
		ViewRankedVoteRunoff		bool
		ViewPollComparison			bool
		ExtraTallyInfo				ExtraTallyInfo
		PollIdB						int64
		ArticleB					Article
		ComparablePollId			int64
		RawRankedVotes				RawRankedVotes
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
		CommentPrompt:				"Start a discussion, or explain why you voted for " + userVoteString + ".",
		FocusOnTopComment:			!(viewDemographics || viewRankedVoteRunoff || viewPollComparison),

		DemographicLabels:			demographicLabels,
		ComparablePolls:			comparablePolls,
		ViewDemographics:			viewDemographics,
		ViewRankedVoteRunoff:		viewRankedVoteRunoff,
		ViewPollComparison:			viewPollComparison,
		ExtraTallyInfo:				extraTallyInfo,
		PollIdB:					pollIdB,
		ArticleB:					articleB,
		ComparablePollId:			comparablePollId,
		RawRankedVotes:				rawRankedVotes,
	}

	executeTemplate(w, kViewPollResults, viewPollArgs)
}
