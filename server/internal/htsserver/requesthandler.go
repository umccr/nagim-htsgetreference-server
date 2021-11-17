package htsserver

import (
	"net/http"

	"github.com/ga4gh/htsget-refserver/internal/htsconstants"
	"github.com/ga4gh/htsget-refserver/internal/htsrequest"
)

type requestHandler struct {
	method         htsconstants.HTTPMethod
	endpoint       htsconstants.APIEndpoint
	Writer         http.ResponseWriter
	Request        *http.Request
	HtsReq         *htsrequest.HtsgetRequest
	afterSetupFunc func(handler *requestHandler) error
	handlerFunc    func(handler *requestHandler)
}

func newRequestHandler(method htsconstants.HTTPMethod, endpoint htsconstants.APIEndpoint, afterSetupFunc func(handler *requestHandler) error, handlerFunc func(handler *requestHandler)) *requestHandler {

	reqHandler := new(requestHandler)
	reqHandler.method = method
	reqHandler.endpoint = endpoint
	reqHandler.afterSetupFunc = afterSetupFunc
	reqHandler.handlerFunc = handlerFunc
	return reqHandler
}

func (reqHandler *requestHandler) setup(writer http.ResponseWriter, request *http.Request) error {
	// set all parameters
	htsgetReq, err := htsrequest.SetAllParameters(reqHandler.method, reqHandler.endpoint, writer, request)
	if err != nil {
		return err
	}
	// assign writer, golang request, and htsget request objects to the handler
	reqHandler.Writer = writer
	reqHandler.Request = request
	reqHandler.HtsReq = htsgetReq
	return nil
}

func (reqHandler *requestHandler) handleRequest(writer http.ResponseWriter, request *http.Request) error {
	// setup, create generic htsgetRequest and set to handler
	setupErr := reqHandler.setup(writer, request)
	if setupErr != nil {
		return setupErr
	}

	// after setup, perform any postprocessing steps on the generic htsgetRequest
	afterSetupErr := reqHandler.afterSetupFunc(reqHandler)
	if afterSetupErr != nil {
		return afterSetupErr
	}

	// execute the main handler function
	reqHandler.handlerFunc(reqHandler)
	return nil
}
