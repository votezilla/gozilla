package main

import (
	"encoding/json"
	"github.com/lib/pq"
	"net/http"
	"strings"
)

const (
	kMaxCommentLines = 6
	kCharsPerLine    = 60  // 60 for mobile.  80 would be desktop, but there is no way to detect the difference yet.
)

// This is also a comment tree.
type Comment struct {
	Id				int64		// id == -1 if pointing to the Post. (So not a comment, but the children are all L0 comments.)
	Username		string
	Text			string

	Children		[]*Comment
	Parent			*Comment
}

// For representing a hierarchical tree of comments in a flattened list.
type CommentTag struct {
	Id				int64

	Username		string
	Text			[]string

	IsHead			bool
	IsChildrenStart	bool
	IsChildrenEnd	bool

	IsExpandible	bool

	VoteTally		int64
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
		Id			int64
		PostId		int64
		ParentId	int64
		Text		string
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
			arr := pq.Int64Array{}  // This weirdness is required for scanning into []int64

			err := rows.Scan(&arr)
			check(err)

			newPath = []int64(arr)  // This weirdness is required for scanning into []int64
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
		CommentId	int64
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
			assert(false);
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
// read comment tags from db
//
// This is the one we're using.
// Read comments from the database, then convert it into a flattened tag format that the template file can easily render.
//
// TODO: We'll eventually need to call ReadCommentsFromDB, so the children of each comment can be sorted by upvote.
//
//////////////////////////////////////////////////////////////////////////////
func ReadCommentTagsFromDB(postId int64) []CommentTag {
	prevPathDepth := int64(0)
	var pathLengthDiff int64

	pr("ReadCommentTagsFromDB:")

	commentTags := []CommentTag{}

	// The simpler way for now:
	rows := DbQuery(`SELECT c.Id, Text, COALESCE(u.Username, ''), COALESCE(array_length(Path, 1), 0)
					 FROM $$Comment c
					 LEFT JOIN $$User u
					 ON c.UserId = u.Id
					 WHERE PostId = $1::bigint
					 ORDER BY Path`,
				 postId)
	defer rows.Close()
	for rows.Next() {
		var pathLen	 	 	int64
		var commentText		string
		var newCommentTag	CommentTag

		err := rows.Scan(&newCommentTag.Id, &commentText, &newCommentTag.Username, &pathLen)
		check(err)

		// Compare current path to previous path.
		// Then we assign prevPathDepth to be the parent of the new node.
		pathLengthDiff = pathLen - prevPathDepth
		//prVal("pathLen", pathLen)
		//prVal("pathLengthDiff", pathLengthDiff)
		if pathLengthDiff <= 0 {    // we're a sibling of the previous comment's parent, grandparent, great greatparent, etc.
			for i := int64(0); i < -pathLengthDiff; i++ {
				//pr("  tag: IsChildrenEnd")
				commentTags = append(commentTags, CommentTag{ IsChildrenEnd: true })
			}
		} else if pathLengthDiff == 1 { // we're a child of the previous comment
			//pr("  tag: IsChildrenStart")
			commentTags = append(commentTags, CommentTag{ IsChildrenStart: true })
		} else {
			assertMsg(pathLengthDiff == 0, "We would have something weird here, a comment with grandchildren but no children.")

			// Note: We made it here, so we're a sibling of the previous comment.
		}

		// Convert newlines to be HTML-friendly -
		//    split into lines so the template file can handle it,
		//    add elipsis if too long.
		newCommentTag.Text = strings.Split(commentText, "\n")

		numLinesApprox := 0
		for i, textLine := range newCommentTag.Text {
			if i > 0 {
				numLinesApprox++
			}

			linesLeft := kMaxCommentLines - numLinesApprox


			length := len(textLine)
			numLinesApprox += (length + 59) / 60; // Ceiling divide by 60 for mobile. (TODO: add 80 for desktop?)

			//prVal("length", length)
			//prVal("numLinesApprox", numLinesApprox)

			if numLinesApprox > kMaxCommentLines {

				// Truncate additional lines.
				newCommentTag.Text = newCommentTag.Text[:i+1]

				if linesLeft < 0 {
					linesLeft = 0
				}

				// Truncate last line if too long.
				if length > linesLeft * kCharsPerLine {
					newCommentTag.Text[i] = newCommentTag.Text[i][:linesLeft * kCharsPerLine]
				}

				// End the line with ellipsis.
				newCommentTag.Text[i] += "..."

				newCommentTag.IsExpandible = true

				break
			}
		}

		//prVal("newCommentTag.Text", newCommentTag.Text)


		//         then, ... is a linkn to #this_comment
		//

		// Add this comment tag to the list.
		//prVal("  tag: Text", newCommentTag.Text)
		commentTags = append(commentTags, newCommentTag)

		prevPathDepth = pathLen
	}
	check(rows.Err())

	// Close out our existing child comment depth.
	//prVal("closing prevPathDepth", prevPathDepth)
	for i := int64(0); i < prevPathDepth; i++ {
		//pr("  tag: IsChildrenEnd")
		commentTags = append(commentTags, CommentTag{ IsChildrenEnd: true })
	}

	//prVal("ReadCommentTagsFromDB returning", commentTags)

	//commentTags = commentTags[1:len(commentTags)-1]

	return commentTags
}

/* KEEP THIS CODE!!! vv
// TODO: IT NEEDS TO DO IT THIS WAY, SINCE WE MUST SORT THE CHILDREN BY RANK VOTE.  ANYWAYS... LET'S KEEP IT HOW WE HAVE IT
//       FOR NOW, AND IMPLEMENT THIS A BIT LATER.   vvv



// NOTE: We're not using this code at the moment, so it's untested!
// Read the comment tree for a post fromm the database.
func ReadCommentsFromDB(postId int) *Comment {
	var headComment	Comment
	headComment.Id = -1
	pPrevComment  := &headComment
	//curDepth 	  := 0
	prevPathDepth := int64(0)

	var pathLengthDiff int64

	// The simpler way for now: vvv
	rows := DbQuery(`SELECT PostId, Text, array_length(Path, 1), u.Name
					 FROM $$Comment c
					 LEFT JOIN $$User u
					 ON c.UserId = u.Id
					 WHERE PostId = $1::bigint
					 ORDER BY Path`,
					 postId)
	defer rows.Close()
	for rows.Next() {
		var pathLen	 int64
		var newComment Comment

		err := rows.Scan(&newComment.Id, &newComment.Text, &newComment.Username, &pathLen)
		check(err)

		// Compare current path to previous path.
		// Then we assign prevPathDepth to be the parent of the new node.
		pathLengthDiff = pathLen - prevPathDepth
		if pathLengthDiff <= 0 {  // Current comment is a child of the previous comment's parent, grandparent, etc.
			for i := int64(0); i < int64(1) - pathLengthDiff; i++ { // 0->parent, 1->grantparent, 2->great grandparent, etc.
				pPrevComment = pPrevComment.Parent
			}
		} else {
			assertMsg(pathLengthDiff == 1, "We would have something weird here, a comment with grandchildren but no children.")

			// Note: if pathLengthDiff == 1, we have what we want because pPrevComment is already the parent of newComment.
		}

		// Add the new comment as a child of pPrevComment.
		newComment.Parent = pPrevComment
		pPrevComment.Children = append(pPrevComment.Children, &newComment)

		pPrevComment = &newComment
		prevPathDepth = pathLen
	}
	check(rows.Err())

	return &headComment
}

*/