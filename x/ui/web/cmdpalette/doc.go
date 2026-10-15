// Package cmdpalette is a Cmd+K / Ctrl+K jump list for templ pages.
//
// Button and Modal use the plain stylesheet from Style.
// DaisyButton and DaisyModal use daisyUI classes on a page that already loads daisyUI.
// Both skins share Script. A row without an href or a label is dropped.
//
//	p := cmdpalette.New([]cmdpalette.Item{
//		cmdpalette.ItemOf("Reports", "Home", "/"),
//		cmdpalette.ItemOf("Accounts", "Assets:Cash", "/a/cash"),
//	}).WithTitle("Jump").WithButtonClass("btn btn-sm")
//
// In the layout:
//
//	@p.Assets()       // plain CSS and Script
//	@p.Button()
//	@p.Modal()
//	@p.DaisyButton()  // daisyUI skin; still call Script once
//	@p.DaisyModal()
//
// The host supplies Items. Filtering and Ctrl/⌘+K stay in Script.
// Plain CSS variables: --cmdpalette-accent, --cmdpalette-bg, --cmdpalette-fg,
// --cmdpalette-border, --cmdpalette-radius.
package cmdpalette
