package experiments

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/driver/chooser"
)

// Choose is `lewkit experiments choose`.
type Choose struct {
	title    cmd.StringArg `long:"title" default:"" help:"dialog title"`
	dir      cmd.StringArg `long:"dir" default:"" help:"folder the dialog opens in"`
	name     cmd.StringArg `long:"name" default:"" help:"suggested file name when saving"`
	filter   cmd.StringArg `long:"filter" default:"" help:"glob, such as *.mp3"`
	folder   cmd.Flag      `long:"folder" help:"choose a folder"`
	save     cmd.Flag      `long:"save" help:"choose a path to save"`
	multiple cmd.Flag      `long:"multiple" help:"choose more than one path"`
}

func (Choose) Description() string {
	return "ask for files or a folder and print each path"
}

func (c *Choose) Run(ctx context.Context) error {
	req := chooser.Request{
		Title:     c.title.Value(),
		Directory: c.dir.Value(),
		Name:      c.name.Value(),
		Folder:    c.folder.Value(),
		Save:      c.save.Value(),
		Multiple:  c.multiple.Value(),
	}
	if pattern := c.filter.Value(); pattern != "" {
		req.Filters = []chooser.Filter{{Name: pattern, Patterns: []string{pattern}}}
	}
	paths, err := chooser.Choose(ctx, req)
	if err != nil {
		return err
	}
	for _, path := range paths {
		fmt.Println(path)
	}
	slog.Info("chooser", "paths", len(paths))
	return nil
}
