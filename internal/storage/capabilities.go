package storage

type Capabilities struct {
	Streaming       bool
	ResumableUpload bool
	CopyObject      bool
	Metadata        bool
	RangeRequests   bool
	Versioning      bool
	PresignedURLs   bool
}
