package main

import (
	"log"
	"net/http"
)

func main() {
	// 1.) This handler serves the root page html and .js content
	http.Handle("/", http.FileServer(http.Dir(".")))

	// 2.) This handler serves only the karaoke content
	// To serve a directory on disk (/tmp) under an alternate URL
	// path (/tmpfiles/), use StripPrefix to modify the request
	// URL's path before the FileServer sees it:
	http.Handle("/karaoke/", http.StripPrefix("/karaoke/", http.FileServer(http.Dir("./karaoke/"))))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
