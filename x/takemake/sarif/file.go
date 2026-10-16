package sarif

import (
	"fmt"
	"io"
)

// Version210 is the SARIF version this package reads and writes.
const Version210 = "2.1.0"

// SchemaURI is the SARIF 2.1.0 JSON schema used by x/text/report.
const SchemaURI = "https://json.schemastore.org/sarif-2.1.0.json"

// File is one SARIF log.
type File struct {
	Version string
	Schema  string
	Runs    []*Run
}

// Run is one analysis run: the driver, its rules, and the results.
type Run struct {
	Driver  *Driver
	Column  ColumnKind
	Results []*Result
}

// Driver is tool.driver.
type Driver struct {
	Name           string
	Version        string
	InformationURI string
	Rules          []*Rule
}

// Rule is one reporting descriptor.
// Level is defaultConfiguration.level.
type Rule struct {
	ID               string
	Name             string
	ShortDescription Message
	FullDescription  Message
	Help             Message
	HelpURI          string
	Level            Level
}

// Message is a SARIF message string.
// Text is the plain form. Markdown is the optional rich form.
type Message struct {
	Text     string
	Markdown string
}

// Result is one finding.
// RuleIndex is the driver's rule index when the pointer is set, including
// zero and negative one. A nil pointer omits the property.
type Result struct {
	RuleID    string
	RuleIndex *int
	Kind      Kind
	Level     Level
	Message   Message
	Locations []*Location
	Fixes     []*Fix
}

// Location is one physical location and an optional message.
// URI is artifactLocation.uri.
// Index is artifactLocation.index when the pointer is set, including zero.
type Location struct {
	URI       string
	URIBaseID string
	Index     *int
	Region    Region
	Message   Message
}

// Region is a line and column span. Columns follow the run's ColumnKind.
// Line and column numbers are 1-based. Zero omits that edge.
// Snippet is the optional text inside the region.
type Region struct {
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
	Snippet     string
}

// Fix is one proposed edit.
type Fix struct {
	Description Message
	Changes     []*Change
}

// Change is one artifact's replacements.
// URI is artifactLocation.uri.
// Index is artifactLocation.index when the pointer is set, including zero.
type Change struct {
	URI          string
	URIBaseID    string
	Index        *int
	Replacements []*Replacement
}

// Replacement deletes Region and, when Insert is set, writes that text.
// A non-nil empty Insert is a replacement with an empty string.
// A nil Insert deletes the region and omits insertedContent.
type Replacement struct {
	Region Region
	Insert *string
}

// New returns an empty log at version 2.1.0.
func New() *File {
	return &File{Version: Version210}
}

// SetVersion sets the log version.
func (file *File) SetVersion(version string) *File {
	if file == nil {
		return nil
	}
	file.Version = version
	return file
}

// SetSchema sets the $schema URI.
func (file *File) SetSchema(uri string) *File {
	if file == nil {
		return nil
	}
	file.Schema = uri
	return file
}

// Run appends a run and returns it.
func (file *File) Run() *Run {
	if file == nil {
		return nil
	}
	run := &Run{}
	file.Runs = append(file.Runs, run)
	return run
}

// Tool sets tool.driver.name and returns the driver.
func (run *Run) Tool(name string) *Driver {
	if run == nil {
		return nil
	}
	driver := run.ensureDriver()
	driver.Name = name
	return driver
}

// SetColumn sets the run's column unit.
func (run *Run) SetColumn(kind ColumnKind) *Run {
	if run == nil {
		return nil
	}
	run.Column = kind
	return run
}

// Rule appends a driver rule and returns it.
func (run *Run) Rule(id string) *Rule {
	if run == nil {
		return nil
	}
	rule := &Rule{ID: id}
	driver := run.ensureDriver()
	driver.Rules = append(driver.Rules, rule)
	return rule
}

// Result appends a result for ruleID and returns it.
func (run *Run) Result(ruleID string) *Result {
	if run == nil {
		return nil
	}
	result := &Result{RuleID: ruleID}
	run.Results = append(run.Results, result)
	return result
}

func (run *Run) ensureDriver() *Driver {
	if run.Driver == nil {
		run.Driver = &Driver{}
	}
	return run.Driver
}

// SetName sets the driver name.
func (driver *Driver) SetName(name string) *Driver {
	if driver == nil {
		return nil
	}
	driver.Name = name
	return driver
}

// SetVersion sets the driver version.
func (driver *Driver) SetVersion(version string) *Driver {
	if driver == nil {
		return nil
	}
	driver.Version = version
	return driver
}

// SetInformationURI sets the driver information URI.
func (driver *Driver) SetInformationURI(uri string) *Driver {
	if driver == nil {
		return nil
	}
	driver.InformationURI = uri
	return driver
}

// SetName sets the rule name.
func (rule *Rule) SetName(name string) *Rule {
	if rule == nil {
		return nil
	}
	rule.Name = name
	return rule
}

// SetShortDescription sets shortDescription.text.
func (rule *Rule) SetShortDescription(text string) *Rule {
	if rule == nil {
		return nil
	}
	rule.ShortDescription.Text = text
	return rule
}

// SetFullDescription sets fullDescription.text.
func (rule *Rule) SetFullDescription(text string) *Rule {
	if rule == nil {
		return nil
	}
	rule.FullDescription.Text = text
	return rule
}

// SetHelp sets help.text.
func (rule *Rule) SetHelp(text string) *Rule {
	if rule == nil {
		return nil
	}
	rule.Help.Text = text
	return rule
}

// SetHelpURI sets the rule help URI.
func (rule *Rule) SetHelpURI(uri string) *Rule {
	if rule == nil {
		return nil
	}
	rule.HelpURI = uri
	return rule
}

