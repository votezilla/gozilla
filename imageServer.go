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

var (
	newsSourceIcons map[string]string = map[string]string{
		"abc-news-au": "https://icons.better-idea.org/icon?url=http://www.abc.net.au/news&size=70..120..200",
		"al-jazeera-english": "https://icons.better-idea.org/icon?url=http://www.aljazeera.com&size=70..120..200",
		"ars-technica": "https://icons.better-idea.org/icon?url=http://arstechnica.com&size=70..120..200",
		"associated-press": "https://icons.better-idea.org/icon?url=https://apnews.com/&size=70..120..200",
		"bbc-news": "https://icons.better-idea.org/icon?url=http://www.bbc.co.uk/news&size=70..120..200",
		"bbc-sport": "https://icons.better-idea.org/icon?url=http://www.bbc.co.uk/sport&size=70..120..200",
		"bloomberg": "https://icons.better-idea.org/icon?url=http://www.bloomberg.com&size=70..120..200",
		"breitbart-news": "https://icons.better-idea.org/icon?url=http://www.breitbart.com&size=70..120..200",
		"business-insider": "https://pbs.twimg.com/profile_images/887662979902304257/azSzxYkB.jpg",
		"business-insider-uk": "https://pbs.twimg.com/profile_images/890152475067650048/6MuA0NTT.jpg",
		"buzzfeed": "https://icons.better-idea.org/icon?url=https://www.buzzfeed.com&size=70..120..200",
		"cnbc": "https://icons.better-idea.org/icon?url=http://www.cnbc.com&size=70..120..200",
		"cnn": "https://icons.better-idea.org/icon?url=http://us.cnn.com&size=70..120..200",
		"daily-mail": "https://icons.better-idea.org/icon?url=http://www.dailymail.co.uk/home/index.html&size=70..120..200",
		"der-tagesspiegel": "https://icons.better-idea.org/icon?url=http://www.tagesspiegel.de&size=70..120..200",
		"die-zeit": "https://icons.better-idea.org/icon?url=http://www.zeit.de/index&size=70..120..200",
		"engadget": "https://icons.better-idea.org/icon?url=https://www.engadget.com&size=70..120..200",
		"entertainment-weekly": "https://icons.better-idea.org/icon?url=http://www.ew.com&size=70..120..200",
		"espn": "https://icons.better-idea.org/icon?url=http://espn.go.com&size=70..120..200",
		"espn-cric-info": "https://icons.better-idea.org/icon?url=http://www.espncricinfo.com/&size=70..120..200",
		"financial-times": "https://icons.better-idea.org/icon?url=http://www.ft.com/home/uk&size=70..120..200",
		"focus": "https://icons.better-idea.org/icon?url=http://www.focus.de&size=70..120..200",
		"football-italia": "https://icons.better-idea.org/icon?url=http://www.football-italia.net&size=70..120..200",
		"fortune": "https://icons.better-idea.org/icon?url=http://fortune.com&size=70..120..200",
		"four-four-two": "https://icons.better-idea.org/icon?url=http://www.fourfourtwo.com/news&size=70..120..200",
		"fox-sports": "https://icons.better-idea.org/icon?url=http://www.foxsports.com&size=70..120..200",
		"google-news": "https://icons.better-idea.org/icon?url=https://news.google.com&size=70..120..200",
		"gruenderszene": "https://icons.better-idea.org/icon?url=http://www.gruenderszene.de&size=70..120..200",
		"hacker-news": "https://icons.better-idea.org/icon?url=https://news.ycombinator.com&size=70..120..200",
		"handelsblatt": "https://icons.better-idea.org/icon?url=http://www.handelsblatt.com&size=70..120..200",
		"ign": "https://icons.better-idea.org/icon?url=http://www.ign.com&size=70..120..200",
		"independent": "https://icons.better-idea.org/icon?url=http://www.independent.co.uk&size=70..120..200",
		"mashable": "https://icons.better-idea.org/icon?url=http://mashable.com&size=70..120..200",
		"metro": "https://icons.better-idea.org/icon?url=http://metro.co.uk&size=70..120..200",
		"mirror": "https://icons.better-idea.org/icon?url=http://www.mirror.co.uk/&size=70..120..200",
		"mtv-news": "https://icons.better-idea.org/icon?url=http://www.mtv.com/news&size=70..120..200",
		"mtv-news-uk": "https://icons.better-idea.org/icon?url=http://www.mtv.co.uk/news&size=70..120..200",
		"national-geographic": "https://icons.better-idea.org/icon?url=http://news.nationalgeographic.com&size=70..120..200",
		"new-scientist": "https://icons.better-idea.org/icon?url=https://www.newscientist.com/section/news&size=70..120..200",
		"newsweek": "https://icons.better-idea.org/icon?url=http://www.newsweek.com&size=70..120..200",
		"new-york-magazine": "https://icons.better-idea.org/icon?url=http://nymag.com&size=70..120..200",
		"nfl-news": "https://icons.better-idea.org/icon?url=http://www.nfl.com/news&size=70..120..200",
		"polygon": "https://icons.better-idea.org/icon?url=http://www.polygon.com&size=70..120..200",
		"recode": "https://icons.better-idea.org/icon?url=http://www.recode.net&size=70..120..200",
		"reddit-r-all": "https://icons.better-idea.org/icon?url=https://www.reddit.com/r/all&size=70..120..200",
		"reuters": "https://icons.better-idea.org/icon?url=http://www.reuters.com&size=70..120..200",
		"spiegel-online": "https://icons.better-idea.org/icon?url=http://www.spiegel.de&size=70..120..200",
		"t3n": "https://icons.better-idea.org/icon?url=http://t3n.de&size=70..120..200",
		"talksport": "https://icons.better-idea.org/icon?url=http://talksport.com&size=70..120..200",
		"techcrunch": "https://icons.better-idea.org/icon?url=https://techcrunch.com&size=70..120..200",
		"techradar": "https://icons.better-idea.org/icon?url=http://www.techradar.com&size=70..120..200",
		"the-economist": "https://icons.better-idea.org/icon?url=http://www.economist.com&size=70..120..200",
		"the-guardian-au": "https://icons.better-idea.org/icon?url=https://www.theguardian.com/au&size=70..120..200",
		"the-guardian-uk": "https://icons.better-idea.org/icon?url=https://www.theguardian.com/uk&size=70..120..200",
		"the-hindu": "https://icons.better-idea.org/icon?url=http://www.thehindu.com&size=70..120..200",
		"the-huffington-post": "https://icons.better-idea.org/icon?url=http://www.huffingtonpost.com&size=70..120..200",
		"the-lad-bible": "https://icons.better-idea.org/icon?url=http://www.theladbible.com&size=70..120..200",
		"the-new-york-times": "https://icons.better-idea.org/icon?url=http://www.nytimes.com&size=70..120..200",
		"the-next-web": "https://icons.better-idea.org/icon?url=http://thenextweb.com&size=70..120..200",
		"the-sport-bible": "https://icons.better-idea.org/icon?url=http://www.thesportbible.com&size=70..120..200",
		"the-telegraph": "https://icons.better-idea.org/icon?url=http://www.telegraph.co.uk&size=70..120..200",
		"the-times-of-india": "https://icons.better-idea.org/icon?url=http://timesofindia.indiatimes.com&size=70..120..200",
		"the-verge": "https://icons.better-idea.org/icon?url=http://www.theverge.com&size=70..120..200",
		"the-wall-street-journal": "https://icons.better-idea.org/icon?url=http://www.wsj.com&size=70..120..200",
		"the-washington-post": "https://icons.better-idea.org/icon?url=https://www.washingtonpost.com&size=70..120..200",
		"time": "https://icons.better-idea.org/icon?url=http://time.com&size=70..120..200",
		"usa-today": "https://icons.better-idea.org/icon?url=http://www.usatoday.com/news&size=70..120..200",
		"wired-de": "https://icons.better-idea.org/icon?url=https://www.wired.de&size=70..120..200",
		"wirtschafts-woche": "https://icons.better-idea.org/icon?url=http://www.wiwo.de&size=70..120..200",
	}
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
		
// Download image from imageUrl, use outputName to form name before extension, extension stays the same.
func downsampleImage(imageUrl string, directory string, outputName string, extension string, width int, height int) error {
	prf(is_, "downsampleImage %s -> %s.%s", imageUrl, outputName, extension)
	
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
			Width:		float64(width),
			Height:		float64(height),
			Format:		extension,
			SmartCrop:	true,
		},
	)
    if err != nil {
		return err
	}
	
	check(ioutil.WriteFile(
		"./static/" + directory + "/" + outputName + "." + extension,
		downsampledImg,
		0644,
	))
	prf(is_, "Success!")
	return nil
}

