package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"golang.org/x/oauth2"

	"github.com/skratchdot/open-golang/open"
	"github.com/urfave/cli"
)

type TaskTime time.Time

func (tt TaskTime) MarshalJSON() ([]byte, error) {
	t := time.Time(tt)
	zone, _ := t.Zone()
	return []byte(fmt.Sprintf(`{"DateTime": %q, "TimeZone": %q}`, t.Format("2006-01-02T15:04:05"), string(zone))), nil
}

type Task struct {
	OdataID              string        `json:"@odata.id,omitempty"`
	OdataEtag            string        `json:"@odata.etag,omitempty"`
	ID                   string        `json:"Id,omitempty"`
	CreatedDateTime      *time.Time    `json:"CreatedDateTime,omitempty"`
	LastModifiedDateTime *time.Time    `json:"LastModifiedDateTime,omitempty"`
	ChangeKey            string        `json:"ChangeKey,omitempty"`
	Categories           []interface{} `json:"Categories,omitempty"`
	AssignedTo           string        `json:"AssignedTo,omitempty"`
	HasAttachments       bool          `json:"HasAttachments,omitempty"`
	Importance           string        `json:"Importance,omitempty"`
	IsReminderOn         bool          `json:"IsReminderOn,omitempty"`
	Owner                string        `json:"Owner,omitempty"`
	ParentFolderID       string        `json:"ParentFolderId,omitempty"`
	Sensitivity          string        `json:"Sensitivity,omitempty"`
	Status               string        `json:"Status,omitempty"`
	Subject              string        `json:"Subject,omitempty"`
	Body                 *struct {
		ContentType string `json:"ContentType"`
		Content     string `json:"Content"`
	} `json:"Body,omitempty"`
	CompletedDateTime *TaskTime `json:"CompletedDateTime,omitempty"`
	DueDateTime       *TaskTime `json:"DueDateTime,omitempty"`
	Recurrence        *TaskTime `json:"Recurrence,omitempty"`
	ReminderDateTime  *TaskTime `json:"ReminderDateTime,omitempty"`
	StartDateTime     *TaskTime `json:"StartDateTime,omitempty"`
}

type config map[string]string

var (
	app = cli.NewApp()
)

func (cfg config) doAPI(ctx context.Context, method string, uri string, params interface{}, res interface{}) error {
	var buf *bytes.Buffer
	if params != nil {
		buf = new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(params)
		if err != nil {
			return err
		}
	}

	req, err := http.NewRequest(method, uri, buf)
	if err != nil {
		return err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+cfg["AccessToken"])
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		fmt.Println(resp.Status)
	}
	defer resp.Body.Close()
	var r io.Reader
	r = io.TeeReader(resp.Body, os.Stdout)

	if res != nil {
		err = json.NewDecoder(r).Decode(res)
	} else {
		_, err = io.Copy(ioutil.Discard, r)
	}
	if err != nil {
		println(err.Error())
	}
	return err
}

func getConfig() (string, config, error) {
	dir := os.Getenv("HOME")
	if dir == "" && runtime.GOOS == "windows" {
		dir = os.Getenv("APPDATA")
		if dir == "" {
			dir = filepath.Join(os.Getenv("USERPROFILE"), "Application Data", "to-do")
		}
		dir = filepath.Join(dir, "to-do")
	} else {
		dir = filepath.Join(dir, ".config", "to-do")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", nil, err
	}
	file := filepath.Join(dir, "settings.json")
	cfg := config{}

	b, err := ioutil.ReadFile(file)
	if err != nil && !os.IsNotExist(err) {
		return "", nil, err
	}
	if err != nil {
		cfg["ClientID"] = "da334f61-816a-4cda-a0bf-d1b6aa9240da"
		cfg["ClientSecret"] = "jRRTkBgcvDoxZSCSppzp0vn"
	} else {
		err = json.Unmarshal(b, &cfg)
		if err != nil {
			return "", nil, fmt.Errorf("could not unmarshal %v: %v", file, err)
		}
	}
	return file, cfg, nil
}

func getAccessToken(config map[string]string) (string, error) {
	l, err := net.Listen("tcp", "localhost:8989")
	if err != nil {
		return "", err
	}
	defer l.Close()

	oauthConfig := &oauth2.Config{
		Scopes: []string{
			"https://outlook.office.com/Tasks.Readwrite",
		},
		Endpoint: oauth2.Endpoint{
			AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		},
		ClientID:     config["ClientID"],
		ClientSecret: config["ClientSecret"],
		RedirectURL:  "http://localhost:8989",
	}

	stateBytes := make([]byte, 16)
	_, err = rand.Read(stateBytes)
	if err != nil {
		return "", err
	}

	state := fmt.Sprintf("%x", stateBytes)
	//err = open.Start(oauthConfig.AuthCodeURL(state /*, oauth2.SetAuthURLParam("response_type", "code")*/))
	err = open.Start(oauthConfig.AuthCodeURL(state, oauth2.SetAuthURLParam("response_type", "code")))
	if err != nil {
		return "", err
	}

	quit := make(chan string)
	go http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		code := req.URL.Query().Get("code")
		if code == "" {
			w.Write([]byte(`<script>document.write(location.hash)</script>`))
		} else {
			w.Write([]byte(`<script>window.open("about:blank","_self").close()</script>`))
		}
		quit <- code
	}))

	code := <-quit

	token, err := oauthConfig.Exchange(context.Background(), code)
	if err != nil {
		return "", err
	}

	return token.AccessToken, nil
}

func main() {
	file, cfg, err := getConfig()
	if err != nil {
		log.Fatal("failed to get configuration:", err)
	}
	if cfg["AccessToken"] == "" {
		token, err := getAccessToken(cfg)
		if err != nil {
			log.Fatal("faild to get access token:", err)
		}
		cfg["AccessToken"] = token
		b, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			log.Fatal("failed to store file:", err)
		}
		err = ioutil.WriteFile(file, b, 0700)
		if err != nil {
			log.Fatal("failed to store file:", err)
		}
	}

	app.Name = "to-do"
	app.Usage = "Microsoft To-Do client"
	app.Setup()
	app.Metadata["config"] = cfg
	app.RunAndExitOnError()
}
