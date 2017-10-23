package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/urfave/cli"
)

func init() {
	app.Commands = append(app.Commands, cli.Command{
		Name:    "done",
		Aliases: []string{"d"},
		Usage:   "Done the specified to-do item",
		Action: func(c *cli.Context) error {
			if !c.Args().Present() {
				cli.ShowCommandHelp(c, "done")
				return nil
			}
			todo := app.Metadata["todo"].(*ToDo)
			var tasks struct {
				Value []Task `json:"value"`
			}
			err := todo.doAPI(context.Background(), http.MethodGet, "https://outlook.office.com/api/v2.0/me/tasks", nil, &tasks)
			if err != nil {
				return err
			}
			i, err := strconv.Atoi(c.Args().First())
			if err != nil {
				return err
			}
			if i < 1 || i >= len(tasks.Value) {
				return fmt.Errorf("invalid identifier: %v", i)
			}
			return todo.doAPI(context.Background(), http.MethodPost, tasks.Value[i-1].OdataID+"/complete", nil, nil)
		},
	})
}
