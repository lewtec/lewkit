package onnx

import (
	"fmt"
	"sort"
	"strings"
)

const (
	StatusPassed      = "passed"
	StatusSkipped     = "skipped"
	StatusFailed      = "failed"
	StatusImplemented = "implemented"
	StatusMissing     = "missing"
)

// CaseResult is one node-test outcome.
type CaseResult struct {
	Name      string
	Operators []string
	Status    string
	Detail    string
}

// OperatorCoverage is one ai.onnx name versus node tests.
type OperatorCoverage struct {
	Name   string
	Status string
}

// Coverage is catalog vs implemented vs node-test results.
type Coverage struct {
	Catalog     int
	Implemented int
	Passed      int
	Skipped     int
	Failed      int
	Operators   []OperatorCoverage
}

// CoverageFromResults scores node-test results against the ai.onnx catalog.
func CoverageFromResults(results []CaseResult) Coverage {
	implemented := map[string]struct{}{}
	for _, op := range implementedOperators() {
		implemented[op] = struct{}{}
	}
	passed, skipped, failed := map[string]struct{}{}, map[string]struct{}{}, map[string]struct{}{}
	for _, result := range results {
		if len(result.Operators) != 1 {
			continue
		}
		for _, op := range result.Operators {
			switch result.Status {
			case StatusPassed:
				passed[op] = struct{}{}
			case StatusFailed:
				failed[op] = struct{}{}
			case StatusSkipped:
				skipped[op] = struct{}{}
			}
		}
	}
	out := Coverage{
		Catalog:     len(onnxOperators),
		Implemented: len(implemented),
	}
	for _, name := range onnxOperators {
		_, have := implemented[name]
		_, didPass := passed[name]
		_, didFail := failed[name]
		_, didSkip := skipped[name]
		status := StatusMissing
		switch {
		case didFail:
			status = StatusFailed
			out.Failed++
		case didPass:
			status = StatusPassed
			out.Passed++
		case didSkip:
			status = StatusSkipped
			out.Skipped++
		case have:
			status = StatusImplemented
		}
		out.Operators = append(out.Operators, OperatorCoverage{Name: name, Status: status})
	}
	return out
}

func (c Coverage) CatalogFraction() float64 {
	if c.Catalog == 0 {
		return 0
	}
	return float64(c.Passed) / float64(c.Catalog)
}

func (c Coverage) ImplementedFraction() float64 {
	if c.Implemented == 0 {
		return 0
	}
	return float64(c.Passed) / float64(c.Implemented)
}

func (c Coverage) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "ai.onnx catalog %d\n", c.Catalog)
	fmt.Fprintf(&b, "implemented %d\n", c.Implemented)
	fmt.Fprintf(&b, "operators passed %d skipped %d failed %d\n", c.Passed, c.Skipped, c.Failed)
	fmt.Fprintf(&b, "catalog coverage %.1f%% (%d/%d)\n", 100*c.CatalogFraction(), c.Passed, c.Catalog)
	fmt.Fprintf(&b, "implemented coverage %.1f%% (%d/%d)\n", 100*c.ImplementedFraction(), c.Passed, c.Implemented)
	by := map[string][]string{}
	for _, op := range c.Operators {
		if op.Status != StatusMissing {
			by[op.Status] = append(by[op.Status], op.Name)
		}
	}
	for _, status := range []string{StatusPassed, StatusFailed, StatusSkipped, StatusImplemented} {
		names := by[status]
		sort.Strings(names)
		if len(names) == 0 {
			continue
		}
		fmt.Fprintf(&b, "%s: %s\n", status, strings.Join(names, " "))
	}
	return b.String()
}
