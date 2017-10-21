package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/fatih/color"
	runewidth "github.com/mattn/go-runewidth"
	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{
		Name:    "list",
		Aliases: []string{"l"},
		Usage:   "Show to-do items",
		Action: func(c *cli.Context) error {
			todo := app.Metadata["todo"].(*ToDo)
			var tasks struct {
				Value []Task `json:"value"`
			}
			err := todo.doAPI(context.Background(), http.MethodGet, "https://outlook.office.com/api/v2.0/me/tasks", nil, &tasks)
			if err != nil {
				return err
			}
			if c.Bool("json") {
				return json.NewEncoder(os.Stdout).Encode(&tasks.Value)
			}
			for i, item := range tasks.Value {
				mark := " "
				if item.Status == "Completed" {
					mark = "*"
				}
				subject := runewidth.Truncate(item.Subject, 70, "...")
				fmt.Fprint(color.Output, color.YellowString("%s", mark))
				fmt.Print(" ")
				fmt.Fprint(color.Output, color.GreenString("%05d", i+1))
				fmt.Println(" " + subject)
			}
			return nil
		},
	}
	command.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "json",
			Usage: "output json",
		},
	}
	app.Commands = append(app.Commands, command)
}