// SetLevel sets defaultConfiguration.level.
func (rule *Rule) SetLevel(level Level) *Rule {
	if rule == nil {
		return nil
	}
	rule.Level = level
	return rule
}

// SetRuleIndex sets the rule index, including zero and negative one.
func (result *Result) SetRuleIndex(index int) *Result {
	if result == nil {
		return nil
	}
	result.RuleIndex = &index
	return result
}

// SetKind sets why the result was produced.
func (result *Result) SetKind(kind Kind) *Result {
	if result == nil {
		return nil
	}
	result.Kind = kind
	return result
}

// SetLevel sets the result level.
func (result *Result) SetLevel(level Level) *Result {
	if result == nil {
		return nil
	}
	result.Level = level
	return result
}

// SetMessage sets message.text.
func (result *Result) SetMessage(text string) *Result {
	if result == nil {
		return nil
	}
	result.Message.Text = text
	return result
}

// SetMarkdown sets message.markdown.
func (result *Result) SetMarkdown(markdown string) *Result {
	if result == nil {
		return nil
	}
	result.Message.Markdown = markdown
	return result
}

// Location appends a physical location and returns it.
func (result *Result) Location(uri string) *Location {
	if result == nil {
		return nil
	}
	location := &Location{URI: uri}
	result.Locations = append(result.Locations, location)
	return location
}

// Fix appends a fix and returns it.
func (result *Result) Fix() *Fix {
	if result == nil {
		return nil
	}
	fix := &Fix{}
	result.Fixes = append(result.Fixes, fix)
	return fix
}

// SetURIBase sets artifactLocation.uriBaseId.
func (location *Location) SetURIBase(id string) *Location {
	if location == nil {
		return nil
	}
	location.URIBaseID = id
	return location
}

// SetIndex sets artifactLocation.index, including zero.
func (location *Location) SetIndex(index int) *Location {
	if location == nil {
		return nil
	}
	location.Index = &index
	return location
}

// SetMessage sets the location message text.
func (location *Location) SetMessage(text string) *Location {
	if location == nil {
		return nil
	}
	location.Message.Text = text
	return location
}

// SetStartLine sets the region's start line.
func (location *Location) SetStartLine(line int) *Location {
	if location == nil {
		return nil
	}
	location.Region.StartLine = line
	return location
}

// SetStartColumn sets the region's start column.
func (location *Location) SetStartColumn(column int) *Location {
	if location == nil {
		return nil
	}
	location.Region.StartColumn = column
	return location
}

// SetEndLine sets the region's end line.
func (location *Location) SetEndLine(line int) *Location {
	if location == nil {
		return nil
	}
	location.Region.EndLine = line
	return location
}

// SetEndColumn sets the region's end column.
func (location *Location) SetEndColumn(column int) *Location {
	if location == nil {
		return nil
	}
	location.Region.EndColumn = column
	return location
}

// SetSnippet sets the region's snippet text.
func (location *Location) SetSnippet(text string) *Location {
	if location == nil {
		return nil
	}
	location.Region.Snippet = text
	return location
}

// SetDescription sets the fix description text.
func (fix *Fix) SetDescription(text string) *Fix {
	if fix == nil {
		return nil
	}
	fix.Description.Text = text
	return fix
}

// Change appends an artifact change and returns it.
func (fix *Fix) Change(uri string) *Change {
	if fix == nil {
		return nil
	}
	change := &Change{URI: uri}
	fix.Changes = append(fix.Changes, change)
	return change
}

// SetURIBase sets artifactLocation.uriBaseId.
func (change *Change) SetURIBase(id string) *Change {
	if change == nil {
		return nil
	}
	change.URIBaseID = id
	return change
}

// SetIndex sets artifactLocation.index, including zero.
func (change *Change) SetIndex(index int) *Change {
	if change == nil {
		return nil
	}
	change.Index = &index
	return change
}

// Replace appends a replacement and returns it.
func (change *Change) Replace() *Replacement {
	if change == nil {
		return nil
	}
	replacement := &Replacement{}
	change.Replacements = append(change.Replacements, replacement)
	return replacement
}

// SetStartLine sets the deleted region's start line.
func (replacement *Replacement) SetStartLine(line int) *Replacement {
	if replacement == nil {
		return nil
	}
	replacement.Region.StartLine = line
	return replacement
}

// SetStartColumn sets the deleted region's start column.
func (replacement *Replacement) SetStartColumn(column int) *Replacement {
	if replacement == nil {
		return nil
	}
	replacement.Region.StartColumn = column
	return replacement
}

// SetEndLine sets the deleted region's end line.
func (replacement *Replacement) SetEndLine(line int) *Replacement {
	if replacement == nil {
		return nil
	}
	replacement.Region.EndLine = line
	return replacement
}

// SetEndColumn sets the deleted region's end column.
func (replacement *Replacement) SetEndColumn(column int) *Replacement {
	if replacement == nil {
		return nil
	}
	replacement.Region.EndColumn = column
	return replacement
}

// SetInsert sets insertedContent.text, including the empty string.
func (replacement *Replacement) SetInsert(text string) *Replacement {
	if replacement == nil {
		return nil
	}
	replacement.Insert = &text
	return replacement
}

// Take reads a SARIF 2.1.0 log into a [File].
func Take(r io.Reader) (*File, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: reader", ErrValue)
	}
	body, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrValue, err)
	}
	return decode(body)
}

// Make writes the log as indented JSON.
func (file *File) Make(w io.Writer) error {
	if file == nil {
		return fmt.Errorf("%w: file", ErrValue)
	}
	if w == nil {
		return fmt.Errorf("%w: writer", ErrValue)
	}
	payload, err := encode(file)
	if err != nil {
		return err
	}
	return writeJSON(w, payload)
}
