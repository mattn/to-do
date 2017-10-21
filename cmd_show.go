package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/urfave/cli"
)

func init() {
	command := cli.Command{
		Name:    "show",
		Aliases: []string{"s"},
		Usage:   "Show specified to-do item",
		Action: func(c *cli.Context) error {
			if !c.Args().Present() {
				cli.ShowCommandHelp(c, "show")
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
			if c.Bool("json") {
				return json.NewEncoder(os.Stdout).Encode(&tasks.Value[i-1])
			}
			fmt.Println(tasks.Value[i-1].Subject)
			fmt.Println()
			fmt.Println(tasks.Value[i-1].Body.Content)
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
