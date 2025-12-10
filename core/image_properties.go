package core

// ImageProperties contains metadata about an uploaded image
type ImageProperties struct {
	Hash        string `json:"hash"`
	Height      int    `json:"height"`
	Width       int    `json:"width"`
	ContentType string `json:"content_type"`
}
