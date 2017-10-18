package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
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

func init() {
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "json",
			Usage: "output json",
		},
	}
}

type ToDo struct {
	token  *oauth2.Token
	config *oauth2.Config
	file   string
}

func (todo *ToDo) doAPI(ctx context.Context, method string, uri string, params interface{}, res interface{}) error {
	var stream io.Reader
	if params != nil {
		buf := new(bytes.Buffer)
		err := json.NewEncoder(buf).Encode(params)
		if err != nil {
			return err
		}
		stream = buf
	}

	req, err := http.NewRequest(method, uri, stream)
	if err != nil {
		return err
	}
	req.WithContext(ctx)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+todo.token.AccessToken)
	client := todo.config.Client(ctx, todo.token)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var r io.Reader = resp.Body
	//r = io.TeeReader(resp.Body, os.Stdout)

	if res != nil {
		err = json.NewDecoder(r).Decode(res)
	} else {
		_, err = io.Copy(ioutil.Discard, r)
	}
	return err
}

func (todo *ToDo) Setup() error {
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
		return err
	}
	todo.file = filepath.Join(dir, "settings.json")

	b, err := ioutil.ReadFile(todo.file)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, &todo.token)
	if err != nil {
		return fmt.Errorf("could not unmarshal %v: %v", todo.file, err)
	}
	return nil
}

func (todo *ToDo) AccessToken() error {
	l, err := net.Listen("tcp", "localhost:8989")
	if err != nil {
		return err
	}
	defer l.Close()

	stateBytes := make([]byte, 16)
	_, err = rand.Read(stateBytes)
	if err != nil {
		return err
	}

	state := fmt.Sprintf("%x", stateBytes)

	err = open.Start(todo.config.AuthCodeURL(state, oauth2.SetAuthURLParam("response_type", "code")))
	if err != nil {
		return err
	}

	quit := make(chan string)
	go http.Serve(l, http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		code := req.URL.Query().Get("code")
		if code == "" {
			w.Write([]byte(`<script>document.write(location.hash)</script>`))
		} else {
			w.Write([]byte(`<script>window.open("about:blank","_self").close()</script>`))
		}
		w.(http.Flusher).Flush()
		quit <- code
	}))

	todo.token, err = todo.config.Exchange(context.Background(), <-quit)
	if err != nil {
		return fmt.Errorf("failed to exchange access-token: %v", err)
	}

	b, err := json.MarshalIndent(todo.token, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to store file: %v", err)
	}
	err = ioutil.WriteFile(todo.file, b, 0700)
	if err != nil {
		return fmt.Errorf("failed to store file: %v", err)
	}
	return nil
}

func initialize(c *cli.Context) error {
	todo := &ToDo{
		config: &oauth2.Config{
			Scopes: []string{
				"offline_access",
				"https://outlook.office.com/Tasks.Readwrite",
			},
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
				TokenURL: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
			},
			ClientID:     "da334f61-816a-4cda-a0bf-d1b6aa9240da",
			ClientSecret: "jRRTkBgcvDoxZSCSppzp0vn",
			RedirectURL:  "http://localhost:8989",
		},
	}
	err := todo.Setup()
	if err != nil {
		return fmt.Errorf("failed to get configuration: %v", err)
	}

	if todo.token == nil || todo.token.RefreshToken == "" {
		err = todo.AccessToken()
		if err != nil {
			return fmt.Errorf("faild to get access token: %v", err)
		}
	}

	app.Metadata["todo"] = todo
	return nil
}

func main() {
	app.Name = "to-do"
	app.Usage = "Microsoft To-Do client"
	app.Version = "0.0.1"
	app.Before = initialize
	app.Setup()
	app.Run(os.Args)
}
