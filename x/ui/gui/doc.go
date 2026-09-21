// Package gui is a bubbletea-shaped loop whose View is a pixel tensor.
//
// The spine is [Open] with [Options] → [Run] + [Picture]. [Run] is the
// Elm loop: messages Update; View and Present run on the display ticker
// (and Resize/Expose), skipped when the view signature is unchanged.
// Animation is [Tick] / [Every]. A root [Row]/[Column] fills the window; wrap it in a
// [Box] with Align to center a packed inner cluster. Solid, Marquee, and
// Notepad all paint through one fused kernel
// (rounded-rect slots + ink overlay). Picture keeps a painter scratch;
// Marquee rewrites bar Y on a reused node tree. Layout is CPU ([Box], [Flex], [Stack]).
// Glyphs are [Text] nodes; Picture rasters them into ink in the same kernel.
// [Marquee] View maps offset and size onto [Bar] values; Update is the
// only writer. [Counter] is the Elm example (one int, two buttons).
package gui
