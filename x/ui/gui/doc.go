// Package gui is a bubbletea-shaped loop whose View is a layout [Node].
//
// The spine is [Open] with [Options] → [Run] + [Picture]. [Run] is the
// Elm loop: messages Update; View returns a Node; Run paints it on the
// display ticker (and Resize/Expose), skipped when the picture signature
// is unchanged. Animation is [Tick] / [Every]. A root [Row]/[Column] fills the window; wrap it in a
// [Box] with Align to center a packed inner cluster. Solid, Marquee, and
// Notepad all paint through one fused kernel
// (one over-composite tensor of rounded rects + ink overlay). Layout returns Size; Paint returns the accumulator tensor.
// Marquee rewrites bar Y on a reused node tree. Layout is CPU ([Box], [Flex], [Stack]).
// Glyphs are [Text] nodes; Picture rasters them into ink in the same kernel.
// [Marquee] View maps offset and size onto [Bar] values; Update is the
// only writer. [Counter] is the Elm example (one int, two buttons).
package gui
