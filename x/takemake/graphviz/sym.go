package graphviz

import syn "github.com/lewtec/lewkit/x/parse/graphviz"

// Term and Rule are the Graphviz tokens and rules. The grammar lives in x/parse/graphviz.
type (
	Term = syn.Term
	Rule = syn.Rule
)

const (
	TermStrict     = syn.TermStrict
	TermGraph      = syn.TermGraph
	TermDigraph    = syn.TermDigraph
	TermSubgraph   = syn.TermSubgraph
	TermNode       = syn.TermNode
	TermEdge       = syn.TermEdge
	TermIdentifier = syn.TermIdentifier
	TermNumber     = syn.TermNumber
	TermString     = syn.TermString
	TermHTMLLabel  = syn.TermHTMLLabel
	TermSkip       = syn.TermSkip

	RuleFile               = syn.RuleFile
	RuleTop                = syn.RuleTop
	RuleStatement          = syn.RuleStatement
	RuleAttributeStatement = syn.RuleAttributeStatement
	RuleLead               = syn.RuleLead
	RuleTail               = syn.RuleTail
	RuleList               = syn.RuleList
	RuleEdgeTail           = syn.RuleEdgeTail
	RuleSubgraphStatement  = syn.RuleSubgraphStatement
	RuleStatementTail      = syn.RuleStatementTail
	RuleBlock              = syn.RuleBlock
	RuleEnd                = syn.RuleEnd
	RuleIdentifiers        = syn.RuleIdentifiers
	RuleNodeIdentifier     = syn.RuleNodeIdentifier
	RulePort               = syn.RulePort
	RuleAttributeList      = syn.RuleAttributeList
	RuleMore               = syn.RuleMore
	RulePair               = syn.RulePair
	RuleAtom               = syn.RuleAtom
	RuleStrings            = syn.RuleStrings
)
