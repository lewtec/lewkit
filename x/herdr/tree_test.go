package herdr

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportNodesNestWorktrees(t *testing.T) {
	report := Report{
		Pin: "/pin",
		Order: []Space{
			{ID: "pin", Label: "dotfiles", Checkout: "/pin", RepoRoot: "/pin", Branch: "master", Source: "herdr"},
			{ID: "wt", Label: "feat", Checkout: "/pin-wt", RepoRoot: "/pin", Linked: true, Branch: "feat", Source: "git"},
			{ID: "main", Label: "lewkit", Checkout: "/repo", RepoRoot: "/repo", Branch: "main", Source: "herdr"},
			{ID: "feat", Label: "topic", Checkout: "/repo-wt", RepoRoot: "/repo", Linked: true, Branch: "topic", Source: "layout", Identity: "/old"},
			{ID: "dir", Label: "tmp", Checkout: "/tmp"},
		},
	}
	nodes := report.Nodes()
	byName := map[string]int{}
	for i, node := range nodes {
		if _, ok := byName[node.Name]; !ok {
			byName[node.Name] = i
		}
	}
	require.Contains(t, byName, "dotfiles")
	require.Contains(t, byName, "feat")
	require.Contains(t, byName, "lewkit")
	require.Contains(t, byName, "topic")
	require.Contains(t, byName, "tmp")

	assert.Equal(t, nodes[byName["dotfiles"]].ID, nodes[byName["feat"]].Parent)
	assert.Equal(t, nodes[byName["lewkit"]].ID, nodes[byName["topic"]].Parent)
	assert.Zero(t, nodes[byName["dotfiles"]].Parent)
	assert.Zero(t, nodes[byName["lewkit"]].Parent)
	assert.Zero(t, nodes[byName["tmp"]].Parent)
	assert.Less(t, byName["dotfiles"], byName["feat"])
	assert.Less(t, byName["feat"], byName["lewkit"])
	assert.Less(t, byName["topic"], byName["tmp"])

	var underDot []string
	dot := nodes[byName["dotfiles"]].ID
	for _, node := range nodes {
		if node.Parent == dot {
			underDot = append(underDot, node.Name)
		}
	}
	assert.Equal(t, []string{"kind", "branch", "checkout", "source", "feat"}, underDot)

	var underTopic []string
	topic := nodes[byName["topic"]].ID
	for _, node := range nodes {
		if node.Parent == topic {
			underTopic = append(underTopic, node.Name+":"+node.Message)
		}
	}
	assert.Contains(t, underTopic, "repo:/repo")
	assert.Contains(t, underTopic, "identity:/old")
	assert.Contains(t, underTopic, "branch:topic")
}
