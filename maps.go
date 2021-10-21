package main

import (
	"net/http"
)

///////////////////////////////////////////////////////////////////////////////
//
// maps
//
///////////////////////////////////////////////////////////////////////////////
func mapsHandler(w http.ResponseWriter, r *http.Request){
	args := struct {
		PageArgs
		MapsApiKey			string
		MapsAutoComplete	bool
	}{
		PageArgs:	makePageArgs(r, "Maps", "", ""),
		MapsApiKey: flags.mapsApiKey,
		MapsAutoComplete: str_to_bool(parseUrlParam(r, "autoComplete")),
	}

	executeTemplate(w, kMaps, args)
}