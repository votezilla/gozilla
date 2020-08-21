// ImageService
//
// TODO(BUG): Handle out of memory gracefully.  Should occur when Photoshop is open.
//
// NOTE: Here's a useful query for checking the progress of the imageService:
//
// 		 SELECT ThumbnailStatus, UrlToImage <> '', COUNT(*) FROM vz.NewsPost GROUP BY ThumbnailStatus, UrlToImage <> '' ORDER BY ThumbnailStatus;

package main

import (
	"errors"
	"fmt"
//	"github.com/puerkitobio/goquery"
	"github.com/rubenfonseca/fastimage"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
//	"sort"
	"strconv"
	"strings"
	"time"
	"willnorris.com/go/imageproxy"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
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
	image_DownsampledV2     = 2 // NOTE: THIS SHOULD BE THE NEW SIZE! a - 160 x 116 - thumbnail
	                            //       AND                          b - 160 x 150
	image_DownsampledV3     = 3 // V3 += LARGE THUMBNAIL              c - 570 x [preserve aspect ratio]

	image_DownsampleVersionTarget = image_DownsampledV3

	image_DownsampleError	= -1

	//genThumbPass_PollPost	= 0 // Don't need, because all polls right now use the default dino jpg.
	genThumbPass_LinkPost	= 0
	genThumbPass_NewsPost	= 1
	NUM_GEN_THUMBS_PASSES   = 2

	kImageBatchSize = 5		// Number of images to convert to thumbnails per batch
)

type UrlStatus struct {
	url		string
	status	int
}

type DownsampleResult struct {
	postId		int
	urlToImage	string
	err			error
}

type ImageSizeResult struct {
	imgSrc			string
	width			int
	height			int
	imageQuality	int		// quality, in terms of size.
	err				error
}

// Download image from imageUrl, use outputNameId to form name before extension, extension stays the same.
func downloadImage(imageUrl string, outputNameId string) error {
    resp, err := httpGet(imageUrl, 10.0)
    if err != nil {
		return err
	}
    defer resp.Body.Close()

    //open a file for writing
    file, err := os.Create("./static/downloads/" + outputNameId + filepath.Ext(imageUrl))
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

// Downloads just enough of the image (from the web) to determine its width and height.
func downloadImageSize(imageUrl string) (int, int, error) {
	prVal("downloadImageSize", imageUrl)
	_, size, err := fastimage.DetectImageType(imageUrl)
	prVal("  size", size)
	if err != nil {
		return -1, -1, err
	}
	if size == nil {
		pr("  size is nil")
		return -1, -1, errors.New("downloadImageSize gets nil size and must abort")
	}
	return int(size.Width), int(size.Height), nil
}

// Download image from imageUrl, use outputName to form name before extension, extension stays the same.
func downsampleImage(imageUrl string, directory string, outputName string, extension string, width int, height int) error {
	prf("downsampleImage %s -> %s.%s", imageUrl, outputName, extension)

	// Fix weird URLs.
	imageUrl = strings.Replace(imageUrl, "////", "//", 1)

	resp, err := httpGet(imageUrl, 25.0) 
    if err != nil {
		prf("  ERR 1 %s %s", err.Error(), imageUrl)
		return err
	}
    defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
		prf("  ERR 2 %s %s", err.Error(), imageUrl)
		return err
	}

	options := imageproxy.Options{}
	if width > 0 && height > 0 { // Smart cropping option
		options = imageproxy.Options{
			Width:		float64(width),
			Height:		float64(height),
			Format:		extension,
			SmartCrop:	true,
		}
	} else {                    // Scaled option - only one dimension is specified
		assert(width > 0 && height <= 0 || height > 0 && width <= 0)
		if width > 0	{ options.Width  = float64(width); }
		if height > 0	{ options.Height = float64(height); }
		options.Format = extension
	}
	prVal("  options", options)
	downsampledImg, err := imageproxy.Transform(bytes, options)
    if err != nil {
		prf("  ERR 3 %s %s", err.Error(), imageUrl)
		return err
	}

	err = ioutil.WriteFile(
		"./static/" + directory + "/" + outputName + "." + extension,
		downsampledImg,
		0644,
	)

	if err != nil {
		prf("  ERR 4 %s %s", err.Error(), imageUrl)
	} else {
		pr("Success downsampling image!")
	}
	return nil
}

