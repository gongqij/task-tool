package client

/*
import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.sz.sensetime.com/viper/infra/infra-model-manager/api"
)

func Test_fillRequestParams(t *testing.T) {
	metaURL := "localhost:8999/v1/models"
	req, err := http.NewRequest("GET", metaURL, nil)
	assert.NoError(t, err)
	mpath := &api.ModelPath{}
	req = fillRequestParams(req, mpath)
	assert.Equal(t, metaURL, req.URL.String())
	mpath = &api.ModelPath{Type: api.ModelType_KESTREL}
	req = fillRequestParams(req, mpath)
	assert.Equal(t, "localhost:8999/v1/models?model_path.type=KESTREL", req.URL.String())

	req, err = http.NewRequest("GET", metaURL, nil)
	assert.NoError(t, err)
	mpath = &api.ModelPath{SubType: "align"}
	req = fillRequestParams(req, mpath)
	assert.Equal(t, "localhost:8999/v1/models?model_path.sub_type=align", req.URL.String())

	mpath = &api.ModelPath{Runtime: "trt3", Hardware: "nv_p4", Name: "any.model"}
	req = fillRequestParams(req, mpath)
	assert.Equal(t, "localhost:8999/v1/models?model_path.hardware=nv_p4&model_path.name=any.model&model_path.runtime=trt3&model_path.sub_type=align", req.URL.String())
}
*/
