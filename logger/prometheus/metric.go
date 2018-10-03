package prometheus

// Metric represents a prometheus metric
type Metric struct {
	Name        string
	Description string
	Args        []string
}

var imagePosted = &Metric{
	Name:        "image_server_new_image_request_total",
	Description: "Number of requested images",
}

var imagePostingFailed = &Metric{
	Name:        "image_server_new_image_request_failed_total",
	Description: "Number of failed requested images",
}

var imageProcessed = &Metric{
	Name:        "image_server_processing_version_ok_total",
	Description: "Number of processed images",
	Args:        []string{"ic_format"},
}

var imageAlreadyProcessed = &Metric{
	Name:        "image_server_processing_version_noop_total",
	Description: "Number of already processed images",
	Args:        []string{"ic_format"},
}

var imageProcessedWithErrors = &Metric{
	Name:        "image_server_processing_version_failed_total",
	Description: "Number of failed processed images",
	Args:        []string{"ic_format"},
}

var allImagesAlreadyProcessed = &Metric{
	Name:        "image_server_processing_versions_noop_total",
	Description: "Number of already processed all images",
	Args:        []string{"namespace"},
}

var sourceDownloaded = &Metric{
	Name:        "image_server_fetch_source_downloaded_total",
	Description: "Number of downloaded source images",
}

var originalDownloaded = &Metric{
	Name:        "image_server_fetch_original_downloaded_total",
	Description: "Number of downloaded original images",
}

var originalDownloadFailed = &Metric{
	Name:        "image_server_fetch_original_unavailable_total",
	Description: "Number of unavailable downloaded original images",
}

var originalDownloadSkipped = &Metric{
	Name:        "image_server_fetch_original_download_skipped_total",
	Description: "Number of skipped downloaded original images",
}
