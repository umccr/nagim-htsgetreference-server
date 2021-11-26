package htsserver

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	log "github.com/ga4gh/htsget-refserver/internal/htslog"
	"github.com/ga4gh/htsget-refserver/internal/htsrequest"
	"github.com/jwangsadinata/go-multimap/slicemultimap"
	"github.com/s12v/go-jwks"
	"github.com/square/go-jose"
	"github.com/xenitab/go-oidc-middleware/options"
	"golang.org/x/crypto/ed25519"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/ga4gh/htsget-refserver/internal/htsdao"
	"github.com/ga4gh/htsget-refserver/internal/htserror"
	"github.com/ga4gh/htsget-refserver/internal/htsticket"
)

type HtsGetRegion struct {
	Id    string `json:"chromosome"`
	Start *int   `json:"start,omitempty"`
	End   *int   `json:"end,omitempty"`
}

type HtsGetArtifactConcrete struct {
	VariantsPath string `json:"variantsPath"`
}

type HtsGetArtifact struct {
	Samples map[string]HtsGetArtifactConcrete `json:"samples"`
}

type Manifest struct {
	Id         string                    `json:"id"`
	PatientIds []string                  `json:"patientIds"`
	Url        string                    `json:"htsgetUrl"`
	Artifacts  map[string]HtsGetArtifact `json:"htsgetArtifacts"`
	Regions    []HtsGetRegion            `json:"htsgetRegions"`
}

/**
Fills in the blockURLs with the corresponding blocks allowed as per this controlled access dataset
*/
func controlledAccess(issuer string, datasetId string, handler *requestHandler, dao *htsdao.DataAccessObject) []*htsticket.URL {
	// the first step for controlled access is to fetch the corresponding manifest
	url := fmt.Sprintf("%s/api/manifest/%s", issuer, datasetId)

	spaceClient := http.Client{
		Timeout: time.Second * 15,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Error("%v", err)
		return nil
	}

	res, getErr := spaceClient.Do(req)
	if getErr != nil {
		log.Error("%v", getErr)
		return nil
	}

	if res.Body != nil {
		defer res.Body.Close()
	}

	body, readErr := ioutil.ReadAll(res.Body)
	if readErr != nil {
		log.Error("%v", readErr)
		return nil
	}

	manifest := Manifest{}
	jsonErr := json.Unmarshal(body, &manifest)
	if jsonErr != nil {
		log.Error("%v", jsonErr)
		return nil
	}

	// TODO: do a basic check of is the specimen allowed to be accessed

	// log.Debug("%v", manifest)
	if handler.HtsReq.HeaderOnlyRequested() {
		// only header is requested, requires one URL range encompassing only the header data
		log.Debug("Ticket handler choosing a header only response")

		blockURLs := make([]*htsticket.URL, 0)

		headerBlockUrl := (*dao).GetHeaderByteRangeUrl()

		blockURLs = append(blockURLs, headerBlockUrl)

		return blockURLs
	}

	regions := make([]*htsrequest.Region, 0)

	if handler.HtsReq.AllRegionsRequested() {
		log.Debug("Ticket handler choosing a multi block all regions response")

		// the user has requested all regions - the result we give back is only those regions listed
		// in the manifest
		// TODO: enforce sort ordering on the manifest regions (is probably true currently but not guaranteed)
		for _, manifestRange := range manifest.Regions {
			// our manifest *only* ever uses chromosome ids in "1", "2", "X" format..

			// TODO: need to detect the underlying format of the VCF and match it.. currently all are chrX etc
			compatibleReferenceName := manifestRange.Id

			if !strings.HasPrefix(compatibleReferenceName, "chr") {
				compatibleReferenceName = fmt.Sprintf("chr%s", compatibleReferenceName)
			}

			regions = append(regions, &htsrequest.Region{ReferenceName: compatibleReferenceName, Start: manifestRange.Start, End: manifestRange.End})
		}

	} else {
		log.Debug("Ticket handler choosing a multi block selective regions response")

		// this possibly is useful for some logic but for the moment we aren't really using
		perChromosome := slicemultimap.New()

		for _, r := range handler.HtsReq.GetRegions() {
			perChromosome.Put(r.ReferenceName, r)
		}

		// TODO: fix the logic for the POST case where we can actually get different referenceNames
		// this probably only works for the single GET case at the moment

		for _, r := range handler.HtsReq.GetRegions() {

			if *r.Start == -1 && *r.End == -1 {

				log.Debug("Attempting to serve chromosome region %s", r.ReferenceName)

				for _, manifestRange := range manifest.Regions {
					compatibleReferenceName := manifestRange.Id

					if !strings.HasPrefix(compatibleReferenceName, "chr") {
						compatibleReferenceName = fmt.Sprintf("chr%s", compatibleReferenceName)
					}

					// chromosome names must match between the request region and manifest region or else we just skip to
					// next manifest rule
					if r.ReferenceName != compatibleReferenceName {
						continue
					}

					// because the user has asked for the whole chromosome - we are going to just serve up every manifest
					// region that matches
					regions = append(regions, &htsrequest.Region{ReferenceName: compatibleReferenceName, Start: manifestRange.Start, End: manifestRange.End})
				}
			} else {
				// for every region - we need to find a manifest region that 'allows' us
				// presume we aren't allowed
				allowed := false

				log.Debug("Attempting to get permission for region request %s %s-%s", r.ReferenceName, r.StartString(), r.EndString())

				for _, manifestRange := range manifest.Regions {

					// chromosome names must match between the request region and manifest region or else we just skip to
					// next manifest rule
					if r.ReferenceName != manifestRange.Id {
						if r.ReferenceName != fmt.Sprintf("chr%s", manifestRange.Id) {
							continue
						}
					}

					var manifestStart int
					if manifestRange.Start == nil {
						manifestStart = 0
					} else {
						manifestStart = *manifestRange.Start
					}

					var manifestEnd int
					if manifestRange.End == nil {
						manifestEnd = 1000000000
					} else {
						manifestEnd = *manifestRange.End
					}

					log.Debug("Comparing to manifest region %s %d-%d", manifestRange.Id, manifestStart, manifestEnd)

					var requestStart int
					var requestEnd int

					requestStart = *r.Start
					requestEnd = *r.End

					log.Debug("%d>=%d and %d<=%d", requestStart, manifestStart, requestEnd, manifestEnd)

					if requestStart >= manifestStart && requestEnd <= manifestEnd {
						allowed = true
						break
					}
				}

				if allowed {
					regions = append(regions, &htsrequest.Region{ReferenceName: r.GetReferenceName(), Start: r.Start, End: r.End})
				} else {
					handler.Writer.WriteHeader(403)

					json.NewEncoder(handler.Writer).Encode(fmt.Sprintf("Could not access region %s %s-%s", r.GetReferenceName(), r.StartString(), r.EndString()))

					return nil
				}
			}
		}
	}

	blockURLs := make([]*htsticket.URL, 0)

	urls := (*dao).GetChunkedInPlaceBlocks(regions)

	if urls == nil {
		log.Error("Chunked blocks call failed")
	}

	blockURLs = append(blockURLs, urls...)

	return blockURLs
}

