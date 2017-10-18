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
		Name:    "delete",
		Aliases: []string{"d"},
		Usage:   "Delete specified to-do item",
		Action: func(c *cli.Context) error {
			if !c.Args().Present() {
				cli.ShowCommandHelp(c, "delete")
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
			if i < 1 || i > len(tasks.Value) {
				return fmt.Errorf("invalid identifier: %v", i)
			}
			return todo.doAPI(context.Background(), http.MethodDelete, tasks.Value[i-1].OdataID, nil, nil)
		},
	})
}
