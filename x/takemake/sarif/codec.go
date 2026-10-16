package sarif

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type logJSON struct {
	Version string     `json:"version"`
	Schema  string     `json:"$schema,omitempty"`
	Runs    *[]runJSON `json:"runs"`
}

type runJSON struct {
	Tool       toolJSON     `json:"tool"`
	ColumnKind ColumnKind   `json:"columnKind,omitempty"`
	Results    []resultJSON `json:"results,omitempty"`
}

type toolJSON struct {
	Driver driverJSON `json:"driver"`
}

type driverJSON struct {
	Name           string     `json:"name"`
	Version        string     `json:"version,omitempty"`
	InformationURI string     `json:"informationUri,omitempty"`
	Rules          []ruleJSON `json:"rules,omitempty"`
}

type ruleJSON struct {
	ID                   string             `json:"id"`
	Name                 string             `json:"name,omitempty"`
	ShortDescription     *messageJSON       `json:"shortDescription,omitempty"`
	FullDescription      *messageJSON       `json:"fullDescription,omitempty"`
	Help                 *messageJSON       `json:"help,omitempty"`
	HelpURI              string             `json:"helpUri,omitempty"`
	DefaultConfiguration *configurationJSON `json:"defaultConfiguration,omitempty"`
}

type configurationJSON struct {
	Level Level `json:"level,omitempty"`
}

type messageJSON struct {
	Text     string `json:"text,omitempty"`
	Markdown string `json:"markdown,omitempty"`
}

type resultJSON struct {
	RuleID    string         `json:"ruleId,omitempty"`
	RuleIndex *int           `json:"ruleIndex,omitempty"`
	Kind      Kind           `json:"kind,omitempty"`
	Level     Level          `json:"level,omitempty"`
	Message   messageJSON    `json:"message"`
	Locations []locationJSON `json:"locations,omitempty"`
	Fixes     []fixJSON      `json:"fixes,omitempty"`
}

type locationJSON struct {
	PhysicalLocation *physicalJSON `json:"physicalLocation,omitempty"`
	Message          *messageJSON  `json:"message,omitempty"`
}

type physicalJSON struct {
	ArtifactLocation *artifactJSON `json:"artifactLocation,omitempty"`
	Region           *regionJSON   `json:"region,omitempty"`
}

type artifactJSON struct {
	URI       string `json:"uri,omitempty"`
	URIBaseID string `json:"uriBaseId,omitempty"`
	Index     *int   `json:"index,omitempty"`
}

type regionJSON struct {
	StartLine   int          `json:"startLine,omitempty"`
	StartColumn int          `json:"startColumn,omitempty"`
	EndLine     int          `json:"endLine,omitempty"`
	EndColumn   int          `json:"endColumn,omitempty"`
	Snippet     *contentJSON `json:"snippet,omitempty"`
}

type contentJSON struct {
	Text string `json:"text"`
}

type fixJSON struct {
	Description     *messageJSON `json:"description,omitempty"`
	ArtifactChanges []changeJSON `json:"artifactChanges"`
}

type changeJSON struct {
	ArtifactLocation artifactJSON      `json:"artifactLocation"`
	Replacements     []replacementJSON `json:"replacements"`
}

type replacementJSON struct {
	DeletedRegion   regionJSON   `json:"deletedRegion"`
	InsertedContent *contentJSON `json:"insertedContent,omitempty"`
}

func decode(body []byte) (*File, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.DisallowUnknownFields()
	var raw logJSON
	if err := dec.Decode(&raw); err != nil {
		return nil, decodeErr(err)
	}
	var extra struct{}
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return nil, fmt.Errorf("%w: trailing json", ErrSyntax)
		}
		return nil, decodeErr(err)
	}
	if raw.Version != Version210 {
		return nil, fmt.Errorf("%w: version %q", ErrForm, raw.Version)
	}
	if raw.Runs == nil {
		return nil, fmt.Errorf("%w: runs", ErrSyntax)
	}
	file := &File{Version: raw.Version, Schema: raw.Schema}
	runs, err := runsFrom(*raw.Runs)
	if err != nil {
		return nil, err
	}
	file.Runs = runs
	return file, nil
}

func decodeErr(err error) error {
	if errorsIsEnum(err) {
		return err
	}
	if strings.Contains(err.Error(), "unknown field") {
		return fmt.Errorf("%w: %v", ErrForm, err)
	}
	return fmt.Errorf("%w: %v", ErrSyntax, err)
}

func errorsIsEnum(err error) bool {
	return errors.Is(err, ErrLevel) || errors.Is(err, ErrKind) || errors.Is(err, ErrColumnKind)
}

