package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/urfave/cli"
)

func init() {
	app.Commands = append(app.Commands, cli.Command{
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
			for i, item := range tasks.Value {
				mark := " "
				if item.Status == "Completed" {
					mark = "*"
				}
				fmt.Printf("%s %05d %s\n", mark, i+1, item.Subject)
			}
			return nil
		},
	})
}
