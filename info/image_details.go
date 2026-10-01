package info

import (
	"encoding/json"
	"io/ioutil"
	"log"
)

type ImageProperties struct {
	Hash        string `json:"hash"`
	Height      int    `json:"height"`
	Width       int    `json:"width"`
	ContentType string `json:"content_type"`
	// Orientation is the EXIF orientation (1-8); 0 or 1 means upright. Not
	// saved: Width and Height already report the upright size.
	Orientation int `json:"-"`
}

// ImageDetailsToJSON returns a string with JSON representation of the ImageDetails
func ImageDetailsToJSON(d *ImageProperties) (string, error) {
	b, err := json.Marshal(d)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// SaveImageDetail saves ImageDetails in a JSON file on "path"
func SaveImageDetail(d *ImageProperties, path string) error {
	json, err := ImageDetailsToJSON(d)
	if err != nil {
		log.Println(err)
		return err
	}

	d1 := []byte(json)
	err = ioutil.WriteFile(path, d1, 0644)

	if err != nil {
		log.Println(err)
	}
	return err
}
