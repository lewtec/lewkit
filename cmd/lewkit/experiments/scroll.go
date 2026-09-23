package experiments

import (
	"context"
	"fmt"
	"image"
	"log/slog"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/ui/gui"
)

const vulkanEvaluatorID = "ndeval_vulkan"

type scrollCmd struct {
	width  cmd.IntArg[int] `long:"width" default:"1024" help:"window width"`
	height cmd.IntArg[int] `long:"height" default:"1024" help:"window height"`
}

func (scrollCmd) Description() string {
	return "1024x1024 Vulkan benchmark of rounded scrolling boxes"
}

func (c *scrollCmd) Run(ctx context.Context) error {
	model, err := gui.NewMarquee()
	if err != nil {
		return err
	}
	width, height := c.width.Value(), c.height.Value()
	model.Update(window.Resize{Size: image.Pt(width, height)})
	evaluator, err := openVulkanEvaluator(ctx)
	if err != nil {
		return err
	}
	defer evaluator.Close()
	if named, ok := evaluator.(interface{ Name() string }); ok {
		slog.Info("scroll", "evaluator", named.Name(), "width", width, "height", height)
	}
	return runGUI(ctx, "scroll", gui.Options{
		Title:     "lewkit scroll",
		Width:     width,
		Height:    height,
		Evaluator: evaluator,
	}, model)
}

func openVulkanEvaluator(ctx context.Context) (ndarray.Evaluator, error) {
	handles, err := driver.List[ndarray.Evaluator](ctx)
	if err != nil {
		return nil, err
	}
	for _, handle := range handles {
		if handle.ID != vulkanEvaluatorID {
			continue
		}
		return handle.Open(ctx)
	}
	return nil, fmt.Errorf("%w: %s", driver.ErrUnavailable, vulkanEvaluatorID)
}
