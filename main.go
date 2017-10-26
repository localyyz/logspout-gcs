package main

import (
	"os"

	"github.com/gliderlabs/logspout/router"
)

// NOTE: this main() function is not included in the compiled Docker
// container application. The program below is just a test program to
// verify the code in modules.go which gets packaged with gliderlabs/logspout.
func main() {
	os.Setenv("GCS_KEY_FILE", "Localyyz-22c47178a7ac.json")

	// --

	bucketID := "localyyz"
	address := bucketID

	_, err := NewGCSAdapter(&router.Route{
		Adapter: "gcs",
		Address: address,
		Options: map[string]string{
			"path": "",
		},
	})
	if err != nil {
		panic(err)
	}

	for {
	}
}
