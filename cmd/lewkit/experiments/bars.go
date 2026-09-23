package experiments

import (
	"context"
	"fmt"
	"image"
	"log/slog"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/ndarray"
	"github.com/lewtec/lewkit/x/ui/gui"
)

const barsSize = 1024

const vulkanEvaluatorID = "ndeval_vulkan"

// barsCmd is `lewkit experiments window bars`.
type barsCmd struct{}

func (barsCmd) Description() string {
	return "1024x1024 Vulkan marquee benchmark"
}

func (*barsCmd) Run(ctx context.Context) error {
	model, err := gui.NewMarquee()
	if err != nil {
		return err
	}
	model.Update(window.Resize{Size: image.Pt(barsSize, barsSize)})
	evaluator, err := openVulkanEvaluator(ctx)
	if err != nil {
		return err
	}
	defer evaluator.Close()
	if named, ok := evaluator.(interface{ Name() string }); ok {
		slog.Info("bars", "evaluator", named.Name(), "size", barsSize)
	}
	return runGUI(ctx, "bars", gui.Options{
		Title:     "lewkit bars",
		Width:     barsSize,
		Height:    barsSize,
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
