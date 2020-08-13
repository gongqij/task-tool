package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

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
					Name:  "task_id,t",
					Usage: "optional, -task_id xxx -t xxx  get taskInfo by taskID",
				},
				cli.StringFlag{
					Name:  "object_type,o",
					Usage: "optional, -object_type xxx -o xxx get tasks by objectType",
				},
				cli.BoolFlag{
					Name:  "all,a",
					Usage: "optional, -all get all tasks",
				},
			},
			Action: func(c *cli.Context) error {
				flagNumVerify(c)
				if c.String("task_id") != "" {
					err := mgr.GetTaskInfoById(c.String("task_id"))
					if err != nil {
						return err
					}
					return nil
				}
				if c.String("object_type") != "" {
					resp, err := mgr.ListAllTasks()
					if err != nil {
						return err
					}
					printResult(resp, c.String("object_type"))
					return nil
				}
				if c.Bool("all") == true {
					resp, err := mgr.ListAllTasks()
					if err != nil {
						return err
					}
					printResult(resp, "")
					return nil
				}
				return nil
			},
		},
		{
			Name:  "add",
			Usage: "add tasks",
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "num,n",
					Usage: "required, -task_id xxx -t xxx  get taskInfo by taskID",
				},
				cli.StringFlag{
					Name:  "object_type,o",
					Usage: "required, -object_type xxx -o xxx get tasks by objectType",
				},
				cli.StringFlag{
					Name:  "rtsp,r",
					Usage: "required, -object_type xxx -o xxx get tasks by objectType",
				},
				cli.StringFlag{
					Name:  "minio_key,m",
					Usage: "optional, -object_type xxx -o xxx get tasks by objectType",
				},
			},
			Action: func(c *cli.Context) error {
				if c.Int("num") == 0 || c.String("object_type") == "" || c.String("rtsp") == "" {
					return fmt.Errorf("请指定任务数、任务类型和rtsp源")
				}
				err := mgr.AddTasks(c.Int("num"), c.String("object_type"), c.String("rtsp"), c.String("minio_key"))
				if err != nil {
					return err
				}
				logrus.Println("添加任务成功！！！")
				return nil
			},
		},
		{
			Name:  "delete",
			Usage: "delete tasks",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "task_id,t",
					Usage: "optional, -task_id xxx -t xxx  delete single task by taskID",
				},
				cli.StringFlag{
					Name:  "object_type,o",
					Usage: "optional, -object_type xxx -o xxx delete tasks by objectType",
				},
				cli.BoolFlag{
					Name:  "all,a",
					Usage: "optional, -all delete tasks by objectType",
				},
			},
			Action: func(c *cli.Context) error {
				flagNumVerify(c)
				if c.String("task_id") != "" {
					err := mgr.DeleteTaskById(c.String("task_id"))
					if err != nil {
						return err
					}
					return nil
				}
				if c.String("object_type") != "" {
					err := mgr.DeleteTaskByObjectType(c.String("object_type"))
					if err != nil {
						return err
					}
					return nil
				}
				if c.Bool("all") == true {
					err := mgr.DeleteAllTasks()
					if err != nil {
						return err
					}
					return nil
				}
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
	taskID        string
	status        string
	sourceAddress string
}

func printResult(resp *client.TaskInfoResp, objectType string) {
	listRes := make(map[string][]List)
	logrus.Info("Total tasks Num: ", len(resp.Tasks))
	for _, v := range resp.Tasks {
		listRes[v.Info.ObjectType] = append(listRes[v.Info.ObjectType], List{
			taskID:        v.Info.TaskID,
			status:        v.Status.Status,
			sourceAddress: v.Info.SourceAddress,
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
		fmt.Println("taskID\t\t\tstatus\t\t\tsource")
		for _, v := range v {
			fmt.Printf("%s\t%s\t%s\n", v.taskID, v.status, v.sourceAddress)
		}
	}
}

func flagNumVerify(c *cli.Context) {
	flagNum := 0
	if c.String("task_id") != "" {
		flagNum++
	}
	if c.String("object_type") != "" {
		flagNum++
	}
	if c.Bool("all") == true {
		flagNum++
	}
	if flagNum != 1 {
		logrus.Fatal("请指定一个参数：(-task_id、-t)或(-object_type、-o)或(-all,-a)")
	}
}
