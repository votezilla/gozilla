package main

import (
    "fmt"
    "io"
    "io/ioutil"
    "path/filepath"
    "os"
    "strconv"
    "willnorris.com/go/imageproxy"
)

// Download image from imageUrl, use outputNameId to form name before extension, extension stays the same.
func DownloadImage(imageUrl string, outputNameId int) {
    resp, err := httpGet(imageUrl, 60.0)
    check(err)
    defer resp.Body.Close()

    //open a file for writing
    file, err := os.Create("./static/downloads/" + strconv.Itoa(outputNameId) + filepath.Ext(imageUrl))
    check(err)
    defer file.Close()
    
    _, err = io.Copy(file, resp.Body)
    check(err)

    fmt.Println("Success!")
}
		
// Download image from imageUrl, use outputNameId to form name before extension, extension stays the same.
func DownsampleImage(imageUrl string, outputNameId int) {
	resp, err := httpGet(imageUrl, 60.0)
    check(err)
    defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	check(err)
	
	downsampledImg, err := imageproxy.Transform(
		bytes,
		imageproxy.Options{
			Width:		125,
			Height:		75,
			Format:		"jpeg",
			SmartCrop:	true,
		},
	)
	check(err)
	
	check(ioutil.WriteFile(
		"./static/downloads/" + strconv.Itoa(outputNameId) + "downsample.jpeg",
		downsampledImg,
		0644,
	))
	fmt.Println("Success!")   
}
	
//////////////////////////////////////////////////////////////////////////////
//
// image server - Continually checks for new images to shrink.  Images must be shrunk
//				  to thumbnail size for faster webpage loads.
//
//////////////////////////////////////////////////////////////////////////////
func ImageServer() {
	if flags.test != "" {
		DownloadImage("http://i.imgur.com/m1UIjW1.jpg", 999999)
		DownsampleImage("https://octodex.github.com/images/codercat.jpg", 222222)
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