// Downsample an image asynchronously, return infomation about id and error status to the channel after.
func downsamplePostImage(imageUrl string, id int, c chan DownsampleResult) {
	prf(is_, "Downsampling #%d image %s\n", id, imageUrl)
	
	err := downsampleImage(imageUrl, "thumbnails", strconv.Itoa(id), "jpeg", 125, 75)
	
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
		check(downsampleImage("https://octodex.github.com/images/codercat.jpg", "thumbnails", "cat", "jpeg", 125, 75))
		return
	}
	
	if flags.mode == "fetchNewsSourceIcons" {
		for newsSource, imageUrl := range newsSourceIcons {
			check(downsampleImage(imageUrl, "newsSourceIcons", newsSource, "png", 16, 16))
		}
		return
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
		
		query := `
			SELECT UrlToImage, Id 
			FROM $$NewsPost 
			WHERE ThumbnailStatus = 0 AND UrlToImage <> ''
			ORDER BY COALESCE(PublishedAt, Created) DESC
			LIMIT ` + strconv.Itoa(kImageBatchSize) + ";"
			
		prVal(ns_, "query", query)

		rows := DbQuery(query)
		
		numImages := 0	
		for rows.Next() {
			err := rows.Scan(&imageURLs[numImages], &ids[numImages])
			check(err)
			numImages++
		}
		check(rows.Err())
		rows.Close()

		prVal(is_, "numImages", numImages)
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

					DbExec(
						`UPDATE $$NewsPost 
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
						DbExec(
							`UPDATE $$NewsPost 
							 SET ThumbnailStatus = -1
							 WHERE Id = $1::bigint`,
							id)
					}
					
					break downsampleImagesLoop
			}		
		}
		
		DbTrackOpenConnections()
	}
}