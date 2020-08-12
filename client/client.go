package client

import (
	"io"
	"net/http"
	"net/url"
	"time"

	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"strconv"
)

const DefaultTimeout = 10 * time.Minute
const DefaultEndPoint = "http://127.0.0.1:8081"

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
	TaskID        string `json:"task_id"`
	ObjectType    string `json:"object_type"`
	SourceAddress string `json:"source_address"`
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

func (m *Manager) httpGet(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	return m.httpDo(req)
}

func (m *Manager) httpPost(url, contentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return m.httpDo(req)
}

func (m *Manager) httpDelete(url string) (*http.Response, error) {
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}
	return m.httpDo(req)
}

// List
func (m *Manager) ListAllTasks() (*TaskInfoResp, error) {
	if m.checkOffline() {
		m.endpoint = DefaultEndPoint
	}
	metaURL := fmt.Sprintf("%s/v1/tasks?page_request.offset=0&page_request.limit=1000", m.endpoint)
	resp, err := m.httpGet(metaURL)
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
		//fmt.Println(string(body))
		if err := json.Unmarshal(body, result); err != nil {
			return nil, err
		}
		return result, nil
	}
	return nil, fmt.Errorf("get all task failed: %v", resp.StatusCode)
}

func (m *Manager) GetTaskInfoById(taskId string) error {
	if m.checkOffline() {
		m.endpoint = DefaultEndPoint
	}
	metaURL := fmt.Sprintf("%s/v1/tasks/%s", m.endpoint, taskId)
	resp, err := m.httpGet(metaURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close() // nolint
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode == 200 {
		fmt.Println(string(body))
		return nil
	}
	return fmt.Errorf("get taskInfo failed: %v", string(body))
}

//add
func (m *Manager) AddTasks(taskNum int, objectType, rtsp, key string) error {
	if m.checkOffline() {
		m.endpoint = DefaultEndPoint
	}
	body := []byte(fmt.Sprintf(`{"task":{"task_id":"123412341234","object_type":"%s","source_address":"%s","camera_info": {"camera_id":"1","device_id":"string","device_type":"string","place_code":"string","place_name":"string","tollgate_id":"string","tollgate_name":"string","source_id":"string","internal_id": {"region_id": 1,"camera_idx": 1 } },"camera_vision_info_config": {"vision_info_name":"traffic","vision_info_version": 10000,"data": {"key":"%s"},"enable_disaccord": true,"enable_anomaly": true},"output": {"protocol":"KAFKA","broker":"string","topic":"string","http_url":"string" },"creation_time":"2018-06-08T02:44:20.682Z","feature_version": 0 }}"`, objectType, rtsp, key))
	url := fmt.Sprintf("%s/v1/tasks", m.endpoint)
	errSlice := make([]string, taskNum)
	for i := 0; i < taskNum; i++ {
		resp, err := m.httpPost(url, "application/json", bytes.NewReader(body))
		if err != nil {
			errSlice[i] = err.Error()
		}
		respBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			errSlice[i] = err.Error()
		}
		if resp.StatusCode == 200 {
			logrus.Println("add one task succeeded")
			fmt.Println(string(respBody))
		} else {
			errSlice[i] = string(respBody)
		}
		resp.Body.Close()
	}
	var errStr string
	var errNum int
	for _, v := range errSlice {
		if v != "" {
			errNum++
			errStr = errStr + v
		}
	}
	if errStr != "" {
		return errors.New(errStr + "    failed num:" + strconv.Itoa(errNum))
	}
	return nil
}

func (m *Manager) DeleteTaskById(taskId string) error {
	if m.checkOffline() {
		m.endpoint = DefaultEndPoint
	}
	metaURL := fmt.Sprintf("%s/v1/tasks/%s", m.endpoint, taskId)
	resp, err := m.httpDelete(metaURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close() // nolint
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode == 200 {
		logrus.Infof("delete task succeeded taskID:%s", taskId)
		return nil
	}
	return fmt.Errorf("delete failed: %v", string(body))
}

func (m *Manager) DeleteAllTasks() error {
	allTask, err := m.ListAllTasks()
	if err != nil {
		return err
	}
	for _, v := range allTask.Tasks {
		err := m.DeleteTaskById(v.Info.TaskID)
		if err != nil {
			return err
		}
	}
	logrus.Info("delete all tasks succeeded")
	return nil
}

func (m *Manager) DeleteTaskByObjectType(objectType string) error {
	allTask, err := m.ListAllTasks()
	if err != nil {
		return err
	}
	for _, v := range allTask.Tasks {
		if v.Info.ObjectType == objectType {
			err := m.DeleteTaskById(v.Info.TaskID)
			if err != nil {
				return err
			}
		}
	}
	logrus.Info("delete all tasks succeeded")
	return nil
}
