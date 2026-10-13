package apk

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/build/gocmd"
	"github.com/lewtec/lewkit/x/build/version"
	"github.com/lewtec/lewkit/x/taskgroup"
	"github.com/lewtec/lewkit/x/workflow"
)

// BuildGoLibs cross-compiles the app into workDir/app/src/main/jniLibs/<abi>/libeletrocromo.so.
// stamp is injected via -ldflags -X (goreleaser-style) when apps import internal/version.
//
// The libraries are one workflow. jni depends on every ABI step, and it is
// the default, so the run waits for every library. A cgo-free arm64 step
// depends on toolexec, which compiles _cgo_android_export.go in so
// JNI_OnLoad is in .dynsym. Every other ABI step has no dependency, so
// those libraries build beside the wrapper.
// toolexec and each ABI step are orchestrators. They call gocmd, so the
// compiler on ctx builds a cgo-free arm64 library and the host wrapper.
// Both are in that compiler's scope. cgo and any other GOARCH are not.
// stdout is accepted for callers that already hold a log writer. The steps
// write on the taskgroup line writer.
func BuildGoLibs(ctx context.Context, workDir, goMainDir string, abis []string, stamp version.Info, appID string, cgoEnabled bool, stdout io.Writer) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	graph, paths, cleanup, err := libsGraph(workDir, goMainDir, abis, stamp, appID, cgoEnabled)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	fut, err := workflow.Run(ctx, graph, nil)
	if err != nil {
		return nil, err
	}
	if err := fut.Wait(ctx); err != nil {
		return nil, err
	}
	return paths, nil
}

// libsGraph is the Android Go libraries.
// jni depends on every ABI step.
// A cgo-free arm64 step depends on toolexec.
// The other ABI steps do not.
func libsGraph(workDir, goMainDir string, abis []string, stamp version.Info, appID string, cgoEnabled bool) (workflow.Graph, []string, func(), error) {
	if len(abis) == 0 {
		abis = DefaultABIs
	}
	ldflags := stamp.WithAppID(appID)
	var toolDir, toolArg string
	cleanup := func() {
		if toolDir != "" {
			os.RemoveAll(toolDir)
		}
	}
	var steps []workflow.Step
	needTool := false
	if !cgoEnabled {
		for _, abi := range abis {
			if abiToGOARCH[abi] == "arm64" {
				needTool = true
				break
			}
		}
	}
	if needTool {
		steps = append(steps, workflow.Step{
			Name:         "toolexec",
			Desc:         "toolexec publishes JNI_OnLoad",
			Orchestrator: true,
			Tasks: []workflow.Task{
				workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
					dir, arg, err := nocgoAndroidToolexec(ctx, []string{"arm64-v8a"}, false)
					if err != nil {
						return err
					}
					toolDir = dir
					toolArg = arg
					st.Update(arg)
					return nil
				}),
			},
		})
	}
	names := make([]string, 0, len(abis))
	paths := make([]string, 0, len(abis))
	seen := map[string]bool{}
	for _, abi := range abis {
		goarch, ok := abiToGOARCH[abi]
		if !ok {
			cleanup()
			return workflow.Graph{}, nil, func() {}, fmt.Errorf("%w: %q", ErrUnsupportedABI, abi)
		}
		name := "abi/" + abi
		if seen[name] {
			cleanup()
			return workflow.Graph{}, nil, func() {}, fmt.Errorf("workflow: %w: %s", workflow.ErrDuplicate, name)
		}
		seen[name] = true
		names = append(names, name)
		dest := filepath.Join(workDir, "app", "src", "main", "jniLibs", abi, "libeletrocromo.so")
		paths = append(paths, dest)
		deps := []string(nil)
		desc := abi
		if needTool && goarch == "arm64" {
			deps = []string{"toolexec"}
			desc = abi + " after toolexec"
		}
		abi, goarch := abi, goarch
		steps = append(steps, workflow.Step{
			Name:         name,
			Deps:         deps,
			Desc:         desc,
			Orchestrator: true,
			Tasks: []workflow.Task{
				workflow.Func(func(ctx context.Context, st *taskgroup.Status) error {
					st.Update(dest)
					arg := ""
					if goarch == "arm64" && !cgoEnabled {
						arg = toolArg
					}
					return buildABI(ctx, goMainDir, abi, goarch, dest, ldflags, arg, cgoEnabled)
				}),
			},
		})
	}
	steps = append(steps, workflow.Step{
		Name: "jni",
		Deps: names,
		Desc: "jni depends on every abi",
	})
	return workflow.Graph{Dir: workDir, Steps: steps, Defaults: []string{"jni"}}, paths, cleanup, nil
}

func buildABI(ctx context.Context, goMainDir, abi, goarch, dest, ldflags, toolexec string, cgoEnabled bool) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	slog.Info("android abi", "abi", abi, "goarch", goarch)
	cgoFlag := "0"
	linkFlags := ldflags
	args := []string{"-trimpath", "-ldflags", linkFlags, "-o", dest, "."}
	if cgoEnabled {
		args = []string{"-buildmode=c-shared", "-trimpath", "-ldflags", linkFlags, "-o", dest, "."}
	} else if goarch == "arm64" {
		// JNI_OnLoad is in .dynsym only when the wrapper compiles in
		// _cgo_android_export.go. -toolexec is that wrapper plus the
		// export file, so the compile cache key includes the file.
		// -tags appmode compiles the JNI host. The runtime symbol
		// references need -checklinkname=0.
		if toolexec == "" {
			return fmt.Errorf("go build %s: toolexec is empty", abi)
		}
		linkFlags = strings.TrimSpace(ldflags + " -checklinkname=0")
		args = []string{
			"-trimpath",
			"-tags", "appmode",
			"-toolexec", toolexec,
			"-ldflags", linkFlags,
			"-o", dest, ".",
		}
	}
	env := append(os.Environ(), "GOOS=android", "GOARCH="+goarch)
	if goarch == "arm" {
		env = append(env, "GOARM=7")
	}
	if cgoEnabled {
		cc, err := ndkCC(ctx, goarch)
		if err != nil {
			return err
		}
		cgoFlag = "1"
		env = append(env, "CC="+cc)
		slog.Info("android cgo", "cc", cc)
	}
	env = append(env, "CGO_ENABLED="+cgoFlag)
	// The step is an orchestrator. gocmd reads the compiler on ctx.
	// The host go command takes an IO task. The embedded compiler takes
	// CPU children. A cgo-free arm64 build is in that compiler's scope.
	if err := (gocmd.Command{Dir: goMainDir, Env: env, Args: args}).Run(ctx); err != nil {
		return fmt.Errorf("go build %s (GOARCH=%s CGO_ENABLED=%s): %w", abi, goarch, cgoFlag, err)
	}
	return nil
}
