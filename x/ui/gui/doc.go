// Package gui is a bubbletea-shaped loop whose View is a layout [Node].
//
// The spine is [Open] with [Options] → [Run] + [Picture]. [Run] is the
// Elm loop: messages Update; View returns a Node; Run paints it on the
// display ticker (and Resize/Expose/Density), skipped when an embedded [Dirty] is clear. [See] and [Touch] return that flag. The picture signature
// is unchanged. Animation is [Tick] / [Every]. A root [Row]/[Column] fills the window; wrap it in a
// [Box] with Align to center a packed inner cluster. [Stage] paints [Stage.Scene] across the whole frame, including dead zones, and lays [Stage.Child] in the alive box. Render wraps a root that is not a Stage, so text stays in that box; a [Raster] root is the scene. [Hint] remains a nested inset into the same box. Solid, Marquee, and
// Notepad all paint through one fused kernel
// (one over-composite tensor of rounded rects + ink overlay). A present
// screen paints those same fills. Vulkan can paint a mounted tensor on
// the device. Metal keeps the fill list, so the process does not need
// MoltenVK. Layout lengths are logical pixels. [Run] multiplies them by the window
// scale and paints device pixels, so a control keeps its physical size when
// the window moves to a denser screen. The frame stays device pixels, so a
// raster still fills every pixel. Layout returns Size; Paint returns the accumulator tensor.
// A node records one [Mark] per draw, in paint order, through [Picture.Fill],
// [Picture.Text], [Picture.Image], and [Picture.Backdrop].
// Each [Picture.Render] streams those marks and lowers them into fills, ink, and the backdrop.
// [Play] draws the lowered [Frame] on a [Canvas]. [Attach] sends it to any [Drawer].
// [Vulkan] is that attachment for a swapchain. An OpenGL framebuffer implements the same [Canvas.Draw].
// Marquee rewrites bar Y on a reused node tree. Layout is CPU ([Box], [Flex], [Stack]).
// Glyphs are [Text] nodes; Picture rasters them into ink in the same kernel.
// [Marquee] View maps offset and size onto [Bar] values; Update is the
// only writer. [Counter] is the Elm example (one int, two buttons).
// [WelcomeArgs] is the start screen: the lockup, recent folders, and an optional accent.
// [EnsureDir] opens it when a tool still needs a directory and stdin is not a terminal.
package gui
