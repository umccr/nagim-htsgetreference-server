package htsdao

import (
	"compress/gzip"
	"github.com/ga4gh/htsget-refserver/internal/awsutils"
	"github.com/ga4gh/htsget-refserver/internal/bgzf"
	log "github.com/ga4gh/htsget-refserver/internal/htslog"
	"github.com/ga4gh/htsget-refserver/internal/htsrequest"
	"github.com/ga4gh/htsget-refserver/internal/htsticket"
	"github.com/ga4gh/htsget-refserver/internal/tabix"
	"sort"
	"time"
)

// Index unifies CSI and tabix.
type Index interface {
	Chunks(string, int, int) ([]bgzf.Chunk, error)
	NameColumn() int
	BeginColumn() int
	EndColumn() int
	ZeroBased() bool
	MetaChar() rune
	Skip() int
}

type tIndex struct{ *tabix.Index }

func (t tIndex) NameColumn() int {
	return int(t.Index.NameColumn)
}
func (t tIndex) BeginColumn() int {
	return int(t.Index.BeginColumn)
}

func (t tIndex) EndColumn() int {
	return int(t.Index.EndColumn)
}

func (t tIndex) ZeroBased() bool {
	return t.Index.ZeroBased
}

func (t tIndex) MetaChar() rune {
	return t.Index.MetaChar
}

func (t tIndex) Skip() int {
	return int(t.Index.Skip)
}

type AWSDao struct {
	id  string
	url string
}

func NewAWSDao(id string, url string) *AWSDao {
	dao := new(AWSDao)
	dao.id = id
	dao.url = url

	log.Debug("Creating AWSDAo for %s, %s", id, url)
	return dao
}

func (dao *AWSDao) GetContentLength() int64 {
	contentLength, err := awsutils.HeadS3Object(awsutils.S3Dto{
		ObjPath: dao.url,
	})
	log.Debug("S3 Object %s has content length %d", dao.url, contentLength)
	if err != nil {
		log.Error("GetContentLength: %v", err)
		return 0
	} else {
		return contentLength
	}
}

func (dao *AWSDao) GetHeaderByteRangeUrl() *htsticket.URL {

	indexBodyReader, err := awsutils.GetS3Object(awsutils.S3Dto{
		ObjPath: dao.url + ".tbi",
	})

	gz, err := gzip.NewReader(indexBodyReader)

	if err != nil {
		log.Error("GetHeaderByteRangeUrl: %v", err)
		return nil
	}

	defer gz.Close()

	t, err := tabix.ReadFrom(gz)
	if err != nil {
		log.Error("GetHeaderByteRangeUrl: %v", err)
		return nil
	}

	for _, name := range t.Names() {
		chunks, _ := t.Chunks(name, 0, 1000000000)

		for _, chunk := range chunks {
			return MakeHeaderUrl(dao, chunk)
		}

		return nil
	}

	return nil
}

func (dao *AWSDao) GetBgzipEof() *htsticket.URL {

	// TODO: this should come from a config setting.. or we should teach the server htsget inlining
	req, _ := awsutils.PresignGetObject(awsutils.S3Dto{
		ObjPath: "s3://umccr-10g-data-dev/bgzip-eof.bin",
	})

	return htsticket.NewURL().
		SetURL(req).
		SetClassBody()
}


