package main

import (
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

	IsChildrenStart	bool
	IsChildrenEnd	bool
}



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

// This is the one we're using.
// Read comments from the database, then convert it into a flattened tag format that the template file can easily render.
func ReadCommentTagsFromDB(postId int64) (commentTags []CommentTag) {
	prevPathDepth := int64(0)
	var pathLengthDiff int64

	// The simpler way for now:
	rows := DbQuery(`SELECT PostId, Text, array_length(Path, 1), u.Name
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

	return
}