func goDownloadImageSize(imgSrc string, c chan ImageSizeResult) {
	prf("calling gorouting downloadImageSize(%s)", imgSrc)

	width, height, err := downloadImageSize(imgSrc)

	//prf("   the result is %d, %d, %s", width, height, err)

	minDim := min_int(width, height)
	maxDim := max_int(width, height)
	imageQuality := minDim * minDim * maxDim // Rewards both the minimum dimension (to discourage banners) while also encouraging a larger area

	prf("minDim: %d maxDim: %d imageQuality %d imgSrc: %s",
			minDim, maxDim, imageQuality, imgSrc)

	c <- ImageSizeResult{imgSrc, width, height, imageQuality, err}
}


// If imgSrc is a relative URL, converts it to an absolute URL (using baseUrl).  Returns the result, or an error if unsuccessful.
func makeUrlAbsolute(imgSrc, baseUrl string) (string, error) {

	imgUrl, err := url.Parse(imgSrc)

	if err != nil {
	//	prf("Error parsing URL: %s %v %s", imgSrc, imgUrl, err)
		return "", err
	}

	if !imgUrl.IsAbs() {
	//	pr("Image URL is not absolute")

		baseUrl, err := url.Parse(baseUrl)
		if err != nil {
	//		prf("Error parsing base URL: %s %s", linkUrl.Url, err)
			return "", err
		}

		imgUrl := baseUrl.ResolveReference(imgUrl)

		//prVal("Fixed Image Url:", imgUrl)

		imgSrc = imgUrl.String()

		//prVal("Fixed imgSrc:", imgSrc)
	}

	return imgSrc, nil
}


