package llvm

// Term is one LLVM IR token. Keywords and punctuation are literals, not terms.
//
//lewkit:enum
type Term string

const (
	TermIdentifier              Term = "identifier"
	TermNumber                  Term = "number"
	TermString                  Term = "string"
	TermCString                 Term = "c_string"
	TermLocalReference          Term = "local_reference"
	TermGlobalReference         Term = "global_reference"
	TermLocalDefinition         Term = "local_definition"
	TermGlobalDefinition        Term = "global_definition"
	TermIntegerType             Term = "integer_type"
	TermComdatName              Term = "comdat_name"
	TermLabelName               Term = "label_name"
	TermExclamation             Term = "exclamation"
	TermMetadataExclamation     Term = "metadata_exclamation"
	TermAttributeGroupReference Term = "attribute_group_reference"
	TermBraced                  Term = "braced"
	TermEllipsis                Term = "ellipsis"
)

// Rule is one LLVM IR syntax rule. Hidden rules the builder invents are not members.
//
//lewkit:enum
type Rule string

const (
	RuleFile                    Rule = "file"
	RuleSource                  Rule = "source"
	RuleLayout                  Rule = "layout"
	RuleTripleDefinition        Rule = "triple_definition"
	RuleModuleAssembly          Rule = "module_assembly"
	RuleTypedef                 Rule = "typedef"
	RuleGlobalVariable          Rule = "global_variable"
	RuleFunction                Rule = "function"
	RuleHeader                  Rule = "header"
	RuleParameters              Rule = "parameters"
	RuleParameter               Rule = "parameter"
	RuleBody                    Rule = "body"
	RuleInstruction             Rule = "instruction"
	RuleType                    Rule = "value_type"
	RuleTypeWithoutStruct       Rule = "type_without_struct"
	RuleStar                    Rule = "star"
	RuleFunctionTail            Rule = "function_tail"
	RuleAddressSpace            Rule = "address_space"
	RuleWord                    Rule = "word"
	RuleAlignment               Rule = "alignment"
	RuleValue                   Rule = "value"
	RuleTypedValue              Rule = "typed_value"
	RuleOperand                 Rule = "operand"
	RuleOperandWithoutStruct    Rule = "operand_without_struct"
	RuleGetElementPointer       Rule = "get_element_pointer"
	RuleConstantExpression      Rule = "constant_expression"
	RuleAggregate               Rule = "aggregate"
	RuleMetadataNode            Rule = "metadata_node"
	RuleAttach                  Rule = "attach"
	RulePiece                   Rule = "piece"
	RuleSync                    Rule = "sync"
	RuleCallingConventionNumber Rule = "calling_convention_number"
	RuleCallBody                Rule = "call_body"
	RuleTopMetadata             Rule = "top_metadata"
	RuleAttributeGroup          Rule = "attribute_group"
	RuleComdatDefinition        Rule = "comdat_definition"
)
