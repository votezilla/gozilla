package main

import (
    "fmt"
    "io"
    "io/ioutil"
    "path/filepath"
    "strconv"
	"time"
    "os"
    "willnorris.com/go/imageproxy"
)

const (
	image_Unprocessed		= 0
	image_Downsampled		= 1 // 125 x 75
	
	image_DownsampleError	= -1
)

type DownsampleResult struct {
	postId	int
	err		error
}

// Download image from imageUrl, use outputNameId to form name before extension, extension stays the same.
func downloadImage(imageUrl string, outputNameId int) error {
    resp, err := httpGet(imageUrl, 60.0)
    if err != nil {
		return err
	}
    defer resp.Body.Close()

    //open a file for writing
    file, err := os.Create("./static/downloads/" + strconv.Itoa(outputNameId) + filepath.Ext(imageUrl))
    if err != nil {
		return err
	}
    defer file.Close()
    
    _, err = io.Copy(file, resp.Body)
    if err != nil {
		return err
	}

    fmt.Println("Success!")
    return nil
}
		
// Download image from imageUrl, use outputNameId to form name before extension, extension stays the same.
func downsampleImage(imageUrl string, outputNameId int) error {
	resp, err := httpGet(imageUrl, 60.0)
    if err != nil {
		return err
	}
    defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
		return err
	}
	
	downsampledImg, err := imageproxy.Transform(
		bytes,
		imageproxy.Options{
			Width:		125,
			Height:		75,
			Format:		"jpeg",
			SmartCrop:	true,
		},
	)
    if err != nil {
		return err
	}
	
	check(ioutil.WriteFile(
		"./static/thumbnails/" + strconv.Itoa(outputNameId) + ".jpeg",
		downsampledImg,
		0644,
	))
	fmt.Println("Success!")
	return nil
}

// Downsample an image asynchronously, return infomation about id and error status to the channel after.
func downsamplePostImage(imageUrl string, id int, c chan DownsampleResult) {
	prf(is_, "Downsampling #%d image %s\n", id, imageUrl)
	
	err := downsampleImage(imageUrl, id)
	
	prf(is_, "Result for #%d image %s: %v\n", id, imageUrl, err)
	
	c <- DownsampleResult{id, err}
}

// Remove an item from a list of ints.
func removeItem(s []int, item int) []int {
	for i, x := range s {
		if x == item {
			s[i] = s[len(s)-1]
			return s[:len(s)-1]
		}
	}
	return s
}
	
//////////////////////////////////////////////////////////////////////////////
//
// image server - Continually checks for new images to shrink.  Images must be shrunk
//				  to thumbnail size for faster webpage loads.
//
//////////////////////////////////////////////////////////////////////////////
func ImageServer() {
	if flags.test != "" {
		check(downloadImage("http://numImages.imgur.com/m1UIjW1.jpg", 999999))
		check(downsampleImage("https://octodex.github.com/images/codercat.jpg", 222222))
	}
	
	pr(ns_, "========================================")
	pr(ns_, "======== STARTING IMAGE SERVER =========")
	pr(ns_, "========================================\n")
	
	for {
		pr(ns_, "========================================")
		pr(ns_, "=========== FETCHING IMAGES ============")
		pr(ns_, "========================================\n")

		// Grab a batch of images to downsample from the database.
		const kImageBatchSize = 5
		imageURLs := make([]string, kImageBatchSize)
		ids		  := make([]int,	kImageBatchSize)

		rows := DbQuery(`SELECT UrlToImage, Id FROM votezilla.NewsPost WHERE ThumbnailStatus = 0 AND UrlToImage <> ''
						 LIMIT ` + strconv.Itoa(kImageBatchSize) + ";")
		numImages := 0	
		defer rows.Close()
		for rows.Next() {
			err := rows.Scan(&imageURLs[numImages], &ids[numImages])
			check(err)
			numImages++
		}
		check(rows.Err())

		prVal(is_, "imageURLs", imageURLs)
		prVal(is_, "ids", ids)

		// Download and downsample the images in paralle.
		c := make(chan DownsampleResult)
		timeout := time.After(60 * time.Second)

		for i := 0; i < numImages; i++ {
			go downsamplePostImage(imageURLs[i], ids[i], c)
		}

		numImagesProcessed := 0

		downsampleImagesLoop: for {
			select {
				case downsampleResult := <-c:
					thumbnailStatus := image_Downsampled
					if downsampleResult.err != nil {
						thumbnailStatus = image_DownsampleError
					}

					DbQuery(
						`UPDATE votezilla.NewsPost 
						 SET ThumbnailStatus = $1
						 WHERE Id = $2::bigint`,
						thumbnailStatus,
						downsampleResult.postId)
					
					// Remove this from the list of ids, so we can tell which ids were never processed.
					ids = removeItem(ids, downsampleResult.postId)

					numImagesProcessed++
					if numImagesProcessed == numImages {
						pr(ns_, "Processed all images!")
						break downsampleImagesLoop
					}
				case <- timeout:
					pr(ns_, "Timeout!")
					
					// Set status to -1 for any images that timed out.
					for _, id := range ids {
						prVal(ns_, "Removing timed out id", id)
						DbQuery(
							`UPDATE votezilla.NewsPost 
							 SET ThumbnailStatus = -1
							 WHERE Id = $1::bigint`,
							id)
					}
					
					break downsampleImagesLoop
			}		
		}
	}
}