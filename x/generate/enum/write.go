package enum

import (
	"bytes"
	"fmt"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/lewtec/lewkit/x/generate"
	"github.com/lewtec/lewkit/x/takemake/golang"
)

const errsPath = "github.com/lewtec/lewkit/x/errs"

func render(file fileSpec) ([]byte, error) {
	out := golang.NewFile(file.pkg)
	header := "// " + generate.Notice("enum")
	if file.build != "" {
		header = file.build + "\n\n" + header
	}
	out.Header = header
	for _, spec := range file.enums {
		out.Var().Id(errorName(spec.name)).Comment(errorName(spec.name) + " means the text is not a known " + strings.TrimPrefix(phrase(spec.name), "unknown ") + ".").Values(
			golang.Qual(errsPath, "Wrap").Call(
				golang.Qual(errsPath, "ErrUnknown"),
				golang.Lit(phrase(spec.name)),
			),
		)
	}
	for _, spec := range file.enums {
		writeEnum(out, spec)
	}
	var buf bytes.Buffer
	if err := out.Make(&buf); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEnum, err)
	}
	return buf.Bytes(), nil
}

func writeEnum(out *golang.File, spec enumSpec) {
	recv := recvName(spec.name)
	errName := errorName(spec.name)
	parse := parseName(spec.name)
	out.Func().Comment("String is the text of a "+spec.name+".").
		Recv(recv, golang.Named(spec.name)).
		Id("String").
		Results(golang.NewField(golang.Named("string"))).
		Block(stringBody(spec, recv)...)
	out.Func().Comment("Values lists the "+spec.name+" values.").
		Recv("", golang.Named(spec.name)).
		Id("Values").
		Results(golang.NewField(golang.Slice(golang.Named(spec.name)))).
		Block(golang.Return(valueList(spec)))
	out.Func().Comment("IsValid reports whether the value is a "+spec.name+".").
		Recv(recv, golang.Named(spec.name)).
		Id("IsValid").
		Results(golang.NewField(golang.Named("bool"))).
		Block(golang.Switch(golang.Id(recv),
			golang.Case(memberIDs(spec), golang.Return(golang.Lit(true))),
			golang.Default(golang.Return(golang.Lit(false))),
		))
	out.Func().Comment(parse+" returns the "+spec.name+" named by text.").
		Id(parse).
		Params(golang.NewField(golang.Named("string"), "text")).
		Results(golang.NewField(golang.Named(spec.name)), golang.NewField(golang.Named("error"))).
		Block(parseBody(spec, errName)...)
	out.Func().Comment("Parse sets the value from text.").
		Recv(recv, golang.Pointer(golang.Named(spec.name))).
		Id("Parse").
		Params(golang.NewField(golang.Named("string"), "text")).
		Results(golang.NewField(golang.Named("error"))).
		Block(
			golang.Define([]*golang.Expr{golang.Id("parsed"), golang.Id("err")}, golang.Id(parse).Call(golang.Id("text"))),
			golang.If(golang.Binary(golang.OpNeq, golang.Id("err"), golang.Lit(nil)), golang.Return(golang.Id("err"))),
			golang.Assign(0, []*golang.Expr{golang.Unary(golang.OpMul, golang.Id(recv))}, golang.Id("parsed")),
			golang.Return(golang.Lit(nil)),
		)
	out.Func().Comment("ArgChoices lists the texts "+parse+" accepts.").
		Recv("", golang.Named(spec.name)).
		Id("ArgChoices").
		Results(golang.NewField(golang.Slice(golang.Named("string")))).
		Block(golang.Return(choiceList(spec)))
	out.Func().Comment("MarshalText encodes the value as text.").
		Recv(recv, golang.Named(spec.name)).
		Id("MarshalText").
		Results(golang.NewField(golang.Slice(golang.Named("byte"))), golang.NewField(golang.Named("error"))).
		Block(
			golang.If(golang.Unary(golang.OpNot, golang.Id(recv).Sel("IsValid").Call()),
				golang.Return(golang.Lit(nil), golang.Qual("fmt", "Errorf").Call(
					golang.Lit("%w: %q"),
					golang.Id(errName),
					golang.Id(recv).Sel("String").Call(),
				)),
			),
			golang.Return(golang.Slice(golang.Named("byte")).AsExpr().Call(golang.Id(recv).Sel("String").Call()), golang.Lit(nil)),
		)
	out.Func().Comment("UnmarshalText sets the value from text.").
		Recv(recv, golang.Pointer(golang.Named(spec.name))).
		Id("UnmarshalText").
		Params(golang.NewField(golang.Slice(golang.Named("byte")), "text")).
		Results(golang.NewField(golang.Named("error"))).
		Block(golang.Return(golang.Id(recv).Sel("Parse").Call(golang.Named("string").AsExpr().Call(golang.Id("text")))))
}

