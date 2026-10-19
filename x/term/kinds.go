package term

// Method is one RPC name. The string is the wire text.
//
//lewkit:enum
type Method string

const (
	MethodSessionGet Method = "session.get"
	MethodPaneList   Method = "pane.list"
	MethodPaneGet    Method = "pane.get"
	MethodPaneCreate Method = "pane.create"
	MethodPaneWrite  Method = "pane.write"
	MethodPaneKill   Method = "pane.kill"
	MethodWatch      Method = "watch"
	MethodTreeList   Method = "tree.list"
	MethodPaneMove   Method = "pane.move"
	MethodPaneClose  Method = "pane.close"
	MethodDirCreate  Method = "dir.create"
	MethodDirRemove  Method = "dir.remove"
	MethodDirMove    Method = "dir.move"
)

// Notice is a server push. The string is the wire text.
// A pane notice is that pane's record. A session notice is the ubersession
// membership. Clients that follow one pane do not receive session notices.
// A client watching a folder does.
//
//lewkit:enum
type Notice string

const (
	NoticePane       Notice = "pane"
	NoticePaneClosed Notice = "pane.closed"
	NoticeSession    Notice = "session"
)

// Action is one command-palette command. The string is the filter token.
//
//lewkit:enum
type Action string

const (
	ActionNew          Action = "new"
	ActionCommand      Action = "command"
	ActionUbersession  Action = "ubersession"
	ActionFollow       Action = "follow"
	ActionAlso         Action = "also"
	ActionUnfollow     Action = "unfollow"
	ActionFocus        Action = "focus"
	ActionSend         Action = "send"
	ActionKill         Action = "kill"
	ActionDetach       Action = "detach"
	ActionFolder       Action = "folder"
	ActionUp           Action = "up"
	ActionEnter        Action = "enter"
	ActionMove         Action = "move"
	ActionFollowFolder Action = "follow.folder"
)

// Step is where the command palette is asking for input.
//
//lewkit:enum
type Step string

const (
	StepActions Step = "actions"
	StepPanes   Step = "panes"
	StepLine    Step = "line"
	StepEntries Step = "entries"
)
