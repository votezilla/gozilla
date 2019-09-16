package main

import (
	"errors"
	"fmt"
	"github.com/puerkitobio/goquery"  
	"github.com/rubenfonseca/fastimage"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"sort"
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
	image_DownsampleError	= -1
	
	genThumbsPass_ScrapeUserPostImage = 0
	genThumbsPass_DownsampleNewsImage = 1
	NUM_GEN_THUMBS_PASSES             = 2
	
	kImageBatchSize = 5		// Number of images to convert to thumbnails per batch
)

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
func downloadImage(imageUrl string, outputNameId int) error {
    resp, err := httpGet(imageUrl, 10.0)
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

// Downloads just enough of the image (from the web) to determine its width and height.
func downloadImageSize(imageUrl string) (int, int, error) {
	prVal(is_, "downloadImageSize", imageUrl)
	_, size, err := fastimage.DetectImageType(imageUrl)
	prVal(is_, "  size", size)
	if err != nil {
		return -1, -1, err
	}	
	if size == nil {
		pr(is_, "  size is nil")
		return -1, -1, errors.New("downloadImageSize gets nil size and must abort")
	}
	return int(size.Width), int(size.Height), nil
}
		
// Download image from imageUrl, use outputName to form name before extension, extension stays the same.
func downsampleImage(imageUrl string, directory string, outputName string, extension string, width int, height int) error {
	prf(is_, "downsampleImage %s -> %s.%s", imageUrl, outputName, extension)
	
	resp, err := httpGet(imageUrl, 10.0)
    if err != nil {
		prVal(is_, "  ERR 1", err)
		return err
	}
    defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
    if err != nil {
		prVal(is_, "  ERR 2", err)
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
		prVal(is_, "  ERR 3", err)
		return err
	}
	
	err = ioutil.WriteFile(
		"./static/" + directory + "/" + outputName + "." + extension,
		downsampledImg,
		0644,
	)
	
	if err != nil {
		prVal(is_, "  ERR 4", err)
	} else {
		pr(is_, "Success downsampling image!")
	}
	return nil
}

func goDownloadImageSize(imgSrc string, c chan ImageSizeResult) {
	prf(is_, "calling gorouting downloadImageSize(%s)", imgSrc)

	width, height, err := downloadImageSize(imgSrc) 

	//prf(is_, "   the result is %d, %d, %s", width, height, err)
	
	minDim := min_int(width, height)
	maxDim := max_int(width, height)
	imageQuality := minDim * minDim * maxDim // Rewards both the minimum dimension (to discourage banners) while also encouraging a larger area
	
	prf(is_, "minDim: %d maxDim: %d imageQuality %d imgSrc: %s",
			minDim, maxDim, imageQuality, imgSrc)

	c <- ImageSizeResult{imgSrc, width, height, imageQuality, err}
} 


// If imgSrc is a relative URL, converts it to an absolute URL (using baseUrl).  Returns the result, or an error if unsuccessful.
func makeUrlAbsolute(imgSrc, baseUrl string) (string, error) {
	
	imgUrl, err := url.Parse(imgSrc)
	
	if err != nil {
	//	prf(go_, "Error parsing URL: %s %v %s", imgSrc, imgUrl, err)
		return "", err
	}

	if !imgUrl.IsAbs() {
	//	pr(go_, "Image URL is not absolute")

		baseUrl, err := url.Parse(baseUrl)
		if err != nil {
	//		prf(go_, "Error parsing base URL: %s %s", linkUrl.Url, err)
			return "", err
		}

		imgUrl := baseUrl.ResolveReference(imgUrl)

		//prVal(go_, "Fixed Image Url:", imgUrl)

		imgSrc = imgUrl.String()

		//prVal(go_, "Fixed imgSrc:", imgSrc)
	}

	return imgSrc, nil
}

// Figure out which thumbnail to use based on the Url of the link submitted.
// Return the string of the image url if it exists, or "" if there is an error.
func scrapeWebpageForBestImage(pageUrl string) ([]ImageSizeResult, error) {
	prVal(is_, "scrapeWebpageForBestImage pageUrl", pageUrl)
	
    // Fix the URL scheme
    if !strings.HasPrefix(pageUrl, "http://") && !strings.HasPrefix(pageUrl, "https://") {
       pageUrl = "http://" + pageUrl
       
       prVal(is_, "fixed linkUrl", pageUrl)
	}

    // Make HTTP request
    pr(is_, "Making HTTP Response")
    response, err := httpGet(pageUrl, 30.0)
    
    prVal(is_, "response", response)
    prVal(is_, "response.Body", response.Body)
    
    defer response.Body.Close()
    if err != nil {
        prVal(is_, "HTTP request failed", err) 
        return []ImageSizeResult{}, err
    }

    // Create a goquery document from the HTTP response
    document, err := goquery.NewDocumentFromReader(response.Body)
    if err != nil {
        prVal(is_, "Error loading body. ", err)
        return []ImageSizeResult{}, err
    }
    
    prVal(is_, "document", document)

    // Find and return all image URLs
    // Which image is the right one?
    // Excellent article!: https://tech.shareaholic.com/2012/11/02/how-to-find-the-image-that-best-respresents-a-web-page/
    
    // Look for the meta og:image tag, which indicates this is the image this website wants for its thumbnail!
    ogImage := ""
	document.Find("meta").Each(func(i int, s *goquery.Selection) {
	    property, _ := s.Attr("property"); 
	    
	    if property == "og:image" {	
			
	        ogImage, _ = s.Attr("content")
	        
	        ogImage, _ = makeUrlAbsolute(ogImage, pageUrl)
				        
	        return // continue
	    }
	})
	if ogImage != "" {
		prVal(is_, "ogImage Found!", ogImage)
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
			prf(is_, "Error parsing URL: %s %v %s", imgSrc, imgUrl, err)
			return // continue
		}

		imgSrc, err = makeUrlAbsolute(imgSrc, pageUrl)
		
		if err != nil {
			images[imgSrc] = 1
		}
	})
	
	prVal(is_, "images", images)
	//bestImage := "" // default to the placeholder image	
	
	if len(images) == 0 {
		return []ImageSizeResult{}, errors.New("No images found on website")
	}
		
	start := time.Now()
		
	// Get the sizes of the images, pick the best one with a size-based heuristic, multithreaded.	
	c := make(chan ImageSizeResult)

	//for _, imgSrc := range(images) {
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
					pr(is_, "Processed all images!")
					break downsampleImagesLoop
				}
			case <- timeout:
				pr(is_, "Timeout!")

				break downsampleImagesLoop
		}		
	}
	
	sort.Slice(imageSortResults, func(i, j int) bool { return imageSortResults[i].imageQuality < 
	                                                          imageSortResults[j].imageQuality })

	prf(is_, "With finding best image took %s", time.Since(start))	
	
	return imageSortResults, nil
}