// GetByteRangeUrls return the content of this file as a set of 'block' URLs
func (dao *AWSDao) GetByteRangeUrls() []*htsticket.URL {

	indexBodyReader, err := awsutils.GetS3Object(awsutils.S3Dto{
		ObjPath: dao.url + ".tbi",
	})

	if err != nil {
		log.Error("GetByteRangeUrls: %v", err)
		return nil
	}

	gz, err := gzip.NewReader(indexBodyReader)

	if err != nil {
		log.Error("GetByteRangeUrls: %v", err)
		return nil
	}

	defer gz.Close()

	t, err := tabix.ReadFrom(gz)
	if err != nil {
		log.Error("GetByteRangeUrls: %v", err)
		return nil
	}

	var goodNames = []string{"chr1", "chr2", "chr3", "chr4", "chr5", "chr6", "chr7", "chr8", "chr9", "chr10", "chr11", "chr12", "chr13", "chr14", "chr15", "chr16", "chr17", "chr18", "chr19", "chr20", "chr21", "chr22", "chr23", "chrM", "chrX", "chrY"}

	sort.Strings(goodNames)

	urls := []*htsticket.URL{}

	// start by adding in the header (everything *before* the first index chunk)
	for _, name := range t.Names() {
		chunks, _ := t.Chunks(name, 0, 1000000000)

		for _, chunk := range chunks {
			urls = append(urls, MakeHeaderUrl(dao, chunk))
			break
		}
		break
	}

	for _, name := range t.Names() {

		i := sort.SearchStrings(goodNames, name)

		if i < len(goodNames) && goodNames[i] == name {
			// we want all the chunks for this name
			chunks, _ := t.Chunks(name, 0, 1000000000)

			for _, chunk := range chunks {
				// we only include those chunks from the main known regions (I know this is bad but for demo
				// purposes this is fine - and avoid super large responses of all these tiny unmapped regions)
				urls = append(urls, MakeBodyUrl(dao, name, chunk))
			}
		} else {
			log.Debug("SKipping chunk %s because we did not recognise the region name", name)
		}
	}

	return urls
}

// GetChunkedInPlaceBlocks return the URLs for this
func (dao *AWSDao) GetChunkedInPlaceBlocks(regions []*htsrequest.Region) []*htsticket.URL {

	startTime := time.Now()

	// locate the index file and read it in
	indexBodyReader, err := awsutils.GetS3Object(awsutils.S3Dto{
		ObjPath: dao.url + ".tbi",
	})

	gz, err := gzip.NewReader(indexBodyReader)

	if err != nil {
		log.Error("GetChunkedInPlaceBlocks: %v", err)
		return nil
	}

	defer gz.Close()

	t, err := tabix.ReadFrom(gz)
	if err != nil {
		log.Error("GetChunkedInPlaceBlocks: %v", err)
		return nil
	}

	// Code to measure
	duration := time.Since(startTime)

	log.Debug("loading tabix = %s", duration)

	// we are going to build an array of URLs pointing directly at the blocks in S3
	urls := make([]*htsticket.URL, 0)

	// start by adding in the header (everything *before* the first index chunk)
	for _, name := range t.Names() {
		chunks, _ := t.Chunks(name, 0, 1000000000)

		for _, chunk := range chunks {
			urls = append(urls, MakeHeaderUrl(dao, chunk))
			break
		}
		break
	}

	// for every region requested
	for _, r := range regions {
		// handle open-ended region request
		start := 0
		if r.StartRequested() {
			start = r.GetStart()
		}

		// handle open-ended region request (i.e all of "chr1") by asking for the region up to maxint unless set
		end := 1000000000
		if r.EndRequested() {
			end = r.GetEnd()
		}

		startTime = time.Now()

		// consult the index for the block range of the asked for region
		chunks, next, _ := t.ChunksWithNext(r.GetReferenceName(), start, end)

		chunksLookupDuration := time.Since(startTime)

		log.Debug("Region %s %d-%d lookup into %d BGZIP chunks took %s", r.GetReferenceName(), start, end, len(chunks), chunksLookupDuration)


		for _, chunk := range chunks {
			startTime = time.Now()

			// reads the GZIP block header from the actual file data
			// this is the only location for the actual block size data
			//lastBlockReader, _ := awsutils.GetS3ObjectRange(awsutils.S3Dto{
			//	ObjPath: dao.url,
			//}, chunk.End.File, chunk.End.File+128)

			//lastBlockContents, _ := ioutil.ReadAll(lastBlockReader)
			//lastBlockSize := binary.LittleEndian.Uint16(lastBlockContents[16:]) + 1

			blockHeaders := htsticket.NewHeaders().SetRangeHeader(chunk.Begin.File, next-1)

			req, err := awsutils.PresignGetObjectRange(awsutils.S3Dto{
				ObjPath: dao.url,
			}, chunk.Begin.File, next-1)

			chunkDuration := time.Since(startTime)

			log.Debug("Index chunk %v extended out to %d and took %s", chunk, next, chunkDuration)
			// log.Debug("Index chunk %v and a final block size of %d (from data 2 bytes before .. %v .. 2 bytes after) took %s", chunk, lastBlockSize, lastBlockContents[14:20], chunkDuration)

			if err != nil {
				log.Error("Skipping chunk due to error creating pre-signed S3 link %v", err)

			} else {
				url := htsticket.NewURL().
					SetURL(req).
					SetHeaders(blockHeaders).
					SetClassBody()

				urls = append(urls, url)
			}

		}
	}

	// type Chunk struct {
	//	Begin struct {
	//	//	File  int64
	//	//	Block uint16
	//	//}
	//	End   struct {
	//	//	File  int64
	//	//	Block uint16
	//	//}
	//}
	//Chunk is a region of a BGZF file.
	//type Offset ¶
	//type Offset
	//blockSize := htsconstants.SingleBlockByteSize
	//var start, end, numBytes int64 = 0, 0, 0
	//numBlocks := 0 //int(math.Ceil(float64(numBytes) / float64(blockSize)))
	/*for i := 1; i <= numBlocks; i++ {
		end = start + blockSize - 1
		if end >= numBytes {
			end = numBytes - 1
		}
		headers := htsticket.NewHeaders()
		headers.SetRangeHeader(start, end)
		url := htsticket.NewURL()
		url.SetURL(dao.url)
		url.SetHeaders(headers)
		start = end + 1
		urls = append(urls, url)
	}*/
	//nBlocks := handler.HtsReq.NRegions() + 1
	//blockURLs = addHeaderBlockURL(blockURLs, handler.HtsReq, nBlocks)
	//for i := range handler.HtsReq.GetRegions() {
	//	blockURLs = addBodyBlockURL(blockURLs, handler.HtsReq, i+1, nBlocks, true, i)
	//}
	return urls
}

