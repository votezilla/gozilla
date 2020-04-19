package main

import (
)

/*
CREATE TABLE vz.Comment (
    Id                  BIGSERIAL   PRIMARY KEY,

    PostId              BIGINT NOT NULL, -- Which post this is a comment on.
    UserId              BIGINT NOT NULL, -- The user who left this comment.
    ParentId            BIGINT,          -- The parent comment, hierarchially.

    Text                VARCHAR(40000) NOT NULL, -- 40,000 is Reddit's maximum text length.
    PrevRevisions       VARCHAR(40000)[],

    -- Materialized path - this sorts it into a tree hierarchy for us, though
    --                     we'll still need to sort by aye/nay vote later...
    --                     think if this is the right algorithm.
    MaterializedPath    BIGINT NOT NULL, -- int64, bits used as follows:
                                         --    16b, 16b, 8b, 8b, 4b, 4b, 4b, 4b
                                         --     L0   L1  L2  L3  L4  L5  L6, L7
    -- Note: for now we'll fetch all comments for a post,
    --       then SORT BY MaterializedPath.
    --       in Go, we'll parse these into a hierarchy, will need pointers.
    --              Then sort each list by aye/nay vote.
    --       Eventually... we'll need to optimize this.

    Created             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    Deleted             BOOL DEFAULT false
);
CREATE UNIQUE INDEX comment_path_index ON vz.Comment (Id, MaterializedPath);  -- UNIQUE index is a btree, so sorting by MaterializedPath should be fast.
*/

/*
const (
	// Comment ordinality bit decoding constants:
	//   int64, bits used as follows:
	//      16b, 16b, 8b, 8b, 4b, 4b, 4b, 4b
	//       L0   L1  L2  L3  L4  L5  L6, L7
	ORD_NUM_LEVELS = 8

	ORD_NUM_BITS_0 = uint64(16)
	ORD_NUM_BITS_1 = uint64(16)
	ORD_NUM_BITS_2 = uint64(8)
	ORD_NUM_BITS_3 = uint64(8)
	ORD_NUM_BITS_4 = uint64(4)
	ORD_NUM_BITS_5 = uint64(4)
	ORD_NUM_BITS_6 = uint64(4)
	ORD_NUM_BITS_7 = uint64(4)

	ORD_BIT_SHIFT_0 = ORD_NUM_BITS_1 + ORD_NUM_BITS_2 + ORD_NUM_BITS_3 + ORD_NUM_BITS_4 + ORD_NUM_BITS_5 + ORD_NUM_BITS_6 + ORD_NUM_BITS_7
	ORD_BIT_SHIFT_1 = ORD_NUM_BITS_2 + ORD_NUM_BITS_3 + ORD_NUM_BITS_4 + ORD_NUM_BITS_5 + ORD_NUM_BITS_6 + ORD_NUM_BITS_7
	ORD_BIT_SHIFT_2 = ORD_NUM_BITS_3 + ORD_NUM_BITS_4 + ORD_NUM_BITS_5 + ORD_NUM_BITS_6 + ORD_NUM_BITS_7
	ORD_BIT_SHIFT_3 = ORD_NUM_BITS_4 + ORD_NUM_BITS_5 + ORD_NUM_BITS_6 + ORD_NUM_BITS_7
	ORD_BIT_SHIFT_4 = ORD_NUM_BITS_5 + ORD_NUM_BITS_6 + ORD_NUM_BITS_7
	ORD_BIT_SHIFT_5 = ORD_NUM_BITS_6 + ORD_NUM_BITS_7
	ORD_BIT_SHIFT_6 = ORD_NUM_BITS_7
	ORD_BIT_SHIFT_7 = 0

	ORD_BIT_MASK_0 = (1 << ORD_NUM_BITS_0) - 1
	ORD_BIT_MASK_1 = (1 << ORD_NUM_BITS_1) - 1
	ORD_BIT_MASK_2 = (1 << ORD_NUM_BITS_2) - 1
	ORD_BIT_MASK_3 = (1 << ORD_NUM_BITS_3) - 1
	ORD_BIT_MASK_4 = (1 << ORD_NUM_BITS_4) - 1
	ORD_BIT_MASK_5 = (1 << ORD_NUM_BITS_5) - 1
	ORD_BIT_MASK_6 = (1 << ORD_NUM_BITS_6) - 1
	ORD_BIT_MASK_7 = (1 << ORD_NUM_BITS_7) - 1
)

*/

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