// Downsample an image asynchronously, return infomation about id and error status to the channel after.
func downsamplePostImage(url string, id int, pass int, c chan DownsampleResult) {
	prf(is_, "Downsampling image #%d pass %d urls %s\n", id, pass, url)
	
	//	genThumbsPass_ScrapeUserPostImage = 0
	//	genThumbsPass_DownsampleNewsImage = 1
	//	NUM_GEN_THUMBS_PASSES             = 2
	var err error
	if pass == genThumbsPass_ScrapeUserPostImage {
		// Scrape website url, pick the best image, assign its url to url.
		prVal(is_, "pass == genThumbsPass_ScrapeUserPostImage url", url)
		images, err := scrapeWebpageForBestImage(url)
		
		prVal(is_, "imgUrl becomes", url)
		
		// If there's an error, must use the placeholder thumbnail.  Returning an error will automatically trigger this.
		if err != nil {
			prVal(is_, "downsamplePostImage encountered some error", err)
			c <- DownsampleResult{id, "", err}
			return
		}	
		
		for len(images) > 0 {
			// x = images.Pop()
			x     := images[len(images)-1]
			images = images[:len(images)-1]

			prVal(is_,	 "x = images.Pop(), x", x)

			err = downsampleImage(x.imgSrc, "thumbnails", strconv.Itoa(id), "jpeg", 125, 75)
			if err != nil {
				// TODO: We must elegantly recover if we get an error here!
				prVal(is_, "downsamplePostImage called downsampleImage and then encountered some error... let's try the next image", err)
				continue
			}

			prf(is_, "Result for #%d image %s: %v\n", id, url, err)

			c <- DownsampleResult{id, url, err}
			return
		}
		
		pr(is_, "No images, or not images were able to be downsampled correctly for some reason.")
		c <- DownsampleResult{id, "", errors.New("No images, or not images were able to be downsampled correctly for some reason.")}
	} else { // Normal downsample
		err = downsampleImage(url, "thumbnails", strconv.Itoa(id), "jpeg", 125, 75)
		if err != nil {
			// TODO: We must elegantly recover if we get an error here!
			prVal(is_, "downsamplePostImage called downsampleImage and then encountered some error", err)
		}

		prf(is_, "Result for #%d image %s: %v\n", id, url, err)

		c <- DownsampleResult{id, url, err}
	}
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
func fetchPostIds2Urls(query string) (ids2urls map[int]string) { //(urls []string, ids []int){
	pr(is_, "fetchPostUrlIds")
	
	ids2urls = make(map[int]string)
	
	rows := DbQuery(query)

	defer rows.Close()
	for rows.Next() {
		var url string
		var id int
		
		err := rows.Scan(&url, &id)
		check(err)
		
		ids2urls[id] = url
	}
	check(rows.Err())
	
	prVal(is_, "ids2urls", ids2urls)	
	prVal(is_, "Num Post Urls Fetched", len(ids2urls))
	return
}

//////////////////////////////////////////////////////////////////////////////
//
// image server - Continually checks for new images to shrink.  Images must be shrunk
//				  to thumbnail size for faster webpage loads.
//
//////////////////////////////////////////////////////////////////////////////
func ImageServer() {
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

	queries := [NUM_GEN_THUMBS_PASSES]string {
		// genThumbsPass_ScrapeUser:
		`SELECT LinkUrl, Id
		 FROM ONLY $$LinkPost
		 WHERE ThumbnailStatus = 0
		 ORDER BY Created DESC
		 LIMIT ` + strconv.Itoa(kImageBatchSize) + ";",
		// genThumbsPass_DownsampleNewsImage
		`SELECT UrlToImage, Id 
		 FROM $$NewsPost 
		 WHERE ThumbnailStatus = 0 AND UrlToImage <> ''
		 ORDER BY COALESCE(PublishedAt, Created) DESC
		 LIMIT ` + strconv.Itoa(kImageBatchSize) + ";",
	}

	pr(is_, "========================================")
	pr(is_, "======== STARTING IMAGE SERVER =========")
	pr(is_, "========================================\n")
	
	for {
		// Downsample news images
		for pass := 0; pass < NUM_GEN_THUMBS_PASSES; pass++ { // REVERT!!!
			pr(is_, "========================================")
			prf(is_, "======= FETCHING IMAGES PASS: %d =======", pass)
			pr(is_, "========================================\n")

			
			// Grab a batch of images to downsample from new news posts.
			ids2urls := fetchPostIds2Urls(queries[pass])
			
			if len(ids2urls) == 0 { // If no URLS, wait 10 seconds and continue checking queries.
				time.Sleep(10 * time.Second)
				continue
			}

			// Download and downsample the images in parallel.
			c := make(chan DownsampleResult)
			timeout := time.After(30 * time.Second)

			for id, url := range ids2urls {
				prf(is_, "trying to create channel to downsample id %d url %s", id, url)
				go downsamplePostImage(url, id, pass, c)
			}
			
			//time.Sleep(5 * time.Minute)
			//return // DON'T CHECK IN!!!!!!!!

			// TODO: Generalize this code.  Can use fn callbacks for the main and timeout cases.
			downsampleImagesLoop: for {
				select {
					case downsampleResult := <-c: // TODO: this code can be moved to downsamplePostImage(), which then all collectively can become the callback function.
						if pass == genThumbsPass_ScrapeUserPostImage {
							DbExec(  
								`UPDATE ONLY $$LinkPost 
								 SET ThumbnailStatus = $1,
								     UrlToImage = $2
								 WHERE Id = $3::bigint`,
								ternary_int(downsampleResult.err == nil, image_Downsampled, image_DownsampleError),
								downsampleResult.urlToImage,
								downsampleResult.postId)
						} else {
							DbExec( 
								`UPDATE $$NewsPost 
								 SET ThumbnailStatus = $1
								 WHERE Id = $2::bigint`,
								ternary_int(downsampleResult.err == nil, image_Downsampled, image_DownsampleError),
								downsampleResult.postId)
						}
						// Remove this from the list of ids, so we can tell which ids were never processed.
						delete(ids2urls, downsampleResult.postId)

						if len(ids2urls) == 0 {
							pr(is_, "Processed all images!")
							break downsampleImagesLoop
						}
					case <- timeout:
						pr(is_, "Timeout!")

						// Set status to -1 for any images that timed out.
						for id, url := range ids2urls {
							prf(is_, "Removing timed out id %d url %s", id, url)
							if pass == genThumbsPass_ScrapeUserPostImage {
								DbExec(
								`UPDATE ONLY $$LinkPost 
								 SET ThumbnailStatus = -1
								 WHERE Id = $1::bigint`,
								id)
							} else {
								DbExec(   
								`UPDATE $$NewsPost 
								 SET ThumbnailStatus = -1
								 WHERE Id = $1::bigint`,
								id)
							}
						}

						break downsampleImagesLoop
				}		
			}
			
			DbTrackOpenConnections()
		}		
	}
}