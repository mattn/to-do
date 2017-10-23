package main

import (
	"context"
	"net/http"

	"github.com/urfave/cli"
)

func init() {
	app.Commands = append(app.Commands, cli.Command{
		Name:    "add",
		Aliases: []string{"a"},
		Usage:   "Add new to-do item",
		Action: func(c *cli.Context) error {
			if !c.Args().Present() {
				cli.ShowCommandHelp(c, "add")
				return nil
			}
			todo := app.Metadata["todo"].(*ToDo)
			var task Task
			task.Subject = c.Args().First()
			content := c.Args().Get(1)
			task.Body = &TaskBody{
				ContentType: "Text",
				Content:     content,
			}
			return todo.doAPI(context.Background(), http.MethodPost, "https://outlook.office.com/api/v2.0/me/tasks", &task, nil)
		},
	})
}