func stringBody(spec enumSpec, recv string) []golang.Code {
	if spec.stringKind {
		return []golang.Code{golang.Return(golang.Named("string").AsExpr().Call(golang.Id(recv)))}
	}
	fallback := ""
	if spec.hasInvalid {
		fallback = spec.invalid
	}
	cases := make([]*golang.Stmt, 0, len(spec.members)+1)
	for _, item := range spec.members {
		cases = append(cases, golang.Case([]*golang.Expr{golang.Id(item.name)}, golang.Return(golang.Lit(item.token))))
	}
	cases = append(cases, golang.Default(golang.Return(golang.Lit(fallback))))
	return []golang.Code{golang.Switch(golang.Id(recv), cases...)}
}

func parseBody(spec enumSpec, errName string) []golang.Code {
	body := []golang.Code{
		golang.Assign(0, []*golang.Expr{golang.Id("text")}, golang.Qual("strings", "TrimSpace").Call(golang.Id("text"))),
	}
	switched := golang.Id("text")
	if spec.empty != "" {
		body = append(body, golang.If(
			golang.Binary(golang.OpEql, golang.Id("text"), golang.Lit("")),
			golang.Return(golang.Id(memberNamed(spec, spec.empty)), golang.Lit(nil)),
		))
	}
	if spec.fold {
		body = append(body, golang.Define([]*golang.Expr{golang.Id("token")}, golang.Qual("strings", "ToLower").Call(golang.Id("text"))))
		switched = golang.Id("token")
	}
	cases := make([]*golang.Stmt, 0, len(spec.members)+1)
	for _, item := range spec.members {
		labels := []*golang.Expr{golang.Lit(caseLabel(item.token, spec.fold))}
		for _, alias := range item.aliases {
			labels = append(labels, golang.Lit(caseLabel(alias, spec.fold)))
		}
		cases = append(cases, golang.Case(labels, golang.Return(golang.Id(item.name), golang.Lit(nil))))
	}
	cases = append(cases, golang.Default(golang.Return(
		zeroValue(spec.stringKind),
		golang.Qual("fmt", "Errorf").Call(golang.Lit("%w: %q"), golang.Id(errName), golang.Id("text")),
	)))
	body = append(body, golang.Switch(switched, cases...))
	return body
}

func memberIDs(spec enumSpec) []*golang.Expr {
	ids := make([]*golang.Expr, len(spec.members))
	for i, item := range spec.members {
		ids[i] = golang.Id(item.name)
	}
	return ids
}

func valueList(spec enumSpec) *golang.Expr {
	elts := make([]*golang.Element, len(spec.members))
	for i, item := range spec.members {
		elts[i] = golang.Elem(golang.Id(item.name))
	}
	return golang.Composite(golang.Slice(golang.Named(spec.name)), elts...)
}

func choiceList(spec enumSpec) *golang.Expr {
	var elts []*golang.Element
	for _, item := range spec.members {
		elts = append(elts, golang.Elem(golang.Lit(item.token)))
		for _, alias := range item.aliases {
			elts = append(elts, golang.Elem(golang.Lit(alias)))
		}
	}
	return golang.Composite(golang.Slice(golang.Named("string")), elts...)
}

func memberNamed(spec enumSpec, tokenText string) string {
	for _, item := range spec.members {
		if item.token == tokenText {
			return item.name
		}
	}
	return spec.members[0].name
}

func caseLabel(text string, fold bool) string {
	if fold {
		return strings.ToLower(text)
	}
	return text
}

func zeroValue(stringKind bool) *golang.Expr {
	if stringKind {
		return golang.Lit("")
	}
	return golang.Lit(0)
}

func errorName(typeName string) string {
	if token.IsExported(typeName) {
		return "Err" + typeName
	}
	return "err" + exported(typeName)
}

func parseName(typeName string) string {
	if token.IsExported(typeName) {
		return "Parse" + typeName
	}
	return "parse" + exported(typeName)
}

func exported(name string) string {
	r, size := utf8.DecodeRuneInString(name)
	return string(unicode.ToUpper(r)) + name[size:]
}

func recvName(typeName string) string {
	r, _ := utf8.DecodeRuneInString(typeName)
	name := string(unicode.ToLower(r))
	if !token.IsIdentifier(name) || token.IsKeyword(name) {
		return "value"
	}
	return name
}

func phrase(name string) string {
	var b strings.Builder
	var prev rune
	for i, r := range name {
		if i > 0 && unicode.IsUpper(r) && (unicode.IsLower(prev) || unicode.IsDigit(prev)) {
			b.WriteByte(' ')
		}
		b.WriteRune(unicode.ToLower(r))
		prev = r
	}
	return "unknown " + b.String()
}
