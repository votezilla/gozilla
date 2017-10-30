package main

import (
    "fmt"
//    "io"
    "io/ioutil"
 //   "log"
 //   "path/filepath"
 //   "os"
    "strconv"
    
    "willnorris.com/go/imageproxy"
    //"github.com/votezilla/gforms"
)

// Download image from imageUrl, use outputNameId to form name before extension, extension stays the same.
func DownloadImage(imageUrl string, outputNameId int) {
    resp, err := httpGet(imageUrl, 60.0)
    check(err)
    defer resp.Body.Close()

/*    //open a file for writing
    file, err := os.Create("./static/downloads/" + strconv.Itoa(outputNameId) + filepath.Ext(imageUrl))
    if err != nil {
        log.Fatal(err)
    }
    // Use io.Copy to just dump the response body to the file. This supports huge files
    _, err = io.Copy(file, resp.Body)
    check(err)
    file.Close()
    fmt.Println("Success!")
*/
    // hack test downsample
	//open a file for writing
	//file, err := os.Create("./static/downloads/" + strconv.Itoa(outputNameId) + "downsample.jpeg")
	//if err != nil {
	//	log.Fatal(err)
	//}
	
	bytes, err := ioutil.ReadAll(resp.Body)
	
	downsampledImg, err := imageproxy.Transform(
		bytes,
		imageproxy.Options{
			Width:		125,
			Height:		75,
			Format:		"jpeg",
			SmartCrop:	true,
		},
	)
	
	// Use io.Copy to just dump the response body to the file. This supports huge files
	//_, err = io.Copy(file, downsampledImg)
	err = ioutil.WriteFile(
		"./static/downloads/" + strconv.Itoa(outputNameId) + "downsample.jpeg",
		downsampledImg,
		0644,
	)
	check(err)
//	file.Close()
	fmt.Println("Success!")   
}
/*
imageproxy.Transform(img []byte, opt Options) ([]byte, error) {

type Options struct {
	// See ParseOptions for interpretation of Width and Height values
	Width  float64
	Height float64

	// If true, resize the image to fit in the specified dimensions.  Image
	// will not be cropped, and aspect ratio will be maintained.
	Fit bool

	// Rotate image the specified degrees counter-clockwise.  Valid values
	// are 90, 180, 270.
	Rotate int

	FlipVertical   bool
	FlipHorizontal bool

	// Quality of output image
	Quality int

	// HMAC Signature for signed requests.
	Signature string

	// Allow image to scale beyond its original dimensions.  This value
	// will always be overwritten by the value of Proxy.ScaleUp.
	ScaleUp bool

	// Desired image format. Valid values are "jpeg", "png", "tiff".
	Format string

	// Crop rectangle params
	CropX      float64
	CropY      float64
	CropWidth  float64
	CropHeight float64

	// Automatically find good crop points based on image content.
	SmartCrop bool
}
*/
	
//////////////////////////////////////////////////////////////////////////////
//
// image server - Continually checks for new images to shrink.  Images must be shrunk
//				  to thumbnail size for faster webpage loads.
//
//////////////////////////////////////////////////////////////////////////////
func ImageServer() {
	if flags.test != "" {
		//DownloadImage("http://i.imgur.com/m1UIjW1.jpg", 999999)
		DownloadImage("https://octodex.github.com/images/codercat.jpg", 222222)
	}

	/*
		username := ""
		if userId != -1 {
			rows := DbQuery("SELECT Username FROM votezilla.User WHERE Id = $1;", userId)
			defer rows.Close()
			if rows.Next() {
				err := rows.Scan(&username)
				check(err)	
			}
			check(rows.Err())
		}
	*/
}