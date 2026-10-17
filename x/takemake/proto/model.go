package proto

// File is one proto file.
type File struct {
	Syntax   string
	Package  string
	Imports  []Import
	Options  []Option
	Messages []*Message
	Enums    []*Enum
	Services []*Service
}

// Import is one import line.
type Import struct {
	Path string
	Kind ImportKind
}

// Option is one name and constant.
type Option struct {
	Name  string
	Value Value
}

// Message is one message, including nested messages and enums.
type Message struct {
	Name     string
	Options  []Option
	Reserved []Reserve
	Enums    []*Enum
	Messages []*Message
	Fields   []*Field
	Oneofs   []*Oneof
}

// Field is one message field. A map uses a map [Type] and no label.
type Field struct {
	Name    string
	Number  int
	Type    Type
	Label   Label
	Options []Option
}

// Oneof is one oneof group.
type Oneof struct {
	Name    string
	Options []Option
	Fields  []*Field
}

// Enum is one enum.
type Enum struct {
	Name     string
	Options  []Option
	Reserved []Reserve
	Values   []*EnumValue
}

// EnumValue is one enumerator.
type EnumValue struct {
	Name    string
	Number  int
	Options []Option
}

// Service is one service.
type Service struct {
	Name    string
	Options []Option
	Methods []*Method
}

// Method is one rpc.
type Method struct {
	Name         string
	Input        string
	Output       string
	ClientStream bool
	ServerStream bool
	Options      []Option
}

// Reserve is one reserved statement, either names or ranges.
type Reserve struct {
	Names  []string
	Ranges []Range
}

// Range is one reserved span. Exact numbers use equal ends.
// Max prints as "to max".
type Range struct {
	Start int
	End   int
	Max   bool
}

// NewFile returns an empty proto file.
func NewFile() *File {
	return &File{}
}

// SetSyntax sets the syntax string, such as proto3.
func (file *File) SetSyntax(syntax string) *File {
	if file == nil {
		return nil
	}
	file.Syntax = syntax
	return file
}

// SetPackage sets the package name.
func (file *File) SetPackage(name string) *File {
	if file == nil {
		return nil
	}
	file.Package = name
	return file
}

// Import stores a plain import.
func (file *File) Import(path string) *File {
	return file.addImport(path, 0)
}

// PublicImport stores import public.
func (file *File) PublicImport(path string) *File {
	return file.addImport(path, ImportPublic)
}

// WeakImport stores import weak.
func (file *File) WeakImport(path string) *File {
	return file.addImport(path, ImportWeak)
}

func (file *File) addImport(path string, kind ImportKind) *File {
	if file == nil {
		return nil
	}
	file.Imports = append(file.Imports, Import{Path: path, Kind: kind})
	return file
}

// Option stores a file option.
func (file *File) Option(name string, value Value) *File {
	if file == nil {
		return nil
	}
	file.Options = append(file.Options, Option{Name: name, Value: value})
	return file
}

// Message stores a top-level message and returns it.
func (file *File) Message(name string) *Message {
	if file == nil {
		return nil
	}
	message := &Message{Name: name}
	file.Messages = append(file.Messages, message)
	return message
}

// Enum stores a top-level enum and returns it.
func (file *File) Enum(name string) *Enum {
	if file == nil {
		return nil
	}
	enum := &Enum{Name: name}
	file.Enums = append(file.Enums, enum)
	return enum
}

// Service stores a service and returns it.
func (file *File) Service(name string) *Service {
	if file == nil {
		return nil
	}
	service := &Service{Name: name}
	file.Services = append(file.Services, service)
	return service
}

// Option stores a message option.
func (message *Message) Option(name string, value Value) *Message {
	if message == nil {
		return nil
	}
	message.Options = append(message.Options, Option{Name: name, Value: value})
	return message
}

// Reserve stores one reserved number statement.
func (message *Message) Reserve(ranges ...Range) *Message {
	if message == nil {
		return nil
	}
	message.Reserved = append(message.Reserved, Reserve{Ranges: append([]Range(nil), ranges...)})
	return message
}

// ReservedNames stores one reserved name statement.
func (message *Message) ReservedNames(names ...string) *Message {
	if message == nil {
		return nil
	}
	message.Reserved = append(message.Reserved, Reserve{Names: append([]string(nil), names...)})
	return message
}

// Enum stores a nested enum and returns it.
func (message *Message) Enum(name string) *Enum {
	if message == nil {
		return nil
	}
	enum := &Enum{Name: name}
	message.Enums = append(message.Enums, enum)
	return enum
}

// Message stores a nested message and returns it.
func (message *Message) Message(name string) *Message {
	if message == nil {
		return nil
	}
	nested := &Message{Name: name}
	message.Messages = append(message.Messages, nested)
	return nested
}