const ISSUER_DAC = "https://didact-patto.dev.umccr.org"

func ticketRequestHandler(handler *requestHandler) {

	dao, err := htsdao.GetDao(handler.HtsReq)
	if err != nil {
		msg := "Could not determine data source path/url from request id"
		htserror.InternalServerError(handler.Writer, &msg)
		return
	}

	// part of our URL must be the dataset we are trying to access
	datasetRequested := handler.HtsReq.GetDataset()

	claims := handler.Request.Context().Value(options.DefaultClaimsContextKeyName).(map[string]interface{})

	var blockURLs []*htsticket.URL

	// this is just some wierdness about how Go JWT parses in the claims
	for _, visaOuter := range claims["ga4gh_passport_v2"].(map[string]interface{}) {
		visaInner := visaOuter.([]interface{})[0]
		v := visaInner.(map[string]interface{})["v"].(string)
		i := visaInner.(map[string]interface{})["i"].(string)
		k := visaInner.(map[string]interface{})["k"].(string)
		s := visaInner.(map[string]interface{})["s"].(string)

		// we only proceed with known *trusted* issuers
		if i == ISSUER_DAC {
			log.Info("Processing interesting visa from issuer %s with content %s", i, v)

			// set up a JWKS cache for visa TODO: actually cache this
			// TODO: use OIDC discovery rather than assuming jwks location
			visaJwksSource := jwks.NewWebSource(fmt.Sprintf("%s/.well-known/jwks", i))
			visaJwksClient := jwks.NewDefaultClient(
				visaJwksSource,
				time.Hour,    // Refresh keys every 1 hour
				12*time.Hour, // Expire keys after 12 hours
			)

			var jwk *jose.JSONWebKey
			jwk, err := visaJwksClient.GetEncryptionKey(k)
			if err != nil {
				log.Error(err.Error())
			}

			x := jwk.Key.(ed25519.PublicKey)

			vBytes := []byte(v)
			sBytes, err := base64.RawURLEncoding.DecodeString(s)

			if ed25519.Verify(x, vBytes, sBytes) {
				visaSplit := strings.Split(v, " ")

				for _, visaClaim := range visaSplit {
					// TODO: check expiry claims
					// TODO: check identity claims match outer passport
					// cover the situation that somehow the controlled access visa appears twice??
					if blockURLs != nil {
						continue
					}

					if strings.HasPrefix(visaClaim, "c:") {
						datasetId := strings.TrimPrefix(visaClaim, "c:")

						// the datset req in the URL has to match this visa - i.e. we need to cover the situation
						// where this user has many datasets at this DAC/htsget endpoint
						if datasetRequested == datasetId {
							blockURLs = controlledAccess(i, datasetId, handler, &dao)
						}
					}
				}
			} else {
				log.Error("Failed signature check for visa from %s", i)
			}
		} else {
			log.Info("Skipped uninteresting visa from issuer %s", i)
		}
	}

	if blockURLs == nil {
		msg := fmt.Sprintf("No valid controlled access visa from our trusted DACs (%v) was found matching dataset %s - so permission is denied", ISSUER_DAC, datasetRequested)
		htserror.PermissionDenied(handler.Writer, &msg)
		return
	}

	blockURLs = append(blockURLs, dao.GetBgzipEof())

	htsticket.FinalizeTicket(handler.HtsReq.GetFormat(), blockURLs, handler.Writer)
}
