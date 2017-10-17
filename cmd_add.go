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
		Usage:   "add to-do",
		Action: func(c *cli.Context) error {
			if !c.Args().Present() {
				cli.ShowCommandHelp(c, "add")
				return nil
			}
			var task Task
			task.Subject = c.Args().First()
			cfg := app.Metadata["config"].(config)
			return cfg.doAPI(context.Background(), http.MethodPost, "https://outlook.office.com/api/v2.0/me/tasks", &task, nil)
		},
	})
}