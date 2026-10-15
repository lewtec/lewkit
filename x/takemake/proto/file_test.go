package proto

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

func TestRoundTripFields(t *testing.T) {
	t.Parallel()
	want := sampleFile()
	got, err := Take(strings.NewReader(sampleText))
	require.NoError(t, err)
	require.Equal(t, "proto3", got.Syntax)
	require.Equal(t, "example.v1", got.Package)
	require.Equal(t, []Import{
		{Path: "other.proto"},
		{Path: "public.proto", Kind: ImportPublic},
		{Path: "weak.proto", Kind: ImportWeak},
	}, got.Imports)
	require.NotEmpty(t, got.Options)
	require.Equal(t, "Person", got.Messages[0].Name)
	name := got.Messages[0].Fields[0]
	require.Equal(t, "name", name.Name)
	require.Equal(t, 1, name.Number)
	require.Equal(t, TypeString, name.Type)
	require.Equal(t, LabelOptional, name.Label)
	require.NotEmpty(t, got.Messages[0].Enums)
	require.Equal(t, "Status", got.Enums[0].Name)
	require.Equal(t, "Greeter", got.Services[0].Name)
	require.Equal(t, want, got)

	var buf bytes.Buffer
	require.NoError(t, got.Make(&buf))
	again, err := Take(&buf)
	require.NoError(t, err)
	require.Equal(t, got, again)
}

func TestRoundTripProto2(t *testing.T) {
	t.Parallel()
	const text = "syntax = \"proto2\";\n" +
		"message M {\n" +
		"  required string a = 1 [default = \"x\"];\n" +
		"  optional int32 b = 2;\n" +
		"}\n"
	want := NewFile().SetSyntax("proto2")
	message := want.Message("M")
	message.Field("a", 1, TypeString).Required().Option("default", String("x"))
	message.Field("b", 2, TypeInt32).Optional()
	got, err := Take(strings.NewReader(text))
	require.NoError(t, err)
	require.Equal(t, want, got)
	roundTrip(t, text)
}

func TestRoundTripNumbersAndStrings(t *testing.T) {
	t.Parallel()
	got, err := Take(strings.NewReader("message M {\n  int32 id = 0x10;\n  int32 mode = 010;\n}\noption s = \"a\\nb\" \"c\";\noption n = -1.5;\noption blank = {};\noption names = [\"a\", \"b\"];\n"))
	require.NoError(t, err)
	require.Equal(t, 16, got.Messages[0].Fields[0].Number)
	require.Equal(t, 8, got.Messages[0].Fields[1].Number)
	require.Equal(t, "a\nbc", got.Options[0].Value.Text)
	require.Equal(t, Float(-1.5), got.Options[1].Value)
	require.Equal(t, Aggregate(), got.Options[2].Value)
	require.Equal(t, List(String("a"), String("b")), got.Options[3].Value)
	roundTrip(t, "message M {\n  int32 id = 0x10;\n  int32 mode = 010;\n}\noption s = \"a\\nb\" \"c\";\noption n = -1.5;\noption blank = {};\noption names = [\"a\", \"b\"];\n")
}

func TestRoundTripONNX(t *testing.T) {
	t.Parallel()
	body, err := os.ReadFile(filepath.Join("..", "..", "ndarray", "onnx", "internal", "proto", "onnx.proto"))
	require.NoError(t, err)
	roundTrip(t, string(body))
}

func TestTakeRejects(t *testing.T) {
	t.Parallel()
	tests := []string{
		"syntax = \"proto3\";\nsyntax = \"proto2\";\n",
		"package a;\npackage b;\n",
		"message M { optional group Foo = 1 { int32 a = 2; } }\n",
		"extend Foo { optional int32 a = 1; }\n",
		"message M { extensions 1 to 10; }\n",
		"message M { int32 id = 0; }\n",
		"option s = \"unterminated;\n",
		"/* unterminated\n",
		"message M { int32 id = 1 }\n",
		"$\n",
	}
	for _, text := range tests {
		t.Run(text, func(t *testing.T) {
			t.Parallel()
			_, err := Take(strings.NewReader(text))
			require.ErrorIs(t, err, ErrSyntax)
			require.ErrorIs(t, err, errs.ErrSyntax)
		})
	}
}

func TestNilArguments(t *testing.T) {
	t.Parallel()
	_, err := Take(nil)
	require.ErrorIs(t, err, ErrValue)
	require.ErrorIs(t, err, errs.ErrInvalid)
	var file *File
	require.ErrorIs(t, file.Make(io.Discard), ErrValue)
	require.ErrorIs(t, NewFile().Make(nil), ErrValue)
	require.Nil(t, file.Message("M"))
	require.Nil(t, file.Enum("E"))
	require.Nil(t, file.Service("S"))
	got, err := Take(bytes.NewReader(nil))
	require.NoError(t, err)
	require.Equal(t, NewFile(), got)
	require.NoError(t, got.Make(io.Discard))
}

