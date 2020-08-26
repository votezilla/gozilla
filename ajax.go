package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/puerkitobio/goquery"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strings"
	"sort"
)

func voteUpDown(postId, userId int64, add, up, comment bool) {
	prf("voteUpdown %d %d %s %s %s", postId, userId, bool_to_str(add), bool_to_str(up), bool_to_str(comment))

    voteTable    := ternary_str(comment, "$$CommentVote", "$$PostVote")
    voteIdColumn := ternary_str(comment, "CommentId", "PostId")

	if add {
    	DbExec(
			fmt.Sprintf(
				`INSERT INTO %s(%s, UserId, Up)
				 VALUES ($1::bigint, $2::bigint, $3::bool)
				 ON CONFLICT (%s, UserId) DO UPDATE
				 SET Up = $3::bool;`,
				 voteTable,
				 voteIdColumn,
				 voteIdColumn),
			postId,
			userId,
			up)
	} else { // remove
		DbExec(
			fmt.Sprintf(
				`DELETE FROM %s
				 WHERE %s = $1::bigint AND UserId = $2::bigint;`,
				 voteTable,
				 voteIdColumn),
			postId,
			userId)
	}

/*
	DbExec(
		fmt.Sprintf(
		   `UPDATE %s
			SET voteTally = voteTally + $1
			WHERE %s = $2::bigint`,
			voteTable,
			voteIdColumn),
		vote.Add
		vote.PostId)
*/
}

