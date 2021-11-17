package htsdao

import (
	"github.com/ga4gh/htsget-refserver/internal/awsutils"
	"github.com/ga4gh/htsget-refserver/internal/htsconfig"
	"github.com/ga4gh/htsget-refserver/internal/htsrequest"
	"github.com/ga4gh/htsget-refserver/internal/htsutils"
	"strings"
)

func getMatchingDao(id string, registry *htsconfig.DataSourceRegistry) (DataAccessObject, error) {
	path, err := registry.GetMatchingPath(id)
	if err != nil {
		return nil, err
	}
	if htsutils.IsValidURL(path) {
		if strings.HasPrefix(path, awsutils.S3Proto) {
			return NewAWSDao(id, path), nil
		} else {
			return NewURLDao(id, path), nil
		}
	}
	return NewFilePathDao(id, path), nil
}

func GetDao(req *htsrequest.HtsgetRequest) (DataAccessObject, error) {
	registry := req.GetDataSourceRegistry()
	return getMatchingDao(req.GetID(), registry)
}
