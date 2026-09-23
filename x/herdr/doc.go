// Package herdr talks to a running Herdr session and lays out its workspaces.
//
// [Client] is the socket CLI: workspaces, tabs, panes, worktree open, and
// workspace.move_block. [Git] reads checkouts and linked worktrees.
// [RepoBranch] is the REPO:BRANCH argument. [Reorder] nests linked worktrees
// under an open main, parks a feature branch off the main checkout, renames
// worktree workspaces to their branch, and orders the pin checkout first.
package herdr
