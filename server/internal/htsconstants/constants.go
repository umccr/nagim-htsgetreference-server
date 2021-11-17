// Package htsconstants contains program constants
//
// Module constants contains general program constants
package htsconstants

import (
	"encoding/hex"
	"time"
)

/* **************************************************
 * TIME RELATED CONSTANTS
 * ************************************************** */

// StartupTime default startup time for service-info
var StartupTime = time.Date(2020, 9, 1, 12, 0, 0, 0, time.UTC).UTC().Format(time.RFC3339)

// SingleBlockByteSize suggested byte size of response from a single ticket url
var SingleBlockByteSize = int64(5e8)

// BamFieldsN canonical number of fields in SAM/BAM (excluding tags)
var BamFieldsN = 11

// BamFields (map[string]int): ordered map of canonical column name to position
var BamFields map[string]int = map[string]int{
	"QNAME": 0,  // read names
	"FLAG":  1,  // read bit flags
	"RNAME": 2,  // reference sequence name
	"POS":   3,  // alignment position
	"MAPQ":  4,  // mapping quality score
	"CIGAR": 5,  // CIGAR string
	"RNEXT": 6,  // reference sequence name of the next fragment template
	"PNEXT": 7,  // alignment position of the next fragment in the template
	"TLEN":  8,  // inferred template size
	"SEQ":   9,  // read bases
	"QUAL":  10, // base quality scores
}

// BamExcludedValues correct values when column is removed by column
var BamExcludedValues []string = []string{
	"*",   // QNAME
	"0",   // FLAG
	"*",   // RNAME
	"0",   // POS
	"255", // MAPQ
	"*",   // CIGAR
	"*",   // RNEXT
	"0",   // PNEXT
	"0",   // TLEN
	"*",   // SEQ
	"*",   // QUAL
}

// BamBGZF bytes marking BGZF Block
var BamBGZF, _ = hex.DecodeString("1f8b08040000000000ff060042430200")

// BamBGZFLen byte length of a single BGZF block header marker
var BamBGZFLen = len(BamBGZF)

// BamEOF BAM end of file byte sequence
var BamEOF, _ = hex.DecodeString("1f8b08040000000000ff0600424302001b0003000000000000000000")

// BamEOFLen length (number of bytes) of BAM end of file byte sequence
var BamEOFLen = len(BamEOF)

// ReadsDataURLPath path to reads data endpoint
var ReadsDataURLPath = "reads/data/"

// VariantsDataURLPath path to variants data endpoint
var VariantsDataURLPath = "variants/data/"

// FileByteRangeURLPath path to local file bytestream endpoint
var FileByteRangeURLPath = "file-bytes"

// FormatBam canonical htsget format string for .bam files
var FormatBam = "BAM"

// FormatCram canonical htsget format string for .cram files
var FormatCram = "CRAM"

// FormatVcf canonical htsget format string for .vcf(.gz) files
var FormatVcf = "VCF"

// FormatBcf canonical htsget format string for .bcf files
var FormatBcf = "BCF"

// ClassHeader canonical htsget class string for header segment
var ClassHeader = "header"

// ClassBody canonical htsget class string for body segment
var ClassBody = "body"