///////////////////////////////////////////////////////////////////////////////
//
// AJAX Handlers
//
///////////////////////////////////////////////////////////////////////////////
func ajaxVote(w http.ResponseWriter, r *http.Request) {
	pr("ajaxVote")
	prVal("r.Method", r.Method)

	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	userId := GetSession(r);
	if userId == -1 { // Secure cookie not found.  Either session expired, or someone is hacking.
		// So go to the register page.
		pr("Must be logged in to vote.")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

    //parse request to struct
    var vote struct {
		PostId		int
		Add			bool
		Up			bool
		IsComment	bool
	}

    err := json.NewDecoder(r.Body).Decode(&vote)
    if err != nil {
		prVal("Failed to decode json body", r.Body)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    prVal("vote", vote)

    voteUpDown(int64(vote.PostId), userId, vote.Add, vote.Up, vote.IsComment)

    // create json response from struct
    a, err := json.Marshal(vote)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Write(a)
}

// Scrape a webapge for content.
func fetchUrlDoc(url string) (*goquery.Document, error) {
	pr("fetchUrlDoc")

	prVal("linkUrl", url)

	// Fix the URL scheme
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
	   url = "http://" + url

	   prVal("fixed linkUrl", url)
	}

	// Make HTTP request.
	response, err := httpGet(url, 4.5)  // It will time out at 5 seconds anyways.
	if err != nil {
		return nil, errors.New("HTTP request failed. " + err.Error())
	}
	defer response.Body.Close()

	prVal("response", response)

	// Create a goquery document from the HTTP response
	document, err := goquery.NewDocumentFromReader(response.Body)
	if err != nil {
		return nil, errors.New("Error loading LinkUrl body. " + err.Error())
    }

    return document, nil
}


func ajaxScrapeTitle(w http.ResponseWriter, r *http.Request) {
	pr("ajaxScrapeTitle")

	prVal("r", r)
	if r.Method != "POST" {
		prVal("r.Method is not POST", r.Method)
		return
	}

    // Parse request.
    var linkUrl struct {
		Url		string
	}
	prVal("r.Body", r.Body)
    err := json.NewDecoder(r.Body).Decode(&linkUrl)
    if err != nil {
		prVal("Failed to decode json body", r.Body)
        return
    }

	// Fetch document.
	prVal("linkUrl.Url", linkUrl.Url)
	document, err := fetchUrlDoc(linkUrl.Url)
	check(err)
	prVal("document", document)

	// Scan for the og:Title.
	title := ""
	document.Find("meta").Each(func(i int, s *goquery.Selection) {
	    property, _ := s.Attr("property");

	    if property == "og:title" {
	        title, _ = s.Attr("content")

	        prVal("ogTitle Found!", title)
	    }
	})

	// Create json response from struct.
	response := struct {
		Title string
	} {
		Title: title,
	}
	prVal("response", response)
    a, err := json.Marshal(response)
    if err != nil {
		serveError(w, err)
		return
    }
    w.Write(a)
}

// Figure out which thumbnail to use based on the Url of the link created.
func ajaxScrapeImageURLs(w http.ResponseWriter, r *http.Request) {
	pr("ajaxScrapeImageURLs")

	prVal("r", r)
	if r.Method != "POST" {
		prVal("r.Method is not POST", r.Method)
		return
	}

    //parse request to struct
    var linkUrl struct {
		Url		string
	}

	prVal("r.Body", r.Body)

    err := json.NewDecoder(r.Body).Decode(&linkUrl)
    if err != nil {
		prVal("Failed to decode json body", r.Body)
        return
    }

    document, err := fetchUrlDoc(linkUrl.Url)

    prVal("document", document)

    // Find and return all image URLs
    parsedImages := struct {
		Images		[]string
	}{}

	// To ensure there are no duplicates in the image list.
	dupChecker := make(map[string]bool)

	addImageNoDups := func(imgSrc string) {
		if !dupChecker[imgSrc] {
			parsedImages.Images = append(parsedImages.Images, imgSrc)
			dupChecker[imgSrc] = true
		}
	}

    // Which image is the right one?
    // Excellent article!: https://tech.shareaholic.com/2012/11/02/how-to-find-the-image-that-best-respresents-a-web-page/
    ogImage := ""

	// Look for meta ogImage tags.
	document.Find("meta").Each(func(i int, s *goquery.Selection) {
	    property, _ := s.Attr("property");

	    if property == "og:image" {
	        ogImage, _ = s.Attr("content")

	        ogImage, _ = makeUrlAbsolute(ogImage, linkUrl.Url)

	        prVal("ogImage Found!", ogImage)

	        prVal("parsedImages.Images scanned", parsedImages.Images)

	        addImageNoDups(ogImage)
	    }
	})

	// Scan all images as well, so the user has more thumbnail options.
	if true {
		document.Find("img").Each(func(index int, element *goquery.Selection) {
			imgSrc, exists := element.Attr("src")

			prVal("imgSrc", imgSrc)

			if exists {
				imgSrc, err := makeUrlAbsolute(imgSrc, linkUrl.Url)

				if err != nil {
					prVal("error", err)
				} else {
					addImageNoDups(imgSrc)
				}
			}
		})

		prVal("parsedImages.Images scanned", parsedImages.Images)

		// Sort in descending order by image quality (based on size of the image)
		//sort.Slice(parsedImages.Images, func(i, j int) bool { return parsedImages.Images[i] > parsedImages.Images[j] })
		sort.Strings(parsedImages.Images)

		//prVal("parsedImages.Images sorted", parsedImages.Images)

		//parsedImages.AllImages = append(parsedImages.OGImages, parsedImages.Images)

		prVal("parsedImages", parsedImages)
/*
		// Remove duplicates
		imagesNoDups := []string{parsedImages.Images[0]}
		prVal("len(parsedImages.Images)", len(parsedImages.Images))

		for i := 1; i < len(parsedImages.Images); i++ {
			prf("Comparing i(%d) to i-1(%d)", i, i-1)
			prVal("imagesNoDups[i]", parsedImages.Images[i])

			if parsedImages.Images[i] != parsedImages.Images[i-1] {
				imagesNoDups = append(imagesNoDups, imagesNoDups[i])
			}
		}
		parsedImages.Images = imagesNoDups

		prVal("parsedImages.Images removedDups", parsedImages.Images)


/*		// THIS CODE WORKS, IT'S JUST TOO SLOW.  COULD KEEP IT FOR THE IMAGE SERVICE.

		// Get the sizes of the images, pick the best one with a size-based heuristic, multithreaded.
		c := make(chan ImageSizeResult)

		for _, imgSrc := range(parsedImages.Images) {
			go goDownloadImageSize(imgSrc, c)
		}

		timeout := time.After(30 * time.Second)

		numImages := len(parsedImages.Images)

		imageSortResults := make([]ImageSizeResult, 0)

		for _, imgSrc := range(parsedImages.Images) {
			go goDownloadImageSize(imgSrc, c)
		}

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

		// Sort in descending order by image quality (based on size of the image)
		sort.Slice(imageSortResults, func(i, j int) bool { return imageSortResults[i].imageQuality >
																  imageSortResults[j].imageQuality })

		prVal("imageSortResults 222", imageSortResults)

		parsedImages.Images = parsedImages.Images[:0] // Set the slice lenght to 0 while keeping the allocated memory
		for _, imageSortResult := range(imageSortResults) {
			parsedImages.Images = append(parsedImages.Images, imageSortResult.imgSrc)
		}

		prVal("parsedImages.Images", parsedImages.Images)
*/
	}

	// Default image
	addImageNoDups(kDefaultImage)


    // create json response from struct
    a, err := json.Marshal(parsedImages)
    if err != nil {
		prVal("Unable to marshall images for ", parsedImages)
        serveError(w, err)
        return
    }
    w.Write(a)
}