// Field stores a field and returns it.
func (message *Message) Field(name string, number int, typ Type) *Field {
	if message == nil {
		return nil
	}
	field := newField(name, number, typ)
	message.Fields = append(message.Fields, field)
	return field
}

// Map stores a map field and returns it.
func (message *Message) Map(name string, number int, key, elem Type) *Field {
	return message.Field(name, number, Map(key, elem))
}

// Oneof stores a oneof and returns it.
func (message *Message) Oneof(name string) *Oneof {
	if message == nil {
		return nil
	}
	oneof := &Oneof{Name: name}
	message.Oneofs = append(message.Oneofs, oneof)
	return oneof
}

func newField(name string, number int, typ Type) *Field {
	return &Field{Name: name, Number: number, Type: typ}
}

// Optional sets the optional label.
func (field *Field) Optional() *Field {
	if field == nil {
		return nil
	}
	field.Label = LabelOptional
	return field
}

// Required sets the required label.
func (field *Field) Required() *Field {
	if field == nil {
		return nil
	}
	field.Label = LabelRequired
	return field
}

// Repeated sets the repeated label.
func (field *Field) Repeated() *Field {
	if field == nil {
		return nil
	}
	field.Label = LabelRepeated
	return field
}

// Option stores a field option.
func (field *Field) Option(name string, value Value) *Field {
	if field == nil {
		return nil
	}
	field.Options = append(field.Options, Option{Name: name, Value: value})
	return field
}

// Option stores a oneof option.
func (oneof *Oneof) Option(name string, value Value) *Oneof {
	if oneof == nil {
		return nil
	}
	oneof.Options = append(oneof.Options, Option{Name: name, Value: value})
	return oneof
}

// Field stores a field in the oneof and returns it.
func (oneof *Oneof) Field(name string, number int, typ Type) *Field {
	if oneof == nil {
		return nil
	}
	field := newField(name, number, typ)
	oneof.Fields = append(oneof.Fields, field)
	return field
}

// Option stores an enum option.
func (enum *Enum) Option(name string, value Value) *Enum {
	if enum == nil {
		return nil
	}
	enum.Options = append(enum.Options, Option{Name: name, Value: value})
	return enum
}

// Reserve stores one reserved number statement.
func (enum *Enum) Reserve(ranges ...Range) *Enum {
	if enum == nil {
		return nil
	}
	enum.Reserved = append(enum.Reserved, Reserve{Ranges: append([]Range(nil), ranges...)})
	return enum
}

// ReservedNames stores one reserved name statement.
func (enum *Enum) ReservedNames(names ...string) *Enum {
	if enum == nil {
		return nil
	}
	enum.Reserved = append(enum.Reserved, Reserve{Names: append([]string(nil), names...)})
	return enum
}

// Value stores an enumerator and returns it.
func (enum *Enum) Value(name string, number int) *EnumValue {
	if enum == nil {
		return nil
	}
	value := &EnumValue{Name: name, Number: number}
	enum.Values = append(enum.Values, value)
	return value
}

// Option stores an enumerator option.
func (value *EnumValue) Option(name string, constant Value) *EnumValue {
	if value == nil {
		return nil
	}
	value.Options = append(value.Options, Option{Name: name, Value: constant})
	return value
}

// Option stores a service option.
func (service *Service) Option(name string, value Value) *Service {
	if service == nil {
		return nil
	}
	service.Options = append(service.Options, Option{Name: name, Value: value})
	return service
}

// RPC stores an rpc and returns it.
func (service *Service) RPC(name, input, output string) *Method {
	if service == nil {
		return nil
	}
	method := &Method{Name: name, Input: input, Output: output}
	service.Methods = append(service.Methods, method)
	return method
}

// RequestStream marks the request as a stream.
func (method *Method) RequestStream() *Method {
	if method == nil {
		return nil
	}
	method.ClientStream = true
	return method
}

// ResponseStream marks the response as a stream.
func (method *Method) ResponseStream() *Method {
	if method == nil {
		return nil
	}
	method.ServerStream = true
	return method
}

// Option stores an rpc option.
func (method *Method) Option(name string, value Value) *Method {
	if method == nil {
		return nil
	}
	method.Options = append(method.Options, Option{Name: name, Value: value})
	return method
}

// Exact is a reserved number.
func Exact(number int) Range {
	return Range{Start: number, End: number}
}

// Span is a reserved inclusive range.
func Span(start, end int) Range {
	return Range{Start: start, End: end}
}

// ThroughMax is a reserved range that runs to max.
func ThroughMax(start int) Range {
	return Range{Start: start, Max: true}
}