/*

func DecodeMaterializedPath(materializedPath uint64, ordinalityAtLevel *[ORD_NUM_LEVELS]uint64) {
	ordinalityAtLevel[0] = (materializedPath >> ORD_BIT_SHIFT_0) & ORD_BIT_MASK_0
	ordinalityAtLevel[1] = (materializedPath >> ORD_BIT_SHIFT_1) & ORD_BIT_MASK_1
	ordinalityAtLevel[2] = (materializedPath >> ORD_BIT_SHIFT_2) & ORD_BIT_MASK_2
	ordinalityAtLevel[3] = (materializedPath >> ORD_BIT_SHIFT_3) & ORD_BIT_MASK_3
	ordinalityAtLevel[4] = (materializedPath >> ORD_BIT_SHIFT_4) & ORD_BIT_MASK_4
	ordinalityAtLevel[5] = (materializedPath >> ORD_BIT_SHIFT_5) & ORD_BIT_MASK_5
	ordinalityAtLevel[6] = (materializedPath >> ORD_BIT_SHIFT_6) & ORD_BIT_MASK_6
	ordinalityAtLevel[7] = (materializedPath >> ORD_BIT_SHIFT_7) & ORD_BIT_MASK_7
}


func EncodeMaterializedPath(ordinalityAtLevel [ORD_NUM_LEVELS]uint64) uint64 {
	return (ordinalityAtLevel[0] << ORD_BIT_SHIFT_0) &
		   (ordinalityAtLevel[1] << ORD_BIT_SHIFT_1) &
		   (ordinalityAtLevel[2] << ORD_BIT_SHIFT_2) &
		   (ordinalityAtLevel[3] << ORD_BIT_SHIFT_3) &
		   (ordinalityAtLevel[4] << ORD_BIT_SHIFT_4) &
		   (ordinalityAtLevel[5] << ORD_BIT_SHIFT_5) &
		   (ordinalityAtLevel[6] << ORD_BIT_SHIFT_6) &
		   (ordinalityAtLevel[7] << ORD_BIT_SHIFT_7)
}*/

// Read the comment tree for a post fromm the database.
func ReadCommentsFromDB(postId int) *Comment {
	var headComment	Comment
	headComment.Id = -1
	pPrevComment  := &headComment
	//curDepth 	  := 0
	prevPathDepth := int64(0)

	var pathLengthDiff int64



		//			 SELECT PostId, Text, array_length(Path, 1), u.Name FROM vz.Comment c
		//			 LEFT JOIN vz.User u
		//			 ON c.UserId = u.Id
		//			 WHERE PostId = 0
		//		     ORDER BY Path;



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
/*	rows := DbQuery(`SELECT PostId, UserId, Text, Path FROM $$Comment
					 WHERE PostId = $1::bigint ORDER BY Path`,
					 postId)
	defer rows.Close()
	for rows.Next() {
		var postId, userId uint64
		var text		   string
		var path		   []int64

		err := rows.Scan(&postId, &userId, &text, pq.Array(&path))
		check(err)

		// Compare current path to previous path.
		lengthDiff := len(path) - len(*prevPath)
		if lengthDiff == 1 {         // Current comment is a child of previous comment

		} else if lengthDiff <= 0 {   // Current comment is a child of the previous comment's parent, grandparent, etc.

		} else {
			panic("lengthDiff is an unexpected value")
		}

		prevPath = &path
	}
	check(rows.Err())
/*
	rows := DbQuery(`SELECT PostId, UserId, Text, MaterializedPath FROM $$Comment
					 WHERE PollId = $1::bigint ORDER BY MaterializedPath`, postId)
	defer rows.Close()
	for rows.Next() {
		var postId, userId, materializedPath uint64
		var text							 string

		err := rows.Scan(&postId, &userId, &text, &materializedPath)
		check(err)

		var ordinalityAtLevel [ORD_NUM_LEVELS]uint64
		DecodeMaterializedPath(materializedPath, &ordinalityAtLevel)

		// TODO: something here!
		nyi()
	}
	check(rows.Err())
*/


	//comment.Id = -1  // Meaning we're pointing to the top-level Post. (So not a comment, but the children are all L0 comments.)
}

/*
func FlattenComments(headComment *Comment, flattened_comments *[]CommentTag) {
//<<
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
	Id				int64	 x
	Username		string	x
	Text			string	x

	IsChildrenStart	bool
	IsChildrenEnd	bool
}
//>>


}*/

func ReadCommentTagsFromDB(postId int64) (commentTags []CommentTag) {
	//var headComment	Comment
	//headComment.Id = -1
	//curDepth 	  := 0

	prevPathDepth := int64(0)
	var pathLengthDiff int64


		//			 SELECT PostId, Text, array_length(Path, 1), u.Name FROM vz.Comment c
		//			 LEFT JOIN vz.User u
		//			 ON c.UserId = u.Id
		//			 WHERE PostId = 0
		//		     ORDER BY Path;

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

		// Add the new comment as a child of pPrevComment.
		//newComment.Parent = pPrevComment
		//pPrevComment.Children = append(pPrevComment.Children, &newComment)

		//pPrevComment = &newComment
		prevPathDepth = pathLen
	}
	check(rows.Err())

	// Close out our existing child comment depth.
	for i := int64(0); i < -pathLengthDiff; i++ {
		commentTags = append(commentTags, CommentTag{ IsChildrenEnd: true })
	}

	prVal(co_, "commentTags", commentTags)

	return
}