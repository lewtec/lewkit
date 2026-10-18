package graphviz

// Term is one Graphviz token.
//
//lewkit:enum
type Term string

const (
	TermStrict     Term = "strict"
	TermGraph      Term = "graph"
	TermDigraph    Term = "digraph"
	TermSubgraph   Term = "subgraph"
	TermNode       Term = "node"
	TermEdge       Term = "edge"
	TermIdentifier Term = "identifier"
	TermNumber     Term = "number"
	TermString     Term = "string"
	TermHTMLLabel  Term = "html_label"
	TermSkip       Term = "skip"
)

// Rule is one Graphviz rule. Hidden rules the builder invents are not members.
//
//lewkit:enum
type Rule string

const (
	RuleFile               Rule = "file"
	RuleTop                Rule = "top"
	RuleStatement          Rule = "statement"
	RuleAttributeStatement Rule = "attribute_statement"
	RuleLead               Rule = "lead"
	RuleTail               Rule = "tail"
	RuleList               Rule = "list"
	RuleEdgeTail           Rule = "edge_tail"
	RuleSubgraphStatement  Rule = "subgraph_statement"
	RuleStatementTail      Rule = "statement_tail"
	RuleBlock              Rule = "block"
	RuleEnd                Rule = "end"
	RuleIdentifiers        Rule = "identifiers"
	RuleNodeIdentifier     Rule = "node_identifier"
	RulePort               Rule = "port"
	RuleAttributeList      Rule = "attribute_list"
	RuleMore               Rule = "more"
	RulePair               Rule = "pair"
	RuleAtom               Rule = "atom"
	RuleStrings            Rule = "strings"
)
