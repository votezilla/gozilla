package main

import (
	"encoding/json"
	"github.com/puerkitobio/goquery"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"strings"
	"sort"
)

///////////////////////////////////////////////////////////////////////////////
//
// AJAX Handlers
//
///////////////////////////////////////////////////////////////////////////////
func ajaxVoteHandler(w http.ResponseWriter, r *http.Request) {
	pr("ajaxVoteHandler")
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
		PostId	int
		Add		bool
		Up		bool
	}

    err := json.NewDecoder(r.Body).Decode(&vote)
    if err != nil {
		prVal("Failed to decode json body", r.Body)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    prVal("vote", vote)

	if vote.Add {
    	DbExec(
			`INSERT INTO $$PostVote(PostId, UserId, Up)
			 VALUES ($1::bigint, $2::bigint, $3::bool)
			 ON CONFLICT (PostId, UserId) DO UPDATE
			 SET Up = $3::bool;`,
			vote.PostId,
			userId,
			vote.Up)
	} else { // remove
		DbExec(
			`DELETE FROM $$PostVote
			 WHERE PostId = $1::bigint AND UserId = $2::bigint;`,
			vote.PostId,
			userId)
	}

    // create json response from struct
    a, err := json.Marshal(vote)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Write(a)
}

// Figure out which thumbnail to use based on the Url of the link created.
func ajaxScrapeImageURLs(w http.ResponseWriter, r *http.Request) {
	pr("ajaxScrapeImageURLs")
	prVal("r.Method", r.Method)
	prVal("r", r)
	if r.Method != "POST" {
		prVal("r.Method must is not POST", r.Method)
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

    prVal("linkUrl", linkUrl)
    prVal("linkUrl", linkUrl.Url)

    // Fix the URL scheme
    if !strings.HasPrefix(linkUrl.Url, "http://") && !strings.HasPrefix(linkUrl.Url, "https://") {
       linkUrl.Url = "http://" + linkUrl.Url

       prVal("fixed linkUrl", linkUrl.Url)
	}

    // Make HTTP request
    //response, err := httpGet_Old(linkUrl.Url, 60.0)
    response, err := httpGet_Old(linkUrl.Url, 60.0)
    if err != nil {
        prVal("HTTP request failed", err)
        return
    }
    defer response.Body.Close()

    prVal("response", response)

    // Create a goquery document from the HTTP response
    document, err := goquery.NewDocumentFromReader(response.Body)
    if err != nil {
        prVal("Error loading LinkUrl body. ", err)
        return
    }

    prVal("document", document)

    // Find and return all image URLs
    parsedImages := struct {
		OGImages	[] string
		Images		[]string
	}{
		[]string{},
		[]string{},
	}

    // Which image is the right one?
    // Excellent article!: https://tech.shareaholic.com/2012/11/02/how-to-find-the-image-that-best-respresents-a-web-page/
    ogImage := ""

	document.Find("meta").Each(func(i int, s *goquery.Selection) {
	    property, _ := s.Attr("property");

	    if property == "og:image" {
	        ogImage, _ = s.Attr("content")

	        ogImage, _ = makeUrlAbsolute(ogImage, linkUrl.Url)

	        prVal("ogImage Found!", ogImage)

	        parsedImages.OGImages = append(parsedImages.Images, ogImage)
	        prVal("parsedImages.Images scanned", parsedImages.Images)
	    }
	})

	// Allow this code to continue, so user has more thumbnail options
	//if ogImage == "" {
	if true {
		//parsedImages.Images

		dupChecker := make(map[string]bool)

		document.Find("img").Each(func(index int, element *goquery.Selection) {
			imgSrc, exists := element.Attr("src")

			prVal("imgSrc", imgSrc)

			if exists {
				imgSrc, err := makeUrlAbsolute(imgSrc, linkUrl.Url)

				prVal("  absolutePath", imgSrc)

				if err != nil {
					prVal("error", err)
				} else if !dupChecker[imgSrc] {
					parsedImages.Images = append(parsedImages.Images, imgSrc)
					dupChecker[imgSrc] = true
					prVal("parsedImages.Images scanned", parsedImages.Images)
				}
			}
		})

		prVal("parsedImages.Images scanned", parsedImages.Images)



		// Sort in descending order by image quality (based on size of the image)
		//sort.Slice(parsedImages.Images, func(i, j int) bool { return parsedImages.Images[i] > parsedImages.Images[j] })
		sort.Strings(parsedImages.Images)

		//prVal("parsedImages.Images sorted", parsedImages.Images)

		for i := 0; i < len(parsedImages.Images); i++ {
			prf("parsedImages.Images[%d]: %s", i, parsedImages.Images[i])
		}
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


/*		// THIS CODE WORKS, IT'S JUST TOO SLOW.  COULD KEEP IT FOR THE IMAGE SERVER.

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

    // create json response from struct
    a, err := json.Marshal(parsedImages)
    if err != nil {
		prVal("Unable to marshall images for ", parsedImages)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Write(a)
}
