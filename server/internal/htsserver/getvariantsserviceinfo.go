package htsserver

import (
	log "github.com/ga4gh/htsget-refserver/internal/htslog"
	"net/http"

	"github.com/ga4gh/htsget-refserver/internal/htsconstants"
)

func getVariantsServiceInfo(writer http.ResponseWriter, request *http.Request) {
	err := newRequestHandler(
		htsconstants.GetMethod,
		htsconstants.APIEndpointVariantsServiceInfo,
		noAfterSetup,
		serviceInfoRequestHandler,
	).handleRequest(writer, request)
	if err != nil {
		log.Error("%v", err)
		return
	}
}
