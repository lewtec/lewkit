package decomp

import (
	"context"
	"fmt"
	"strings"
)

const explainPrompt = "You explain one native function. Reply with one JSON object and no other text. " +
	"The object has a source field. source is C for this function."

// Explain asks the attached model for C for the function at address.
// The source is stored when the reply has a source field. Blocks are not compared.
func (artifact *Artifact) Explain(ctx context.Context, address uint64, config MatchConfig) (Result, error) {
	if artifact == nil || artifact.project == nil || artifact.project.model == nil {
		return Result{}, ErrNoModel
	}
	function := artifact.byAddress[address]
	if function == nil {
		return Result{}, fmt.Errorf("%w: 0x%x", ErrNotFound, address)
	}
	rounds := config.Rounds
	if rounds <= 0 {
		rounds = 1
	}
	listing, err := artifact.Listing(address)
	if err != nil {
		listing = ""
	}
	result := Result{Address: address, Name: function.Name, Signature: function.Signature}
	var last error
	for round := 1; round <= rounds; round++ {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		text, err := artifact.project.model.Complete(ctx, Prompt{
			System: explainPrompt,
			User:   "Explain this function.\n\n" + listing,
		})
		result.Rounds = round
		if err != nil {
			return result, err
		}
		parsed, err := parseCandidate(text)
		if err != nil {
			last = err
			result.Residuals = []string{err.Error()}
			continue
		}
		if strings.TrimSpace(parsed.Source) == "" {
			last = fmt.Errorf("explanation has no source")
			result.Residuals = []string{last.Error()}
			continue
		}
		previous := function.Source
		function.Source = parsed.Source
		if err := artifact.saveFunction(ctx, function); err != nil {
			function.Source = previous
			return result, err
		}
		result.Source = parsed.Source
		result.Matched = true
		result.Residuals = nil
		return result, nil
	}
	if last == nil {
		last = fmt.Errorf("explanation has no source")
	}
	return result, last
}
