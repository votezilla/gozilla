package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/lib/pq"
)

const (
	kMaxCommentLines = 6
	kCharsPerLine    = 60  // 60 for mobile.  80 would be desktop, but there is no way to detect the difference yet.

	tabs = "                                                                                                    "
)

// This is also a comment tree.
type Comment struct {
	Id           int64 // id == -1 if pointing to the Post. (So not a comment, but the children are all L0 comments.)
	Username     string
	Text         []string // an array of strings, separated by <br>.  Do it this way so the template can handle it.
	VoteTally    int
	Quality      int
	IsExpandible bool
	PostId       int64
	IsHead       bool

	Children []*Comment
	Parent   *Comment
}

// For representing a hierarchical tree of comments in a flattened list.
type CommentTag struct {
	Id        int64
	Username  string
	Text      []string // an array of strings, separated by <br>.  Do it this way so the template can handle it.
	VoteTally int

	IsHead          bool
	IsChildrenStart bool
	IsChildrenEnd   bool

	IsExpandible bool
}

//////////////////////////////////////////////////////////////////////////////
//
// ajax create comment
//
//////////////////////////////////////////////////////////////////////////////
func ajaxCreateComment(w http.ResponseWriter, r *http.Request) {
	pr("ajaxCreateComment")
	prVal("r.Method", r.Method)

	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	userId := GetSession(r)
	if userId == -1 { // Secure cookie not found.  Either session expired, or someone is hacking.
		// So go to the register page.
		pr("Must be logged in to create a comment.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	prVal("userId", userId)

	//parse request to struct
	var newComment struct {
		Id       int64
		PostId   int64
		ParentId int64
		Text     string
	}

	err := json.NewDecoder(r.Body).Decode(&newComment)
	if err != nil {
		prVal("Failed to decode json body", r.Body)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	prVal("=======>>>>> newComment", newComment)

	// Get the postId and path from the parent's info, in the database.
	newPath := []int64{} // New path = append(parent's path, num children).
	{
		// Have the database determine what the new path should be.
		// e.g	Parent path:	1, 2, 3
		//      Child0 path: 	1, 2, 3, 0
		//      Child1 path: 	1, 2, 3, 1
		//      New Child path: [1, 2, 3] + (NumChildren)
		rows := DbQuery("SELECT ARRAY_APPEND(Path, NumChildren) FROM $$Comment WHERE Id = $1::bigint", newComment.ParentId)
		defer rows.Close()
		if rows.Next() {
			arr := pq.Int64Array{} // This weirdness is required for scanning into []int64

			err := rows.Scan(&arr)
			check(err)

			newPath = []int64(arr) // This weirdness is required for scanning into []int64
		} else {
			// If it's not in the database, it must be because it has Id = -1 (the top-level post)...
			assert(newComment.ParentId == -1)

			// The head comment of the tree, must be added!
			// This allows us to maintain a count of top-level posts, in this head record's NumChildren.
			DbExec(`INSERT INTO $$Comment (Id, PostId, UserId, ParentId, Text, Path, NumChildren)
					VALUES (-1, $1::bigint, -1, -1, '', '{}'::bigint[], 0);`,
				newComment.PostId)
		}
		check(rows.Err())
	}

	// TODO: add a database transaction here.
	//       See: http://go-database-sql.org/prepared.html

	// Send the new comment to the database.
	newComment.Id = DbInsert(
		`INSERT INTO $$Comment (PostId, UserId, ParentId, Text, Path)
	     VALUES ($1::bigint, $2::bigint, $3::bigint, $4, $5::bigint[])
	     returning Id;`,
		newComment.PostId,
		userId,
		newComment.ParentId,
		newComment.Text,
		pq.Array(newPath))

	// Increment the parent's number of children.
	DbExec(`UPDATE $$Comment SET NumChildren = NumChildren + 1 WHERE Id = $1::bigint`, newComment.ParentId)

	// Increment the Post's NumComments field here.
	DbExec(`UPDATE $$Post SET NumComments = NumComments + 1 WHERE Id = $1::bigint`, newComment.PostId)

	// Have user like their own comments by default.
	voteUpDown(newComment.Id, userId, true, true, true)

	// Convert newlines to be HTML-friendly.  (Do it here so the JSON response gets it and also it will get reapplied
	// in ReadCommentTagsFromDB.)
	newComment.Text = strings.Replace(newComment.Text, "\n", "<br>", -1)

	// create json response from struct.  It needs to know newCommentId so it knows where to put the focus after the window reload.
	a, err := json.Marshal(newComment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(a)
}

//////////////////////////////////////////////////////////////////////////////
//
// ajax expand comment
//
//////////////////////////////////////////////////////////////////////////////
func ajaxExpandComment(w http.ResponseWriter, r *http.Request) {
	pr("ajaxCreateComment")
	prVal("r.Method", r.Method)

	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	userId := GetSession(r)
	if userId == -1 { // Secure cookie not found.  Either session expired, or someone is hacking.
		// So go to the register page.
		pr("Must be logged in to create a comment.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	prVal("userId", userId)

	//parse request to struct
	var expandComment struct {
		CommentId int64
	}

	err := json.NewDecoder(r.Body).Decode(&expandComment)
	if err != nil {
		prVal("Failed to decode json body", r.Body)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	prVal("=======>>>>> expandComment", expandComment)

	var expandedComment struct {
		Text string
	}
	{
		rows := DbQuery("SELECT Text FROM $$Comment WHERE Id = $1::bigint", expandComment.CommentId)
		defer rows.Close()
		if rows.Next() {
			err := rows.Scan(&expandedComment.Text)
			check(err)
		} else {
			assert(false)
		}
		check(rows.Err())
	}

	prVal("=======>>>>> expandedComment", expandedComment)

	// create json response from struct.  It needs to know newCommentId so it knows where to put the focus after the window reload.
	a, err := json.Marshal(expandedComment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(a)
}

//////////////////////////////////////////////////////////////////////////////
//
// Print comment tree, for debugging.
//
//////////////////////////////////////////////////////////////////////////////
func prComments(commentTree *Comment, depth int) {
	prf("%sComment #%d (%d): %s", tabs[0:depth*2], commentTree.Id, commentTree.Quality, commentTree.Text)

	for _, child := range commentTree.Children {
		prComments(child, depth+1)
	}
}

//////////////////////////////////////////////////////////////////////////////
//
// Calculate comment quality, based on upvotes and number of comments.
//
//////////////////////////////////////////////////////////////////////////////
func calcCommentQuality(commentTree *Comment) {
	commentsBy := map[string]bool{}

	for _, child := range commentTree.Children {
		calcCommentQuality(child)

		commentsBy[child.Username] = true
	}

	// 100 * #upvotes + 10 * num unique users leaving comments + 1 * num child comments.
	commentTree.Quality = 100*commentTree.VoteTally + 10*len(commentsBy) + len(commentTree.Children)
}

//////////////////////////////////////////////////////////////////////////////
//
// Sort the comments by quality, ascending order.
//
//////////////////////////////////////////////////////////////////////////////
func sortComments(commentTree *Comment) {
	for _, child := range commentTree.Children {
		sortComments(child)
	}

	sort.Slice(commentTree.Children[:], func(i, j int) bool {
		return commentTree.Children[i].Quality > commentTree.Children[j].Quality
	})
}

//////////////////////////////////////////////////////////////////////////////
//
// read comment tags from db
//
// This is the one we're using.
// Read comments from the database, then convert it into a flattened tag format that the template file can easily render.
//
// TODO: We'll eventually need to call ReadCommentsFromDB, so the children of each comment can be sorted by upvote.
//
//////////////////////////////////////////////////////////////////////////////
func ReadCommentsFromDB(postId, userId int64) (headComment Comment, upCommentVotes, downCommentVotes []int64) {
	headComment.Id = -1
	headComment.IsHead = true
	headComment.Children = make([]*Comment, 0)

	pPrevComment := &headComment

	prVal("pPrevComment.Children", pPrevComment.Children)


	prevPathDepth := int64(0)
	var pathLengthDiff int64

	pr("ReadCommentsFromDB:")

	// The simpler way for now:
	rows := DbQuery(
		`WITH comments AS (
			 SELECT c.Id AS Id,
					Text,
					COALESCE(u.Username, '') AS Username,
					COALESCE(array_length(Path, 1), 0) AS PathLength,
					Path
			 FROM $$Comment c
			 LEFT JOIN $$User u
			 ON c.UserId = u.Id
			 WHERE PostId = $1::bigint
			 ORDER BY Path
			),
			votes AS (
				SELECT CommentId,
					   SUM(CASE WHEN Up THEN 1 ELSE -1 END) AS VoteTally
				FROM $$CommentVote
				WHERE CommentId IN (SELECT Id FROM comments)
				GROUP BY CommentId
			)
		SELECT
			Id,
			Text,
			Username,
			PathLength,
			COALESCE(votes.VoteTally, 0) AS VoteTally,
			CASE WHEN v.Up IS NULL THEN 0
				 WHEN v.Up THEN 1
				 ELSE -1
				 END AS Upvoted
		FROM comments
		LEFT JOIN votes ON comments.Id = votes.CommentId
		LEFT JOIN $$CommentVote v ON comments.Id = v.CommentId AND (v.UserId = $2::bigint)
		ORDER BY comments.Path`,
		postId,
		userId)

	defer rows.Close()
	for rows.Next() {
		var newComment Comment
		newComment.Children = make([]*Comment, 0)

		var pathLen int64
		var commentText string
		var upvoted int

		err := rows.Scan(&newComment.Id, &commentText, &newComment.Username, &pathLen, &newComment.VoteTally, &upvoted)
		check(err)

		switch upvoted {
		case 1:
			upCommentVotes = append(upCommentVotes, newComment.Id)
			break
		case -1:
			downCommentVotes = append(downCommentVotes, newComment.Id)
			break
		}

		// Convert newlines to be HTML-friendly -
		//    split into lines so the template file can handle it,
		//    add elipsis if too long.
		newComment.Text = strings.Split(commentText, "\n")

		numLinesApprox := 0
		for i, textLine := range newComment.Text {
			if i > 0 {
				numLinesApprox++
			}

			linesLeft := kMaxCommentLines - numLinesApprox

			length := len(textLine)
			numLinesApprox += (length + 59) / 60 // Ceiling divide by 60 for mobile. (TODO: add 80 for desktop?)

			//prVal("length", length)
			//prVal("numLinesApprox", numLinesApprox)

			if numLinesApprox > kMaxCommentLines {

				// Truncate additional lines.
				newComment.Text = newComment.Text[:i+1]

				if linesLeft < 0 {
					linesLeft = 0
				}

				// Truncate last line if too long.
				if length > linesLeft*kCharsPerLine {
					newComment.Text[i] = newComment.Text[i][:linesLeft*kCharsPerLine]
				}

				// End the line with ellipsis.
				newComment.Text[i] += "..."

				newComment.IsExpandible = true

				break
			}
		}

		// Set the postId
		newComment.PostId = postId

		// Compare current path to previous path.
		pathLengthDiff = pathLen - prevPathDepth
		prVal("pathLen", pathLen)
		prVal("pathLengthDiff", pathLengthDiff)

		// Assign pPrevComment to the be the parent of the new node.
		if pathLengthDiff <= 0 { // we're a sibling of the previous comment, or its parent, grandparent, etc.
			for i := int64(0); i < 1-pathLengthDiff; i++ {
				pr("  pPrevComment = pPrevComment.Parent")
				pPrevComment = pPrevComment.Parent

				assertMsg(pPrevComment != nil, "We are now pointing to the nil parent, so we went up too many levels!")
			}
		} else {
			assertMsg(pathLengthDiff == 1, "We would have something weird here, a comment with grandchildren but no children.")
		}

		// Add newComment as a child of pPrevComment.
		prVal("pPrevComment.Children", pPrevComment.Children)
		pPrevComment.Children = append(pPrevComment.Children, &newComment)
		newComment.Parent = pPrevComment

		// Remember previous values.
		prevPathDepth = pathLen
		pPrevComment = &newComment

		prVal("pPrevComment.Children", pPrevComment.Children)
	}
	check(rows.Err())

	calcCommentQuality(&headComment)

	pr("Unsorted comments")
	prComments(&headComment, 0)

	sortComments(&headComment)

	pr("Sorted comments")
	prComments(&headComment, 0)

	// TODO: sort the comments here, by quality.

	//prVal("headComment", headComment)

	return headComment, upCommentVotes, downCommentVotes
}
