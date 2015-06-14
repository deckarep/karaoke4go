package main

import (
	"log"
	"net/http"
)

func main() {
	http.Handle("/", http.FileServer(http.Dir(".")))

	// To serve a directory on disk (/tmp) under an alternate URL
	// path (/tmpfiles/), use StripPrefix to modify the request
	// URL's path before the FileServer sees it:
	// This handler serves only the karaoke content
	http.Handle("/karaoke/", http.StripPrefix("/karaoke/", http.FileServer(http.Dir("./karaoke/"))))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
