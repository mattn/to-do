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
		Usage:   "list to-do",
		Action: func(c *cli.Context) error {
			var tasks struct {
				Value []Task `json:"value"`
			}
			todo := app.Metadata["todo"].(*ToDo)
			err := todo.doAPI(context.Background(), http.MethodGet, "https://outlook.office.com/api/v2.0/me/tasks", nil, &tasks)
			if err != nil {
				return err
			}
			for _, item := range tasks.Value {
				if item.Status == "Completed" {
					fmt.Print("* ")
				} else {
					fmt.Print("  ")
				}
				fmt.Println(item.Subject)
			}
			return nil
		},
	})
}