func runsFrom(raw []runJSON) ([]*Run, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	runs := make([]*Run, 0, len(raw))
	for _, item := range raw {
		run, err := runFrom(item)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	return runs, nil
}

func runFrom(raw runJSON) (*Run, error) {
	if raw.Tool.Driver.Name == "" {
		return nil, fmt.Errorf("%w: driver name", ErrSyntax)
	}
	driver, err := driverFrom(raw.Tool.Driver)
	if err != nil {
		return nil, err
	}
	results, err := resultsFrom(raw.Results)
	if err != nil {
		return nil, err
	}
	return &Run{Driver: driver, Column: raw.ColumnKind, Results: results}, nil
}

func driverFrom(raw driverJSON) (*Driver, error) {
	rules, err := rulesFrom(raw.Rules)
	if err != nil {
		return nil, err
	}
	return &Driver{
		Name:           raw.Name,
		Version:        raw.Version,
		InformationURI: raw.InformationURI,
		Rules:          rules,
	}, nil
}

func rulesFrom(raw []ruleJSON) ([]*Rule, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	rules := make([]*Rule, 0, len(raw))
	for _, item := range raw {
		if item.ID == "" {
			return nil, fmt.Errorf("%w: rule id", ErrSyntax)
		}
		rules = append(rules, &Rule{
			ID:               item.ID,
			Name:             item.Name,
			ShortDescription: messageFrom(item.ShortDescription),
			FullDescription:  messageFrom(item.FullDescription),
			Help:             messageFrom(item.Help),
			HelpURI:          item.HelpURI,
			Level:            levelFrom(item.DefaultConfiguration),
		})
	}
	return rules, nil
}

func messageFrom(raw *messageJSON) Message {
	if raw == nil {
		return Message{}
	}
	return Message{Text: raw.Text, Markdown: raw.Markdown}
}

func levelFrom(raw *configurationJSON) Level {
	if raw == nil {
		return 0
	}
	return raw.Level
}

func resultsFrom(raw []resultJSON) ([]*Result, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	results := make([]*Result, 0, len(raw))
	for _, item := range raw {
		result, err := resultFrom(item)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func resultFrom(raw resultJSON) (*Result, error) {
	message := messageFrom(&raw.Message)
	if message.Text == "" && message.Markdown == "" {
		return nil, fmt.Errorf("%w: message", ErrSyntax)
	}
	locations, err := locationsFrom(raw.Locations)
	if err != nil {
		return nil, err
	}
	fixes, err := fixesFrom(raw.Fixes)
	if err != nil {
		return nil, err
	}
	return &Result{
		RuleID:    raw.RuleID,
		RuleIndex: raw.RuleIndex,
		Kind:      raw.Kind,
		Level:     raw.Level,
		Message:   message,
		Locations: locations,
		Fixes:     fixes,
	}, nil
}

func locationsFrom(raw []locationJSON) ([]*Location, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	locations := make([]*Location, 0, len(raw))
	for _, item := range raw {
		locations = append(locations, &Location{
			URI:       artifactURI(item.PhysicalLocation),
			URIBaseID: artifactBase(item.PhysicalLocation),
			Index:     artifactIndex(item.PhysicalLocation),
			Region:    regionFrom(item.PhysicalLocation),
			Message:   messageFrom(item.Message),
		})
	}
	return locations, nil
}

func artifactURI(raw *physicalJSON) string {
	if raw == nil || raw.ArtifactLocation == nil {
		return ""
	}
	return raw.ArtifactLocation.URI
}

func artifactBase(raw *physicalJSON) string {
	if raw == nil || raw.ArtifactLocation == nil {
		return ""
	}
	return raw.ArtifactLocation.URIBaseID
}

func artifactIndex(raw *physicalJSON) *int {
	if raw == nil || raw.ArtifactLocation == nil {
		return nil
	}
	return raw.ArtifactLocation.Index
}

func regionFrom(raw *physicalJSON) Region {
	if raw == nil || raw.Region == nil {
		return Region{}
	}
	return regionValue(*raw.Region)
}

func regionValue(raw regionJSON) Region {
	snippet := ""
	if raw.Snippet != nil {
		snippet = raw.Snippet.Text
	}
	return Region{
		StartLine:   raw.StartLine,
		StartColumn: raw.StartColumn,
		EndLine:     raw.EndLine,
		EndColumn:   raw.EndColumn,
		Snippet:     snippet,
	}
}

func fixesFrom(raw []fixJSON) ([]*Fix, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	fixes := make([]*Fix, 0, len(raw))
	for _, item := range raw {
		fix, err := fixFrom(item)
		if err != nil {
			return nil, err
		}
		fixes = append(fixes, fix)
	}
	return fixes, nil
}

func fixFrom(raw fixJSON) (*Fix, error) {
	if len(raw.ArtifactChanges) == 0 {
		return nil, fmt.Errorf("%w: artifact changes", ErrSyntax)
	}
	changes := make([]*Change, 0, len(raw.ArtifactChanges))
	for _, item := range raw.ArtifactChanges {
		change, err := changeFrom(item)
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}
	return &Fix{Description: messageFrom(raw.Description), Changes: changes}, nil
}

func changeFrom(raw changeJSON) (*Change, error) {
	if raw.ArtifactLocation.URI == "" {
		return nil, fmt.Errorf("%w: artifact uri", ErrSyntax)
	}
	if len(raw.Replacements) == 0 {
		return nil, fmt.Errorf("%w: replacements", ErrSyntax)
	}
	replacements := make([]*Replacement, 0, len(raw.Replacements))
	for _, item := range raw.Replacements {
		replacement, err := replacementFrom(item)
		if err != nil {
			return nil, err
		}
		replacements = append(replacements, replacement)
	}
	return &Change{
		URI:          raw.ArtifactLocation.URI,
		URIBaseID:    raw.ArtifactLocation.URIBaseID,
		Index:        raw.ArtifactLocation.Index,
		Replacements: replacements,
	}, nil
}

func replacementFrom(raw replacementJSON) (*Replacement, error) {
	replacement := &Replacement{Region: regionValue(raw.DeletedRegion)}
	if raw.InsertedContent != nil {
		text := raw.InsertedContent.Text
		replacement.Insert = &text
	}
	return replacement, nil
}

func encode(file *File) (logJSON, error) {
	if file.Version != Version210 {
		return logJSON{}, fmt.Errorf("%w: version %q", ErrForm, file.Version)
	}
	runs := make([]runJSON, 0, len(file.Runs))
	for _, run := range file.Runs {
		encoded, err := runToJSON(run)
		if err != nil {
			return logJSON{}, err
		}
		runs = append(runs, encoded)
	}
	return logJSON{Version: file.Version, Schema: file.Schema, Runs: &runs}, nil
}

func runToJSON(run *Run) (runJSON, error) {
	if run == nil {
		return runJSON{}, fmt.Errorf("%w: run", ErrValue)
	}
	if run.Driver == nil || run.Driver.Name == "" {
		return runJSON{}, fmt.Errorf("%w: driver name", ErrValue)
	}
	driver, err := driverToJSON(run.Driver)
	if err != nil {
		return runJSON{}, err
	}
	results, err := resultsToJSON(run.Results)
	if err != nil {
		return runJSON{}, err
	}
	return runJSON{
		Tool:       toolJSON{Driver: driver},
		ColumnKind: run.Column,
		Results:    results,
	}, nil
}

func driverToJSON(driver *Driver) (driverJSON, error) {
	rules, err := rulesToJSON(driver.Rules)
	if err != nil {
		return driverJSON{}, err
	}
	return driverJSON{
		Name:           driver.Name,
		Version:        driver.Version,
		InformationURI: driver.InformationURI,
		Rules:          rules,
	}, nil
}

func rulesToJSON(rules []*Rule) ([]ruleJSON, error) {
	if len(rules) == 0 {
		return nil, nil
	}
	out := make([]ruleJSON, 0, len(rules))
	for _, rule := range rules {
		if rule == nil {
			return nil, fmt.Errorf("%w: rule", ErrValue)
		}
		if rule.ID == "" {
			return nil, fmt.Errorf("%w: rule id", ErrValue)
		}
		out = append(out, ruleJSON{
			ID:                   rule.ID,
			Name:                 rule.Name,
			ShortDescription:     messagePtr(rule.ShortDescription),
			FullDescription:      messagePtr(rule.FullDescription),
			Help:                 messagePtr(rule.Help),
			HelpURI:              rule.HelpURI,
			DefaultConfiguration: configurationPtr(rule.Level),
		})
	}
	return out, nil
}

func messageToJSON(message Message) messageJSON {
	return messageJSON{Text: message.Text, Markdown: message.Markdown}
}

func messagePtr(message Message) *messageJSON {
	if message.Text == "" && message.Markdown == "" {
		return nil
	}
	encoded := messageToJSON(message)
	return &encoded
}

func configurationPtr(level Level) *configurationJSON {
	if level == 0 {
		return nil
	}
	return &configurationJSON{Level: level}
}

func resultsToJSON(results []*Result) ([]resultJSON, error) {
	if len(results) == 0 {
		return nil, nil
	}
	out := make([]resultJSON, 0, len(results))
	for _, result := range results {
		encoded, err := resultToJSON(result)
		if err != nil {
			return nil, err
		}
		out = append(out, encoded)
	}
	return out, nil
}

func resultToJSON(result *Result) (resultJSON, error) {
	if result == nil {
		return resultJSON{}, fmt.Errorf("%w: result", ErrValue)
	}
	if result.Message.Text == "" && result.Message.Markdown == "" {
		return resultJSON{}, fmt.Errorf("%w: message", ErrValue)
	}
	locations, err := locationsToJSON(result.Locations)
	if err != nil {
		return resultJSON{}, err
	}
	fixes, err := fixesToJSON(result.Fixes)
	if err != nil {
		return resultJSON{}, err
	}
	return resultJSON{
		RuleID:    result.RuleID,
		RuleIndex: result.RuleIndex,
		Kind:      result.Kind,
		Level:     result.Level,
		Message:   messageToJSON(result.Message),
		Locations: locations,
		Fixes:     fixes,
	}, nil
}

func locationsToJSON(locations []*Location) ([]locationJSON, error) {
	if len(locations) == 0 {
		return nil, nil
	}
	out := make([]locationJSON, 0, len(locations))
	for _, location := range locations {
		if location == nil {
			return nil, fmt.Errorf("%w: location", ErrValue)
		}
		out = append(out, locationJSON{
			PhysicalLocation: physicalPtr(location.URI, location.URIBaseID, location.Index, location.Region),
			Message:          messagePtr(location.Message),
		})
	}
	return out, nil
}

func regionToJSON(region Region) regionJSON {
	out := regionJSON{
		StartLine:   region.StartLine,
		StartColumn: region.StartColumn,
		EndLine:     region.EndLine,
		EndColumn:   region.EndColumn,
	}
	if region.Snippet != "" {
		out.Snippet = &contentJSON{Text: region.Snippet}
	}
	return out
}

func regionPtr(region Region) *regionJSON {
	out := regionToJSON(region)
	if out == (regionJSON{}) {
		return nil
	}
	return &out
}

func physicalPtr(uri, base string, index *int, region Region) *physicalJSON {
	artifact := artifactPtr(uri, base, index)
	span := regionPtr(region)
	if artifact == nil && span == nil {
		return nil
	}
	return &physicalJSON{ArtifactLocation: artifact, Region: span}
}

func artifactPtr(uri, base string, index *int) *artifactJSON {
	if uri == "" && base == "" && index == nil {
		return nil
	}
	return &artifactJSON{URI: uri, URIBaseID: base, Index: index}
}

func fixesToJSON(fixes []*Fix) ([]fixJSON, error) {
	if len(fixes) == 0 {
		return nil, nil
	}
	out := make([]fixJSON, 0, len(fixes))
	for _, fix := range fixes {
		encoded, err := fixToJSON(fix)
		if err != nil {
			return nil, err
		}
		out = append(out, encoded)
	}
	return out, nil
}

func fixToJSON(fix *Fix) (fixJSON, error) {
	if fix == nil {
		return fixJSON{}, fmt.Errorf("%w: fix", ErrValue)
	}
	if len(fix.Changes) == 0 {
		return fixJSON{}, fmt.Errorf("%w: artifact changes", ErrValue)
	}
	changes := make([]changeJSON, 0, len(fix.Changes))
	for _, change := range fix.Changes {
		encoded, err := changeToJSON(change)
		if err != nil {
			return fixJSON{}, err
		}
		changes = append(changes, encoded)
	}
	return fixJSON{Description: messagePtr(fix.Description), ArtifactChanges: changes}, nil
}

func changeToJSON(change *Change) (changeJSON, error) {
	if change == nil {
		return changeJSON{}, fmt.Errorf("%w: change", ErrValue)
	}
	if change.URI == "" {
		return changeJSON{}, fmt.Errorf("%w: artifact uri", ErrValue)
	}
	if len(change.Replacements) == 0 {
		return changeJSON{}, fmt.Errorf("%w: replacements", ErrValue)
	}
	replacements := make([]replacementJSON, 0, len(change.Replacements))
	for _, replacement := range change.Replacements {
		if replacement == nil {
			return changeJSON{}, fmt.Errorf("%w: replacement", ErrValue)
		}
		encoded := replacementJSON{DeletedRegion: regionToJSON(replacement.Region)}
		if replacement.Insert != nil {
			encoded.InsertedContent = &contentJSON{Text: *replacement.Insert}
		}
		replacements = append(replacements, encoded)
	}
	return changeJSON{
		ArtifactLocation: artifactJSON{URI: change.URI, URIBaseID: change.URIBaseID, Index: change.Index},
		Replacements:     replacements,
	}, nil
}

func writeJSON(w io.Writer, payload logJSON) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(payload); err != nil {
		return fmt.Errorf("%w: %w", ErrValue, err)
	}
	return nil
}
