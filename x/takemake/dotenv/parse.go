package dotenv

import (
	"errors"
	"fmt"
	"strings"

	xparse "github.com/lewtec/lewkit/x/parse"
	syntax "github.com/lewtec/lewkit/x/parse/dotenv"
)

func parse(text string) (*File, error) {
	text = strings.TrimPrefix(text, "\uFEFF")
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	model, err := syntax.Language()
	if err != nil {
		return nil, err
	}
	root, err := model.Parse([]byte(text))
	if err != nil {
		var pe *xparse.Error
		if errors.As(err, &pe) {
			return nil, lineError(ErrLine, pe.Line)
		}
		return nil, lineError(ErrLine, 1)
	}
	built, err := buildFile(root)
	if err != nil {
		return nil, err
	}
	file, ok := built.(*File)
	if !ok || file == nil {
		return nil, lineError(ErrLine, 1)
	}
	return file, nil
}

func buildFile(node xparse.SyntaxNode[syntax.Term, syntax.Rule]) (any, error) {
	file := New()
	var pending []string
	for i := 0; i < node.ChildCount(); i++ {
		child := node.Child(i)
		rule, ok := child.Rule()
		if !ok {
			continue
		}
		switch rule {
		case syntax.RuleBlank:
		case syntax.RuleNote:
			value, err := child.Apply()
			if err != nil {
				return nil, err
			}
			text, _ := value.(string)
			pending = append(pending, text)
		case syntax.RuleAssign:
			assignment := &Assignment{Quote: syntax.QuoteBare}
			if err := child.Fill(assignment); err != nil {
				return nil, err
			}
			if assignment.Key == "" {
				return nil, lineError(ErrKey, child.Line())
			}
			if len(pending) > 0 {
				assignment.Comment = joinComment(strings.Join(pending, "\n"), assignment.Comment)
				pending = nil
			}
			file.Assignments = append(file.Assignments, assignment)
		}
	}
	if len(pending) == 0 {
		return file, nil
	}
	note := strings.Join(pending, "\n")
	if len(file.Assignments) == 0 {
		file.Assignments = append(file.Assignments, &Assignment{Comment: note})
		return file, nil
	}
	last := file.Assignments[len(file.Assignments)-1]
	last.Comment = joinComment(last.Comment, note)
	return file, nil
}

func lineError(err error, line int) error {
	if errors.Is(err, ErrKey) {
		return fmt.Errorf("%w: line %d", ErrKey, line)
	}
	return fmt.Errorf("%w: line %d", ErrLine, line)
}

func joinComment(lead, inlineComment string) string {
	if lead == "" {
		return inlineComment
	}
	if inlineComment == "" {
		return lead
	}
	return lead + "\n" + inlineComment
}