func TestMakeRejectsUnknownConstant(t *testing.T) {
	t.Parallel()
	file := NewFile().Option("broken", Value{})
	require.ErrorIs(t, file.Make(io.Discard), ErrValue)
}

func ExampleFile() {
	file := NewFile().SetSyntax("proto3").SetPackage("example.v1")
	person := file.Message("Person")
	person.Field("name", 1, TypeString)
	person.Field("email", 2, TypeString).Repeated()
	file.Service("Greeter").RPC("SayHello", "Person", "Person")
	if err := file.Make(os.Stdout); err != nil {
		fmt.Println(err)
	}
	// Output:
	// syntax = "proto3";
	//
	// package example.v1;
	//
	// message Person {
	//   string name = 1;
	//   repeated string email = 2;
	// }
	//
	// service Greeter {
	//   rpc SayHello (Person) returns (Person);
	// }
}

func roundTrip(t *testing.T, text string) {
	t.Helper()
	first, err := Take(strings.NewReader(text))
	require.NoError(t, err)
	var buf bytes.Buffer
	require.NoError(t, first.Make(&buf))
	second, err := Take(&buf)
	require.NoError(t, err)
	require.Equal(t, first, second)
}

func sampleFile() *File {
	file := NewFile().SetSyntax("proto3").SetPackage("example.v1")
	file.Import("other.proto")
	file.PublicImport("public.proto")
	file.WeakImport("weak.proto")
	file.Option("go_package", String("example.com/v1"))
	file.Option("java_multiple_files", Bool(true))
	file.Option("(google.api.http)", Aggregate(
		Option{Name: "get", Value: String("/v1/people/{person}")},
		Option{Name: "body", Value: String("*")},
	))
	file.Option("floats", Float(-1.5))
	file.Option("flags", Float(math.Inf(1)))
	person := file.Message("Person")
	person.Option("deprecated", Bool(true))
	person.Reserve(Exact(10), Span(12, 15), ThroughMax(20))
	person.ReservedNames("old_name", "legacy")
	kind := person.Enum("Kind")
	kind.Value("KIND_UNSPECIFIED", 0)
	kind.Value("KIND_OK", 1).Option("deprecated", Bool(true))
	person.Message("Nested").Field("data", 1, TypeBytes)
	person.Field("name", 1, TypeString).Optional().Option("json_name", String("personName"))
	person.Field("id", 2, TypeInt32)
	person.Field("email", 3, TypeString).Repeated()
	person.Map("projects", 4, TypeString, Named("Nested"))
	choice := person.Oneof("choice")
	choice.Field("nick", 5, TypeString)
	choice.Field("code", 6, TypeInt32).Option("deprecated", Bool(true))
	status := file.Enum("Status")
	status.Option("allow_alias", Bool(true))
	status.Value("STATUS_UNSPECIFIED", 0)
	status.Value("OK", 1)
	status.Value("FINE", 1)
	status.Reserve(Exact(9), Span(10, 12))
	status.ReservedNames("GONE")
	greeter := file.Service("Greeter")
	greeter.Option("deprecated", Bool(false))
	greeter.RPC("SayHello", "HelloRequest", "HelloReply")
	greeter.RPC("Chat", "Note", "Note").RequestStream().ResponseStream().Option("idempotency_level", Ident("NO_SIDE_EFFECTS"))
	return file
}

const sampleText = `
// Sample file. Comments are not part of the model.
syntax = "proto3";

package example.v1;

import "other.proto";
import public "public.proto";
import weak "weak.proto";

option go_package = "example.com/v1";
option java_multiple_files = true;
option (google.api.http) = {
  get: "/v1/people/{person}"
  body: "*"
};
option floats = -1.5;
option flags = inf;

message Person {
  option deprecated = true;
  reserved 10, 12 to 15, 20 to max;
  reserved "old_name", "legacy";
  enum Kind {
    KIND_UNSPECIFIED = 0;
    KIND_OK = 1 [deprecated = true];
  }
  message Nested {
    bytes data = 1;
  }
  optional string name = 1 [json_name = "personName"];
  int32 id = 2;
  repeated string email = 3;
  map<string, Nested> projects = 4;
  oneof choice {
    string nick = 5;
    int32 code = 6 [deprecated = true];
  }
}

enum Status {
  option allow_alias = true;
  STATUS_UNSPECIFIED = 0;
  OK = 1;
  FINE = 1;
  reserved 9, 10 to 12;
  reserved "GONE";
}

service Greeter {
  option deprecated = false;
  rpc SayHello (HelloRequest) returns (HelloReply);
  rpc Chat (stream Note) returns (stream Note) {
    option idempotency_level = NO_SIDE_EFFECTS;
  };
}
`
