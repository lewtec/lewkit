package enum

import (
	"bytes"
	"fmt"
	"go/format"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/dave/jennifer/jen"
	"github.com/lewtec/lewkit/x/generate"
)

const errsPath = "github.com/lewtec/lewkit/x/errs"

func render(file fileSpec) ([]byte, error) {
	out := jen.NewFile(file.pkg)
	header := "// " + generate.Notice("enum")
	if file.build != "" {
		header = file.build + "\n\n" + header
	}
	out.HeaderComment(header)
	out.Var().DefsFunc(func(g *jen.Group) {
		for _, spec := range file.enums {
			g.Comment(errorName(spec.name) + " means the text is not a known " + strings.TrimPrefix(phrase(spec.name), "unknown ") + ".")
			g.Id(errorName(spec.name)).Op("=").Qual(errsPath, "Wrap").Call(
				jen.Qual(errsPath, "ErrUnknown"),
				jen.Lit(phrase(spec.name)),
			)
		}
	})
	for _, spec := range file.enums {
		writeEnum(out, spec)
	}
	var buf bytes.Buffer
	if err := out.Render(&buf); err != nil {
		return nil, err
	}
	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrEnum, err)
	}
	return formatted, nil
}

func writeEnum(out *jen.File, spec enumSpec) {
	recv := recvName(spec.name)
	errName := errorName(spec.name)
	parse := parseName(spec.name)
	out.Comment("String is the text of a " + spec.name + ".")
	if spec.stringKind {
		out.Func().Params(jen.Id(recv).Id(spec.name)).Id("String").Params().String().Block(
			jen.Return(jen.String().Parens(jen.Id(recv))),
		)
	} else {
		fallback := ""
		if spec.hasInvalid {
			fallback = spec.invalid
		}
		out.Func().Params(jen.Id(recv).Id(spec.name)).Id("String").Params().String().Block(
			jen.Switch(jen.Id(recv)).BlockFunc(func(g *jen.Group) {
				for _, item := range spec.members {
					g.Case(jen.Id(item.name)).Block(jen.Return(jen.Lit(item.token)))
				}
				g.Default().Block(jen.Return(jen.Lit(fallback)))
			}),
		)
	}
	out.Comment("Values lists the " + spec.name + " values.")
	out.Func().Params(jen.Id(spec.name)).Id("Values").Params().Index().Id(spec.name).Block(
		jen.Return(jen.Index().Id(spec.name).ValuesFunc(func(g *jen.Group) {
			for _, item := range spec.members {
				g.Id(item.name)
			}
		})),
	)
	out.Comment("IsValid reports whether the value is a " + spec.name + ".")
	out.Func().Params(jen.Id(recv).Id(spec.name)).Id("IsValid").Params().Bool().Block(
		jen.Switch(jen.Id(recv)).Block(
			jen.CaseFunc(func(g *jen.Group) {
				for _, item := range spec.members {
					g.Id(item.name)
				}
			}).Block(jen.Return(jen.True())),
			jen.Default().Block(jen.Return(jen.False())),
		),
	)
	out.Comment(parse + " returns the " + spec.name + " named by text.")
	out.Func().Id(parse).Params(jen.Id("text").String()).Params(jen.Id(spec.name), jen.Error()).BlockFunc(func(g *jen.Group) {
		g.Id("text").Op("=").Qual("strings", "TrimSpace").Call(jen.Id("text"))
		if spec.empty != "" {
			g.If(jen.Id("text").Op("==").Lit("")).Block(
				jen.Return(jen.Id(memberNamed(spec, spec.empty)), jen.Nil()),
			)
		}
		switched := jen.Id("text")
		if spec.fold {
			g.Id("token").Op(":=").Qual("strings", "ToLower").Call(jen.Id("text"))
			switched = jen.Id("token")
		}
		g.Switch(switched).BlockFunc(func(sw *jen.Group) {
			for _, item := range spec.members {
				sw.CaseFunc(func(labels *jen.Group) {
					labels.Lit(caseLabel(item.token, spec.fold))
					for _, alias := range item.aliases {
						labels.Lit(caseLabel(alias, spec.fold))
					}
				}).Block(jen.Return(jen.Id(item.name), jen.Nil()))
			}
			sw.Default().Block(jen.Return(zeroValue(spec.stringKind), jen.Qual("fmt", "Errorf").Call(
				jen.Lit("%w: %q"),
				jen.Id(errName),
				jen.Id("text"),
			)))
		})
	})
	out.Comment("Parse sets the value from text.")
	out.Func().Params(jen.Id(recv).Op("*").Id(spec.name)).Id("Parse").Params(jen.Id("text").String()).Error().Block(
		jen.List(jen.Id("parsed"), jen.Id("err")).Op(":=").Id(parse).Call(jen.Id("text")),
		jen.If(jen.Id("err").Op("!=").Nil()).Block(jen.Return(jen.Id("err"))),
		jen.Op("*").Id(recv).Op("=").Id("parsed"),
		jen.Return(jen.Nil()),
	)
	out.Comment("ArgChoices lists the texts " + parse + " accepts.")
	out.Func().Params(jen.Id(spec.name)).Id("ArgChoices").Params().Index().String().Block(
		jen.Return(jen.Index().String().ValuesFunc(func(g *jen.Group) {
			for _, item := range spec.members {
				g.Lit(item.token)
				for _, alias := range item.aliases {
					g.Lit(alias)
				}
			}
		})),
	)
	out.Comment("MarshalText encodes the value as text.")
	out.Func().Params(jen.Id(recv).Id(spec.name)).Id("MarshalText").Params().Params(jen.Index().Byte(), jen.Error()).Block(
		jen.If(jen.Op("!").Id(recv).Dot("IsValid").Call()).Block(
			jen.Return(jen.Nil(), jen.Qual("fmt", "Errorf").Call(
				jen.Lit("%w: %q"),
				jen.Id(errName),
				jen.Id(recv).Dot("String").Call(),
			)),
		),
		jen.Return(jen.Index().Byte().Parens(jen.Id(recv).Dot("String").Call()), jen.Nil()),
	)
	out.Comment("UnmarshalText sets the value from text.")
	out.Func().Params(jen.Id(recv).Op("*").Id(spec.name)).Id("UnmarshalText").Params(jen.Id("text").Index().Byte()).Error().Block(
		jen.Return(jen.Id(recv).Dot("Parse").Call(jen.String().Parens(jen.Id("text")))),
	)
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

func zeroValue(stringKind bool) jen.Code {
	if stringKind {
		return jen.Lit("")
	}
	return jen.Lit(0)
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