// Downsample an image asynchronously, return infomation about id and error status to the channel after.
func downsamplePostImage(url string, currentStatus, id int, c chan DownsampleResult) {
	prf("Downsampling image #%d status %d urls %s\n", id, currentStatus, url)

	assert(image_DownsampleError <= currentStatus && currentStatus <= image_DownsampleVersionTarget)

	//image_Unprocessed		= 0
	//image_Downsampled		= 1 // 125 x 75
	//image_DownsampledV2     = 2 // NOTE: THIS SHOULD BE THE NEW SIZE! a - 160 x 116 - thumbnail
	//                            //       AND                          b - 160 x 150
	//image_DownsampledV3         // V3 += LARGE THUMBNAIL              c - 570 x [preserve aspect ratio]
	//image_DownsampleError	= -1

	var err error
	if currentStatus < image_DownsampledV2 {
		// Small thumbnail - a
		err = downsampleImage(url, "thumbnails", strconv.Itoa(id) + "a", "jpeg", 160, 116)
		if err != nil {
			prVal("downsamplePostImage called downsampleImage and then encountered some error", err.Error())
			c <- DownsampleResult{id, url, err}
			return
		}
		// Small thumbnail - b
		err = downsampleImage(url, "thumbnails", strconv.Itoa(id) + "b", "jpeg", 160, 150)
		if err != nil {
			prVal("downsamplePostImage called downsampleImage and then encountered some error", err.Error())
			c <- DownsampleResult{id, url, err}
			return
		}
	}
	if currentStatus < image_DownsampledV3 {
		// Large Thumbnail - c
		err = downsampleImage(url, "thumbnails", strconv.Itoa(id) + "c", "jpeg", 570, -1)
		if err != nil {
			prVal("downsamplePostImage called downsampleImage and then encountered some error", err.Error())
			c <- DownsampleResult{id, url, err}
			return
		}
	}
	prf("Result for #%d image %s: Success\n", id, url)
	c <- DownsampleResult{id, url, err}
	return
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
// fetch post urls ids - Given a query, fetch the database for posts' urls and ids.
//
//////////////////////////////////////////////////////////////////////////////
func fetchPostIds2Urls(query string) (ids2urls map[int]UrlStatus) { //(urls []string, ids []int){
	pr("fetchPostUrlIds")

	ids2urls = make(map[int]UrlStatus)

	rows := DbQuery(query)

	defer rows.Close()
	for rows.Next() {
		id        := -1
		urlStatus := UrlStatus{}

		err := rows.Scan(&id, &urlStatus.url, &urlStatus.status)
		check(err)

		ids2urls[id] = urlStatus
	}
	check(rows.Err())

	prVal("ids2urls", ids2urls)
	prVal("Num Post Urls Fetched", len(ids2urls))
	return
}

//////////////////////////////////////////////////////////////////////////////
//
// image service - Continually checks for new images to shrink.  Images must be shrunk
//				  to thumbnail size for faster webpage loads.
//
//////////////////////////////////////////////////////////////////////////////
func ImageService() {
	if flags.mode == "fetchNewsSourceIcons" {
		for newsSource, imageUrl := range newsSourceIcons {
			check(downsampleImage(imageUrl, "newsSourceIcons", newsSource, "png", 16, 16))
		}
		return
	}

	// TODO!: Process image thumbnail UrlToImage from LinkUrl submission.
	//		  Require the input not blank and database not blank, so the thumbnail link is always good.
	//		  Give user option to use Mozilla Head.
	//        If there's a problem with the UrlToImage or it's NULL, or the image doesn't downsample
	//		  for some reason, falls back on scraping the page.

	fetchImagesToDownsampleQuery := [NUM_GEN_THUMBS_PASSES]string {
		`SELECT Id, UrlToImage, ThumbnailStatus
		 FROM $$LinkPost
		 WHERE ThumbnailStatus < %d
		   AND UrlToImage <> ''
		   AND Created > now() - interval '2 weeks'
		 ORDER BY Created DESC
		 LIMIT %d;`,

		`SELECT Id, UrlToImage, ThumbnailStatus
		 FROM $$NewsPost
		 WHERE ThumbnailStatus < %d
		   AND UrlToImage <> ''
		   AND Created > now() - interval '2 weeks'
		 ORDER BY COALESCE(PublishedAt, Created) DESC
		 LIMIT %d;`,
	}
	for i := 0; i < NUM_GEN_THUMBS_PASSES; i++ {
		fetchImagesToDownsampleQuery[i] = fmt.Sprintf(
			fetchImagesToDownsampleQuery[i],
			image_DownsampleVersionTarget,
			kImageBatchSize)
	}
	prVal("fetchImagesToDownsampleQuery[0]", fetchImagesToDownsampleQuery[0])
	prVal("fetchImagesToDownsampleQuery[1]", fetchImagesToDownsampleQuery[1])


	pr("========================================")
	pr("======== STARTING IMAGE SERVICE ========")
	pr("========================================\n")

	for { // Infinite loop
		numImageProcessAttempts := 0

		// Downsample news images
		for pass := 0; pass < NUM_GEN_THUMBS_PASSES; pass++ {
			pr("========================================")
			prf("======= FETCHING IMAGES PASS: %d =======", pass)
			pr("========================================\n")


			// Grab a batch of images to downsample from new news posts.
			ids2urls := fetchPostIds2Urls(fetchImagesToDownsampleQuery[pass])

			// Download and downsample the images in parallel.
			c := make(chan DownsampleResult)
			timeout := time.After(30 * time.Second)

			for id, urlStatus := range ids2urls {
				numImageProcessAttempts++

				prf("trying to create channel to downsample id %d url %s status %d", id, urlStatus.url, urlStatus.status)
				go downsamplePostImage(urlStatus.url, urlStatus.status, id, c)
			}

			// TODO: Generalize this code.  Can use fn callbacks for the main and timeout cases.
			downsampleImagesLoop: for {
				select {
					case downsampleResult := <-c:
						newThumbnailStatus := ternary_int(
							downsampleResult.err == nil,
							image_DownsampleVersionTarget,
							image_DownsampleError)

						switch pass {
							//case genThumbPass_PollPost:
							//	DbExec(
							//		`UPDATE $$PollPost
							//		 SET ThumbnailStatus = $1
							//		 WHERE Id = $2::bigint`,
							//		newThumbnailStatus,
							//		downsampleResult.postId)
							case genThumbPass_LinkPost:
								DbExec(
									`UPDATE $$LinkPost
									 SET ThumbnailStatus = $1
									 WHERE Id = $2::bigint`,
									newThumbnailStatus,
									downsampleResult.postId)
							case genThumbPass_NewsPost:
								DbExec(
									`UPDATE $$NewsPost
									 SET ThumbnailStatus = $1
									 WHERE Id = $2::bigint`,
									newThumbnailStatus,
									downsampleResult.postId)
							default:
								assert(false)
						}

						// Remove this from the list of ids, so we can tell which ids were never processed.
						delete(ids2urls, downsampleResult.postId)

						if len(ids2urls) == 0 {
							pr("Processed all images!")
							break downsampleImagesLoop
						}
					case <- timeout:
						pr("Timeout!")

						// Set status to -1 for any images that timed out.
						for id, urlStatus := range ids2urls {
							prf("Removing timed out id %d url %s prevStatus %d", id, urlStatus.url, urlStatus.status)

							switch pass {
								//case genThumbPass_PollPost:
								//	DbExec(
								//		`UPDATE $$PollPost
								//		 SET ThumbnailStatus = -1
								//		 WHERE Id = $1::bigint`,
								//		id)
								case genThumbPass_LinkPost:
									DbExec(
										`UPDATE $$LinkPost
										 SET ThumbnailStatus = -1
										 WHERE Id = $1::bigint`,
										id)
								case genThumbPass_NewsPost:
									DbExec(
										`UPDATE $$NewsPost
										 SET ThumbnailStatus = -1
										 WHERE Id = $1::bigint`,
										id)
								default:
									assert(false)
							}
						}

						break downsampleImagesLoop
				}
			}

			DbTrackOpenConnections()
		}

		// Sleep when there are no records to process.
		if numImageProcessAttempts == 0 {
			time.Sleep(10 * time.Second)
		}
	}
}





/* DEAD SCRATCH CODE:

// Figure out which thumbnail to use based on the Url of the link submitted.
// Return the string of the image url if it exists, or "" if there is an error.
func scrapeWebpageForBestImage(pageUrl string) ([]ImageSizeResult, error) {
	prVal("scrapeWebpageForBestImage pageUrl", pageUrl)

    // Fix the URL scheme
    if !strings.HasPrefix(pageUrl, "http://") && !strings.HasPrefix(pageUrl, "https://") {
       pageUrl = "http://" + pageUrl

       prVal("fixed linkUrl", pageUrl)
	}

    // Make HTTP request
    pr("Making HTTP Response")
    response, err := httpGet(pageUrl, 30.0)

    prVal("response", response)
    prVal("response.Body", response.Body)

    defer response.Body.Close()
    if err != nil {
        prVal("HTTP request failed", err)
        return []ImageSizeResult{}, err
    }

    // Create a goquery document from the HTTP response
    document, err := goquery.NewDocumentFromReader(response.Body)
    if err != nil {
        prVal("Error loading body. ", err)
        return []ImageSizeResult{}, err
    }

    prVal("document", document)

    // Find and return all image URLs
    // Which image is the right one?
    // Excellent article!: https://tech.shareaholic.com/2012/11/02/how-to-find-the-image-that-best-respresents-a-web-page/

    // Look for the meta og:image tag, which indicates this is the image this website wants for its thumbnail!
    ogImage := ""
	document.Find("meta").Each(func(i int, s *goquery.Selection) {
	    property, _ := s.Attr("property")

	    if property == "og:image" {

	        ogImage, _ = s.Attr("content")

	        ogImage, _ = makeUrlAbsolute(ogImage, pageUrl)

	        return // continue
	    }
	})
	if ogImage != "" {
		prVal("ogImage Found!", ogImage)
		return []ImageSizeResult{ImageSizeResult{imgSrc:ogImage}}, nil
	}

	// If ogimage wan't found, we need to scrape all images, download them all, and pick the best (largest) one!
	//var images []string
	images := map[string]int{}
	document.Find("img").Each(func(index int, element *goquery.Selection) {
		imgSrc, exists := element.Attr("src")

		if !exists {
			return // continue - returns us from the lambda fn, so this basically a continue
		}

		imgUrl, err := url.Parse(imgSrc)
		if err != nil {
			prf("Error parsing URL: %s %v %s", imgSrc, imgUrl, err)
			return // continue
		}

		imgSrc, err = makeUrlAbsolute(imgSrc, pageUrl)

		if err != nil {
			images[imgSrc] = 1
		}
	})

	prVal("images", images)
	//bestImage := "" // default to the placeholder image

	if len(images) == 0 {
		return []ImageSizeResult{}, errors.New("No images found on website")
	}

	start := time.Now()

	// Get the sizes of the images, pick the best one with a size-based heuristic, multithreaded.
	c := make(chan ImageSizeResult)

	for imgSrc, _ := range(images) {
		go goDownloadImageSize(imgSrc, c)
	}

	timeout := time.After(30 * time.Second)

	numImages := len(images)

	imageSortResults := make([]ImageSizeResult, 0)

	downsampleImagesLoop: for {
		select {
			case imageSizeResult := <-c:
				imageSortResults = append(imageSortResults, imageSizeResult)

				numImages--

				if numImages == 0 {
					pr("Processed all images!")
					break downsampleImagesLoop
				}
			case <- timeout:
				pr("Timeout!")

				break downsampleImagesLoop
		}
	}

	sort.Slice(imageSortResults, func(i, j int) bool { return imageSortResults[i].imageQuality <
	                                                          imageSortResults[j].imageQuality })

	prf("With finding best image took %s", time.Since(start))

	return imageSortResults, nil
}*/
