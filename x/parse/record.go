package parse

import (
	"reflect"

	"github.com/lewtec/lewkit/x/errs"
)

// Fill copies the fields named on node's rule into dest.
// dest is a pointer to a struct. A [Piece.Build] on a child rule runs first:
// a string fills the named field, and a struct copies each exported field
// whose name and type match dest.
func (node SyntaxNode[Term, Rule]) Fill(dest any) error {
	if !node.Valid() || node.model == nil {
		return errs.Wrap(errs.ErrInvalid, "parse: empty node")
	}
	value := reflect.ValueOf(dest)
	if value.Kind() != reflect.Pointer || value.IsNil() || value.Elem().Kind() != reflect.Struct {
		return errs.Wrap(errs.ErrInvalid, "parse: record is not a struct")
	}
	elem := value.Elem()
	specs := node.model.fields[node.Name()]
	used := make([]bool, node.ChildCount())
	for _, spec := range specs {
		for i := 0; i < node.ChildCount(); i++ {
			if used[i] {
				continue
			}
			child := node.Child(i)
			if spec.symbol != child.Name() {
				continue
			}
			used[i] = true
			switch spec.kind {
			case fieldFlag:
				if err := setBool(elem, spec.name); err != nil {
					return err
				}
			case fieldText:
				if err := node.model.setText(elem, spec.name, child); err != nil {
					return err
				}
			}
			break
		}
	}
	return nil
}

// Apply runs the [Piece.Build] function of node's rule.
func (node SyntaxNode[Term, Rule]) Apply() (any, error) {
	if !node.Valid() || node.model == nil {
		return nil, errs.Wrap(errs.ErrInvalid, "parse: empty node")
	}
	rule, ok := node.Rule()
	if !ok {
		return nil, errs.Wrap(errs.ErrInvalid, "parse: node is not a rule")
	}
	build, ok := node.model.builds[rule.String()]
	if !ok || build == nil {
		return nil, errs.Wrap(errs.ErrInvalid, "parse: rule "+rule.String()+" has no build")
	}
	return build(node)
}

func setBool(elem reflect.Value, name string) error {
	field := elem.FieldByName(name)
	if !field.IsValid() || !field.CanSet() || field.Kind() != reflect.Bool {
		return errs.Wrap(errs.ErrInvalid, "parse: record field "+name)
	}
	field.SetBool(true)
	return nil
}

func (model *Model[Term, Rule]) setText(elem reflect.Value, name string, child SyntaxNode[Term, Rule]) error {
	field := elem.FieldByName(name)
	if !field.IsValid() || !field.CanSet() {
		return errs.Wrap(errs.ErrInvalid, "parse: record field "+name)
	}
	if rule, ok := child.Rule(); ok {
		if build, ok := model.builds[rule.String()]; ok && build != nil {
			value, err := build(child)
			if err != nil {
				return err
			}
			return assignBuilt(elem, field, value)
		}
	}
	if field.Kind() != reflect.String {
		return errs.Wrap(errs.ErrInvalid, "parse: record field "+name)
	}
	field.SetString(child.Text())
	return nil
}

func assignBuilt(dest, field reflect.Value, value any) error {
	if value == nil {
		return nil
	}
	got := reflect.ValueOf(value)
	for got.Kind() == reflect.Pointer {
		if got.IsNil() {
			return nil
		}
		got = got.Elem()
	}
	if got.Kind() == reflect.String {
		if field.Kind() != reflect.String {
			return errs.Wrap(errs.ErrInvalid, "parse: record field "+field.Type().String())
		}
		field.SetString(got.String())
		return nil
	}
	if got.Kind() != reflect.Struct {
		return errs.Wrap(errs.ErrInvalid, "parse: build returned "+got.Kind().String())
	}
	for i := 0; i < got.NumField(); i++ {
		sf := got.Type().Field(i)
		if sf.PkgPath != "" {
			continue
		}
		target := dest.FieldByName(sf.Name)
		if !target.IsValid() || !target.CanSet() {
			continue
		}
		from := got.Field(i)
		if from.Type().AssignableTo(target.Type()) {
			target.Set(from)
		}
	}
	return nil
}