func (dao *AWSDao) String() string {
	return "AWSDao id=" + dao.id + ", url=" + dao.url
}

func MakeHeaderUrl(dao *AWSDao, firstChunk bgzf.Chunk) *htsticket.URL {
	// the first chunk by definition tells us the bounds of the header (which occurs before it) (are we sure this is true??)
	log.Debug("Header was discovered to finish at %d", firstChunk.Begin.File-1)

	blockHeaders := htsticket.NewHeaders().SetRangeHeader(0, firstChunk.Begin.File-1)

	req, _ := awsutils.PresignGetObjectRange(awsutils.S3Dto{
		ObjPath: dao.url,
	}, 0, firstChunk.Begin.File-1)

	return htsticket.NewURL().
		SetURL(req).
		SetHeaders(blockHeaders).
		SetClassHeader()
}

func MakeBodyUrl(dao *AWSDao, ref string, chunk bgzf.Chunk) *htsticket.URL {
	log.Debug("Body chunk for reference %s from %d-%d", ref, chunk.Begin.File+int64(chunk.Begin.Block), chunk.End.File+int64(chunk.End.Block)-1)

	blockHeaders := htsticket.NewHeaders().SetRangeHeader(chunk.Begin.File+int64(chunk.Begin.Block), chunk.End.File+int64(chunk.End.Block)-1)

	req, err := awsutils.PresignGetObjectRange(awsutils.S3Dto{
		ObjPath: dao.url,
	}, chunk.Begin.File+int64(chunk.Begin.Block), chunk.End.File+int64(chunk.End.Block)-1)

	if err != nil {
		log.Error("Creating pre-signed URL %v", err)
	}

	return htsticket.NewURL().
		SetURL(req).
		SetHeaders(blockHeaders).
		SetClassBody()
}
