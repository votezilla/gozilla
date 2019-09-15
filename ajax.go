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
	pr(go_, "ajaxVoteHandler")
	prVal(go_, "r.Method", r.Method)
	
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
    
    //parse request to struct
    var vote struct {
		PostId	int
		UserId	int
		Add		bool
		Up		bool
	}
	
    err := json.NewDecoder(r.Body).Decode(&vote)
    if err != nil {
		prVal(go_, "Failed to decode json body", r.Body)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    prVal(go_, "vote", vote)
	
	if vote.Add {
    	DbExec( // sprintf necessary cause ::bool produces incorrect value in driver.
			`INSERT INTO $$PostVote(PostId, UserId, Up)
			 VALUES ($1::bigint, $2::bigint, $3::bool)
			 ON CONFLICT (PostId, UserId) DO UPDATE 
			 SET Up = $3::bool;`,
			vote.PostId,
			vote.UserId,
			vote.Up)
	} else { // remove
		DbExec(
			`DELETE FROM $$PostVote 
			 WHERE PostId = $1::bigint AND UserId = $2::bigint;`,
			vote.PostId,
			vote.UserId)
	}
    
    // create json response from struct
    a, err := json.Marshal(vote)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Write(a)
}

// Figure out which thumbnail to use based on the Url of the link submitted.
func ajaxScrapeImageURLs(w http.ResponseWriter, r *http.Request) {
	pr(go_, "ajaxScrapeImageURLs")
	prVal(go_, "r.Method", r.Method)
	prVal(go_, "r", r)
	if r.Method != "POST" {
		prVal(go_, "r.Method must is not POST", r.Method)
		return
	}
	
    //parse request to struct
    var linkUrl struct {
		Url		string		
	}
	
	prVal(go_, "r.Body", r.Body)
	
    err := json.NewDecoder(r.Body).Decode(&linkUrl)
    if err != nil {
		prVal(go_, "Failed to decode json body", r.Body)
        return
    }
    
    prVal(go_, "linkUrl", linkUrl)
    prVal(go_, "linkUrl", linkUrl.Url)
    
    // Fix the URL scheme
    if !strings.HasPrefix(linkUrl.Url, "http://") && !strings.HasPrefix(linkUrl.Url, "https://") {
       linkUrl.Url = "http://" + linkUrl.Url
       
       prVal(go_, "fixed linkUrl", linkUrl.Url)
	}

    // Make HTTP request
    //response, err := httpGet_Old(linkUrl.Url, 60.0)
    response, err := httpGet_Old(linkUrl.Url, 60.0)
    if err != nil {
        prVal(go_, "HTTP request failed", err) 
        return
    }
    defer response.Body.Close()
    
    prVal(go_, "response", response)

    // Create a goquery document from the HTTP response
    document, err := goquery.NewDocumentFromReader(response.Body)
    if err != nil {
        prVal(go_, "Error loading LinkUrl body. ", err)
        return
    }
    
    prVal(go_, "document", document)

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
	        
	        prVal(go_, "ogImage Found!", ogImage)
	        
	        parsedImages.OGImages = append(parsedImages.Images, ogImage)
	        prVal(go_, "parsedImages.Images scanned", parsedImages.Images)
	    }
	})
	
	// Allow this code to continue, so user has more thumbnail options
	//if ogImage == "" {
	if true {
		//parsedImages.Images
		
		dupChecker := make(map[string]bool)
		
		document.Find("img").Each(func(index int, element *goquery.Selection) {
			imgSrc, exists := element.Attr("src")
			
			prVal(go_, "imgSrc", imgSrc)
			
			if exists {
				imgSrc, err := makeUrlAbsolute(imgSrc, linkUrl.Url)
				
				prVal(go_, "  absolutePath", imgSrc)

				if err != nil {
					prVal(go_, "error", err)
				} else if !dupChecker[imgSrc] {
					parsedImages.Images = append(parsedImages.Images, imgSrc)
					dupChecker[imgSrc] = true
					prVal(go_, "parsedImages.Images scanned", parsedImages.Images)
				}
			}
		})
		
		prVal(go_, "parsedImages.Images scanned", parsedImages.Images)
		
		

		// Sort in descending order by image quality (based on size of the image)
		//sort.Slice(parsedImages.Images, func(i, j int) bool { return parsedImages.Images[i] > parsedImages.Images[j] })
		sort.Strings(parsedImages.Images)
		
		//prVal(go_, "parsedImages.Images sorted", parsedImages.Images)
		
		for i := 0; i < len(parsedImages.Images); i++ {
			prf(go_, "parsedImages.Images[%d]: %s", i, parsedImages.Images[i])
		}
/*		
		// Remove duplicates
		imagesNoDups := []string{parsedImages.Images[0]}
		prVal(go_, "len(parsedImages.Images)", len(parsedImages.Images))
		
		for i := 1; i < len(parsedImages.Images); i++ {
			prf(go_, "Comparing i(%d) to i-1(%d)", i, i-1)
			prVal(go_, "imagesNoDups[i]", parsedImages.Images[i])
		
			if parsedImages.Images[i] != parsedImages.Images[i-1] {
				imagesNoDups = append(imagesNoDups, imagesNoDups[i])
			}
		}
		parsedImages.Images = imagesNoDups
																  
		prVal(go_, "parsedImages.Images removedDups", parsedImages.Images)
		
		
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
						pr(is_, "Processed all images!")
						break downsampleImagesLoop
					}
				case <- timeout:
					pr(is_, "Timeout!")

					break downsampleImagesLoop
			}		
		}

		// Sort in descending order by image quality (based on size of the image)
		sort.Slice(imageSortResults, func(i, j int) bool { return imageSortResults[i].imageQuality > 
																  imageSortResults[j].imageQuality })
																  
		prVal(go_, "imageSortResults 222", imageSortResults)
		
		parsedImages.Images = parsedImages.Images[:0] // Set the slice lenght to 0 while keeping the allocated memory
		for _, imageSortResult := range(imageSortResults) {
			parsedImages.Images = append(parsedImages.Images, imageSortResult.imgSrc)			
		}
		
		prVal(go_, "parsedImages.Images", parsedImages.Images)
*/		
	}
	
    // create json response from struct
    a, err := json.Marshal(parsedImages)
    if err != nil {
		prVal(go_, "Unable to marshall images for ", parsedImages)
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    w.Write(a)
}
