package main

import (
	"encoding/json"
	"net/http"
	"github.com/lib/pq"
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
	Text			string

	IsHead			bool
	IsChildrenStart	bool
	IsChildrenEnd	bool
}



//////////////////////////////////////////////////////////////////////////////
//
// ajax poll vote
//
//////////////////////////////////////////////////////////////////////////////
func ajaxCreateComment(w http.ResponseWriter, r *http.Request) {
	pr(co_, "ajaxCreateComment")
	prVal(co_, "r.Method", r.Method)

	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	userId := GetSession(r);
	if userId == -1 { // Secure cookie not found.  Either session expired, or someone is hacking.
		// So go to the register page.
		pr(go_, "Must be logged in to create a comment.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	prVal(co_, "userId", userId);

    //parse request to struct
    var newComment struct {
		PostId		int64
		ParentId	int64
		CommentText	string
	}

    err := json.NewDecoder(r.Body).Decode(&newComment)
    if err != nil {
		prVal(co_, "Failed to decode json body", r.Body)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    prVal(co_, "=======>>>>> newComment", newComment)


	// Get the postId and path from the parent's info, in the database.
	var postId		int64
	var numChildren	int64
	var parentPath	[]int64
	{
		rows := DbQuery("SELECT PostId, NumChildren, Path FROM $$Comment WHERE Id = $1::bigint", newComment.ParentId)
		defer rows.Close()
		if rows.Next() {
			err := rows.Scan(&postId, &numChildren, pq.Array(&parentPath))
			check(err)
		} else {
			// If it's not in the database, it must be because it has Id = -1 (the top-level post)...
			assert(newComment.ParentId == -1)

	 		// The head comment of the tree, must be added!
	 		// This allows us to maintain a count of top-level posts, in this head record's NumChildren.
			DbExec(`INSERT INTO vz.Comment (Id, PostId, UserId, ParentId, Text, Path, NumChildren)
					VALUES (-1, $1::bigint, -1, -1, '', '{}'::bigint[], 0);`,
					newComment.PostId)
		}
		check(rows.Err())
	}

	// Determine what the new path should be.
	// e.g	Parent path:	1, 2, 3
	//      Child0 path: 	1, 2, 3, 0
	//      Child1 path: 	1, 2, 3, 1
	//      New Child path: [1, 2, 3] + (NumChildren + 1)

	prVal(co_, "parentPath", parentPath)
	prVal(co_, "len(parentPath)", len(parentPath))

	newPath := append(parentPath, numChildren)

	// TODO: add a database transaction here.
	//       See: http://go-database-sql.org/prepared.html

    // Send the new comment to the database.
	DbExec(
		`INSERT INTO $$Comment (PostId, UserId, ParentId, Text, Path)
	     VALUES ($1::bigint, $2::bigint, $3::bigint, $4, $5::bigint[])`,
		postId,
		userId,
		newComment.ParentId,
		newComment.CommentText,
		pq.Array(newPath))
	// Increment the parent's number of children.
	DbExec(`UPDATE $$Comment SET NumChildren = NumChildren + 1 WHERE Id = $1::bigint`, newComment.ParentId)

    // create json response from struct
    a, err := json.Marshal(true)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Write(a)
}

// This is the one we're using.
// Read comments from the database, then convert it into a flattened tag format that the template file can easily render.
func ReadCommentTagsFromDB(postId int64) []CommentTag {
	prevPathDepth := int64(0)
	var pathLengthDiff int64

	commentTags := []CommentTag{} /*
		CommentTag{
			Id:		  -1,
			ParentId: -1,
			IsHead:	  true,
	}}*/

	// The simpler way for now:
	rows := DbQuery(`SELECT c.Id, Text, COALESCE(u.Name, ''), COALESCE(array_length(Path, 1), 0)
					 FROM $$Comment c
					 LEFT JOIN $$User u
					 ON c.UserId = u.Id
					 WHERE PostId = $1::bigint
					 ORDER BY Path`,
				 postId)
	defer rows.Close()
	for rows.Next() {
		var pathLen	 	 	int64
		var newCommentTag	CommentTag

		err := rows.Scan(&newCommentTag.Id, &newCommentTag.Text, &newCommentTag.Username, &pathLen)
		check(err)

		// Compare current path to previous path.
		// Then we assign prevPathDepth to be the parent of the new node.
		pathLengthDiff = pathLen - prevPathDepth
		if pathLengthDiff <= 0 {    // we're a sibling of the previous comment's parent, grandparent, great greatparent, etc.
			for i := int64(0); i < -pathLengthDiff; i++ {
				commentTags = append(commentTags, CommentTag{ IsChildrenEnd: true })
			}
		} else if pathLengthDiff == 1 { // we're a child of the previous comment
			commentTags = append(commentTags, CommentTag{ IsChildrenStart: true })
		} else {
			assertMsg(pathLengthDiff == 0, "We would have something weird here, a comment with grandchildren but no children.")

			// Note: We made it here, so we're a sibling of the previous comment.
		}

		// Add the text of the comment
		commentTags = append(commentTags, newCommentTag)

		prevPathDepth = pathLen
	}
	check(rows.Err())

	// Close out our existing child comment depth.
	for i := int64(0); i < -pathLengthDiff; i++ {
		commentTags = append(commentTags, CommentTag{ IsChildrenEnd: true })
	}

	prVal(co_, "ReadCommentTagsFromDB returning", commentTags)

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