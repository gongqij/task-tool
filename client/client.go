package client

import (
	"io"
	"net/http"
	"net/url"
	"time"

	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

const DefaultTimeout = 10 * time.Minute

type HTTPAuthFunc func(method string, url *url.URL) (authStr string, err error)

type Options struct {
	HTTPAuthFunc HTTPAuthFunc
}

type Manager struct {
	endpoint   string
	httpClient *http.Client

	option Options
}

type TaskInfo struct {
	TaskID     string `json:"task_id"`
	ObjectType string `json:"object_type"`
}

type TaskStatus struct {
	Status string `json:"status"`
	Msg    string `json:"error_message"`
	Time   string `json:"last_received_time"`
}

type Task struct {
	Info   TaskInfo
	Status TaskStatus
}

type TaskInfoResp struct {
	Tasks []Task
}

func NewManager(endpoint string, timeout time.Duration, opts ...Options) *Manager {
	var option Options
	if len(opts) > 0 {
		option = opts[0]
	}
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	return &Manager{
		endpoint:   endpoint,
		httpClient: &http.Client{Timeout: timeout, Transport: transCfg},
		option:     option,
	}
}

func (m *Manager) checkOffline() bool {
	return m.endpoint == ""
}

func (m *Manager) httpGet(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return m.httpDo(req)
}

func (m *Manager) httpDo(req *http.Request) (*http.Response, error) {
	if m.option.HTTPAuthFunc != nil {
		authStr, err := m.option.HTTPAuthFunc(req.Method, req.URL)
		if err != nil {
			return nil, err
		}
		if authStr != "" {
			req.Header.Add("Authorization", authStr)
		}
	}
	return m.httpClient.Do(req)
}

func (m *Manager) httpPost(url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return m.httpDo(req)
}

// ListAllTask
func (m *Manager) ListAllTasks() (*TaskInfoResp, error) {
	metaURL := fmt.Sprintf("%s/v1/tasks?page_request.offset=0&page_request.limit=1000", m.endpoint)
	req, err := http.NewRequest("GET", metaURL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid http request: %s", metaURL)
	}
	resp, err := m.httpDo(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() // nolint
	if resp.StatusCode == 200 {
		result := &TaskInfoResp{}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		fmt.Println(string(body))
		if err := json.Unmarshal(body, result); err != nil {
			return nil, err
		}
		return result, nil
	}
	return nil, fmt.Errorf("get all task failed: %v", resp.StatusCode)
}

func (m *Manager) GetTaskInfoById(taskId string) error {
	metaURL := fmt.Sprintf("%s/v1/tasks/%s", m.endpoint, taskId)
	req, err := http.NewRequest("GET", metaURL, nil)
	if err != nil {
		return fmt.Errorf("invalid http request: %s", metaURL)
	}
	resp, err := m.httpDo(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() // nolint
	if resp.StatusCode == 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		fmt.Println(string(body))
		return nil
	}
	return fmt.Errorf("get taskInfo failed: %v", resp.StatusCode)
}

/*
func (m *Manager) DeleteModel(mp *api.ModelPath) error {
	if m.checkOffline() {
		return ErrOfflineMode
	}
	url, err := localcache.ModelPathToFilename(mp, "")
	if err != nil {
		return err
	}
	metaURL := fmt.Sprintf("%s/v1/models/%s", m.endpoint, url)
	req, _ := http.NewRequest("DELETE", metaURL, nil)
	resp, err := m.httpDo(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() // nolint
	if resp.StatusCode == 200 {
		return nil
	}
	if resp.StatusCode == 404 {
		return ErrModelNotFound
	}
	return fmt.Errorf("delete model failed: %v", resp.StatusCode)
}*/

/*func fillRequestParams(req *http.Request, mpath *api.ModelPath) *http.Request {
	q := req.URL.Query()
	if mpath.GetType() != api.ModelType_MODEL_UNKNOWN {
		q.Set("model_path.type", mpath.GetType().String())
	}
	if len(mpath.GetSubType()) > 0 {
		q.Set("model_path.sub_type", mpath.GetSubType())
	}
	if len(mpath.GetRuntime()) > 0 {
		q.Set("model_path.runtime", mpath.GetRuntime())
	}
	if len(mpath.GetHardware()) > 0 {
		q.Set("model_path.hardware", mpath.GetHardware())
	}
	if len(mpath.GetName()) > 0 {
		q.Set("model_path.name", mpath.GetName())
	}
	req.URL.RawQuery = q.Encode()

	return req
}

func (m *Manager) UploadModel(model *api.Model, path string, overwrite bool) error {
	if m.checkOffline() {
		return ErrOfflineMode
	}
	checksum, size, err := localcache.ChecksumFile(path)
	if err != nil {
		return &Error{"checksum", err}
	}
	log.Info(path, " checksum: ", checksum, ", size: ", size)

	model.Checksum = checksum
	model.Size = size
	if model.Oid == "" {
		model.Oid = checksum
	}
	req := &api.ModelNewRequest{
		Overwrite: overwrite,
		Model:     model,
	}
	mar := jsonpb.Marshaler{OrigName: true}
	buf := bytes.NewBuffer(nil)
	err = mar.Marshal(buf, req)
	if err != nil {
		return &Error{"marshal", err}
	}

	f, err := os.Open(path) // #nosec
	if err != nil {
		return &Error{"open model", err}
	}
	defer f.Close() // nolint

	blobURL := fmt.Sprintf("%s/v1/blobs/%s", m.endpoint, checksum)
	resp, err := m.httpPost(blobURL, "application/octet-stream", f)
	// TODO retry
	if err != nil {
		return err
	}
	defer resp.Body.Close() // nolint
	switch resp.StatusCode {
	case http.StatusOK:
		log.Info("blob ", checksum, " uploaded")
	case http.StatusConflict:
		log.Info("blob ", checksum, " already exists")
	default:
		return fmt.Errorf("failed to upload blob: %v", resp.StatusCode)
	}

	metaURL := fmt.Sprintf("%s/v1/models", m.endpoint)
	resp, err = m.httpPost(metaURL, "application/json", buf)
	// TODO retry
	if err != nil {
		return err
	}
	defer resp.Body.Close() // nolint
	switch resp.StatusCode {
	case http.StatusOK:
		log.Info("model uploaded: ", model.GetModelPath())
		return nil
	case http.StatusConflict:
		log.Info("model already exists: ", model.GetModelPath())
		return ErrModelExists
	default:
		return fmt.Errorf("failed to upload meta: %v", resp.StatusCode)
	}
}

//SyncModel triggers models synchronization from minio to managers
func (m *Manager) SyncModel() error {
	if m.checkOffline() {
		return ErrOfflineMode
	}
	req := api.ModelSynchronizeRequest{}
	buf, err := marshalRequest(&req)
	if err != nil {
		return err
	}

	URL := fmt.Sprintf("%s/v1/models/synchronize", m.endpoint)
	resp, err := m.httpPost(URL, "application/json", buf) // #nosec
	if err != nil {
		log.Error("failed to trigger models synchronization, ", err)
		return err
	}
	defer resp.Body.Close() // nolint
	if resp.StatusCode >= 200 && resp.StatusCode < 299 {
		return nil
	}

	return fmt.Errorf("failed to trigger models synchronization: %v", resp.StatusCode)
}
*/

// GetSystemInfo returns storage capacity and the last synchronization time

/*func marshalRequest(req proto.Message) (*bytes.Buffer, error) {
	marshaler := jsonpb.Marshaler{OrigName: true}
	buf := bytes.NewBuffer(nil)
	err := marshaler.Marshal(buf, req)
	if err != nil {
		return nil, err
	}
	return buf, nil
}
*/
