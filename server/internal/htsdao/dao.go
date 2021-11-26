package htsdao

import "github.com/ga4gh/htsget-refserver/internal/htsrequest"
import "github.com/ga4gh/htsget-refserver/internal/htsticket"

type DataAccessObject interface {
	GetContentLength() int64

	// GetHeaderByteRangeUrl return the entire header as a single URL
	GetHeaderByteRangeUrl() *htsticket.URL

	// GetByteRangeUrls return the entire file content (not header) as a series of URLs
	GetByteRangeUrls() []*htsticket.URL

	// GetChunkedInPlaceBlocks return each specified region as a byte range URL
    GetChunkedInPlaceBlocks(regions []*htsrequest.Region) []*htsticket.URL

	GetBgzipEof() *htsticket.URL

	String() string
}
