package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"task-tool-cli/client"
	"task-tool-cli/version"
)

const AccessKey = "UzEatlPmdhdU9b3nSEp61I6Y"
const SecretKey = "Hwibpcoa9yaDBVuGU9kOEJo6"
const defaultTimeout = client.DefaultTimeout

func main() {
	app := cli.NewApp()
	app.Usage = "A CLI tool for model management"
	app.Version = version.GetVersion()
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name: "verbose",
		},
		cli.StringFlag{
			Name:  "endpoint",
			Usage: "HTTP endpoint for infra-model-manager",
			Value: "http://127.0.0.1:8081",
		},
		cli.StringFlag{
			Name:  "access_key",
			Usage: "JWT Access Key (optional)",
			Value: AccessKey,
		},
		cli.StringFlag{
			Name:  "secret_key",
			Usage: "JWT Secret Key (optional)",
			Value: SecretKey,
		},
		cli.BoolFlag{
			Name:   "insecure_ignore_tls",
			Usage:  "INSECURE, ignore unknown X509 authority",
			EnvVar: "TASK_INSECURE_IGNORE_TLS",
		},
	}

	var mgr *client.Manager
	app.Before = func(ctx *cli.Context) error {
		if ctx.Bool("verbose") {
			logrus.SetLevel(logrus.DebugLevel)
		}
		if ctx.Bool("insecure_ignore_tls") {
			tr := http.DefaultTransport.(*http.Transport)
			tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} // #nosec
		}

		opts := client.Options{
			HTTPAuthFunc: setupAuth(ctx.GlobalString("access_key"), ctx.GlobalString("secret_key")),
		}
		mgr = client.NewManager(ctx.GlobalString("endpoint"), defaultTimeout, opts)
		return nil
	}

	app.Commands = []cli.Command{
		{
			Name: "version",
			Action: func(c *cli.Context) error {
				version.PrintFullVersionInfo()
				return nil
			},
		},
		{
			Name:  "list",
			Usage: "List tasks",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "task_id",
					Usage: "optional, --task_id=xxx     get taskInfo by taskID",
				},
				cli.StringFlag{
					Name:  "object_type",
					Usage: "optional, --object_type=xxx get tasks by objectType",
				},
			},
			Action: func(c *cli.Context) error {
				if c.String("task_id") != "" {
					err := mgr.GetTaskInfoById(c.String("task_id"))
					if err != nil {
						return err
					}
					return nil
				}
				resp, err := mgr.ListAllTasks()
				if err != nil {
					return err
				}
				printResult(resp, c.String("object_type"))
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		logrus.Fatal(err)
	}

}

func setupAuth(acc, sec string) client.HTTPAuthFunc {
	if acc == "" || sec == "" {
		return nil
	}
	return func(endpoint string, url *url.URL) (string, error) {
		token, err := getToken(acc, sec, url.Scheme+"://"+url.Host)
		if err != nil {
			return "", err
		}
		return "Bearer " + token, nil
	}
}

type TokenResp struct {
	Code  int32  `json:"code"`
	Token string `json:"token"`
}

func getToken(accessKey string, secretKey string, endpoint string) (token string, errResult error) {
	jsonKey := []byte(fmt.Sprintf(`{"access_key": "%s", "secret_key": "%s"}`, accessKey, secretKey))
	url := fmt.Sprintf("%s/components/user_manager/v1/users/sign_token", endpoint)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonKey))
	if err != nil {
		return "", fmt.Errorf("invalid http request: %s", url)
	}
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}
	client := &http.Client{Transport: transCfg}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	result := TokenResp{}
	if resp.StatusCode == 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return "", err
		}
	}
	return result.Token, nil
}

type List struct {
	taskID string
	status string
}

func printResult(resp *client.TaskInfoResp, objectType string) {
	listRes := make(map[string][]List)
	logrus.Info("Total tasks Num: ", len(resp.Tasks))
	for _, v := range resp.Tasks {
		listRes[v.Info.ObjectType] = append(listRes[v.Info.ObjectType], List{
			taskID: v.Info.TaskID,
			status: v.Status.Status,
		})
	}
	if objectType != "" {
		var objectTypeExist bool
		for k, _ := range listRes {
			if objectType == k {
				objectTypeExist = true
				break
			}
		}
		if !objectTypeExist {
			logrus.Fatalf("objectType:%s 不存在！！！", objectType)
		}
	}

	for k, v := range listRes {
		if objectType != "" && k != objectType {
			continue
		}
		fmt.Println("---------------------------------------------------------------------------")
		if k == "OBJECT_ALGO" {
			fmt.Printf("%s\t\t\t\t\tTotalNum:%d\n", k, len(v))
		} else {
			fmt.Printf("%s\t\t\tTotalNum:%d\n", k, len(v))
		}
		for _, v := range v {
			fmt.Printf("  taskID:%s\t\t\tstatus:%s\n", v.taskID, v.status)
		}
	}
}
