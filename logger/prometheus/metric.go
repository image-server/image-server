package prometheus

// Metric represents a prometheus metric
type Metric struct {
	Name        string
	Description string
	Type        string
	Args        []string
}

var imagePosted = &Metric{
	Name:        "new_image_request",
	Description: "Number of requested images",
	Type:        "counter_vec",
}

var imagePostingFailed = &Metric{
	Name:        "new_image_request_failed",
	Description: "Number of failed requested images",
	Type:        "counter_vec",
}

var imageProcessed = &Metric{
	Name:        "processing_version_ok",
	Description: "Number of processed images",
	Type:        "counter_vec",
	Args:        []string{"ic_format"},
}

var imageAlreadyProcessed = &Metric{
	Name:        "processing_version_noop",
	Description: "Number of already processed images",
	Type:        "counter_vec",
	Args:        []string{"ic_format"},
}

var imageProcessedWithErrors = &Metric{
	Name:        "processing_version_failed",
	Description: "Number of failed processed images",
	Type:        "counter_vec",
	Args:        []string{"ic_format"},
}

var allImagesAlreadyProcessed = &Metric{
	Name:        "processing_versions_noop",
	Description: "Number of already processed all images",
	Type:        "counter_vec",
	Args:        []string{"namespace", "hash", "source_url"},
}

var sourceDownloaded = &Metric{
	Name:        "fetch_source_downloaded",
	Description: "Number of downloaded source images",
	Type:        "counter_vec",
}

var originalDownloaded = &Metric{
	Name:        "fetch_original_downloaded",
	Description: "Number of downloaded original images",
	Type:        "counter_vec",
	Args:        []string{"source", "destination"},
}

var originalDownloadFailed = &Metric{
	Name:        "fetch_original_unavailable",
	Description: "Number of unavailable downloaded original images",
	Type:        "counter_vec",
	Args:        []string{"source"},
}

var originalDownloadSkipped = &Metric{
	Name:        "fetch_original_download_skipped",
	Description: "Number of skipped downloaded original images",
	Type:        "counter_vec",
	Args:        []string{"source"},
}
