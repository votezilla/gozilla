package main

import (
    "fmt"
    "io"
    "log"
    "path/filepath"
    "os"
    "strconv"
)

func DownloadImage(imageUrl string, outputNameId int) {
    resp, err := httpGet(imageUrl, 60.0)
    check(err)
    defer resp.Body.Close()

    //open a file for writing
    file, err := os.Create("./static/downloads/" + strconv.Itoa(outputNameId) + filepath.Ext(imageUrl))
    if err != nil {
        log.Fatal(err)
    }
    // Use io.Copy to just dump the response body to the file. This supports huge files
    _, err = io.Copy(file, resp.Body)
    if err != nil {
        log.Fatal(err)
    }
    file.Close()
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