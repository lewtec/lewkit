package ninja

import syn "github.com/lewtec/lewkit/x/parse/ninja"

// Term and Production are the ninja tokens and rules. The grammar lives in x/parse/ninja.
type (
	Term       = syn.Term
	Production = syn.Production
)

const (
	TermIdentifier     = syn.TermIdentifier
	TermBuild          = syn.TermBuild
	TermPool           = syn.TermPool
	TermRule           = syn.TermRule
	TermDefault        = syn.TermDefault
	TermInclude        = syn.TermInclude
	TermSubninja       = syn.TermSubninja
	TermEqual          = syn.TermEqual
	TermColon          = syn.TermColon
	TermPipe           = syn.TermPipe
	TermDoublePipe     = syn.TermDoublePipe
	TermPipeAtSign     = syn.TermPipeAtSign
	TermIndent         = syn.TermIndent
	TermNewline        = syn.TermNewline
	TermValue          = syn.TermValue
	TermPath           = syn.TermPath
	ProductionFile     = syn.ProductionFile
	ProductionVariable = syn.ProductionVariable
	ProductionRule     = syn.ProductionRule
	ProductionPool     = syn.ProductionPool
	ProductionEdge     = syn.ProductionEdge
	ProductionDefault  = syn.ProductionDefault
	ProductionInclude  = syn.ProductionInclude
	ProductionSubninja = syn.ProductionSubninja
	ProductionBinding  = syn.ProductionBinding
	ProductionOutputs  = syn.ProductionOutputs
	ProductionInputs   = syn.ProductionInputs
)
