package clib

import (
	"errors"

	"github.com/lewtec/lewkit/x/errs"
	gotool "github.com/lewtec/lewkit/x/toolchain/go"
)

var (
	ErrHeader                              = errs.Wrap(errs.ErrRequired, "header required")
	ErrEmpty                               = errs.Wrap(errs.ErrEmpty, "no functions")
	ErrType                                = errs.Wrap(errs.ErrUnsupported, "unsupported type")
	ErrSymbol                              = errs.Wrap(errs.ErrUnsupported, "unsupported symbol")
	ErrHost                                = errs.Wrap(errs.ErrUnsupported, "unsupported target")
	ErrClassify                            = errors.New("")
	ErrClibDirectoryAndOutputAreRequired   = errs.Wrap(errs.ErrRequired, "clib: directory and output are required")
	ErrClibNoExports                       = errs.Wrap(errs.ErrNotFound, "clib: no exports")
	ErrClibPathsMustNotContainSpaces       = errors.New("clib: paths must not contain spaces")
	ErrClibToolPathMustNotContainSpaces    = errors.New("clib: tool path must not contain spaces")
	ErrCodeSignatureOverlapsTheHeader      = errors.New("code signature overlaps the header")
	ErrDynamicSegmentIsOutOfRange          = gotool.ErrDynamicSegmentIsOutOfRange
	ErrEntryPointIsOutsideItsSections      = errors.New("entry point is outside its sections")
	ErrExportDirectoryIsOutsideItsSections = errors.New("export directory is outside its sections")
	ErrExportNameIsOutsideItsSections      = errors.New("export name is outside its sections")
	ErrExportNamesAreOutsideItsSections    = errors.New("export names are outside its sections")
	ErrExportNamesAreTruncated             = errors.New("export names are truncated")
	ErrExportTrieIsEmpty                   = errs.Wrap(errs.ErrEmpty, "export trie is empty")
	ErrHasAShortOptionalHeader             = errors.New("has a short optional header")
	ErrHasAShortProgramHeader              = errors.New("has a short program header")
	ErrHasAnUnknownPeMagic                 = errs.Wrap(errs.ErrUnknown, "has an unknown PE magic")
	ErrHasNoDynamicSegment                 = gotool.ErrHasNoDynamicSegment
	ErrHasNoExportTrie                     = errs.Wrap(errs.ErrNotFound, "has no export trie")
	ErrHasNoProgramHeaders                 = errs.Wrap(errs.ErrNotFound, "has no program headers")
	ErrHasNoRoomForDylibLoadCommands       = errs.Wrap(errs.ErrNotFound, "has no room for dylib load commands")
	ErrHasNoTextOrLinkedit                 = errs.Wrap(errs.ErrNotFound, "has no __TEXT or __LINKEDIT")
	ErrIsNotA64BitLittleEndianMachOFile    = errs.Wrap(errs.ErrInvalid, "is not a 64-bit little-endian Mach-O file")
	ErrIsNotALittleEndianElfFile           = errs.Wrap(errs.ErrInvalid, "is not a little-endian ELF file")
	ErrIsNotAPeFile                        = errs.Wrap(errs.ErrInvalid, "is not a PE file")
	ErrIsNotASupportedPeMachine            = errs.Wrap(errs.ErrUnsupported, "is not a supported PE machine")
	ErrIsNotAnElf64File                    = errs.Wrap(errs.ErrInvalid, "is not an ELF64 file")
	ErrIsNotInTheImage                     = errors.New("is not in the image")
	ErrIsNotInTheSymbolTable               = errors.New("is not in the symbol table")
	ErrMachoExportTrieClib                 = errors.New("clib")
	ErrMakeDylibClib                       = errors.New("clib")
	ErrNoLcDyldInfoCommand                 = errs.Wrap(errs.ErrNotFound, "no LC_DYLD_INFO command")
	ErrPackageClausePackageName            = errors.New("package name")
	ErrPeExportsNotAPeFile                 = errs.Wrap(errs.ErrInvalid, "not a PE file")
	ErrProgramHeadersAreOutOfRange         = gotool.ErrProgramHeadersAreOutOfRange
	ErrRequireImageExportsClib             = errors.New("clib")
	ErrRun                                 = errors.New("")
	ErrRunGoClibGo                         = errors.New("clib: go")
	ErrRunGoClibGo2                        = errors.New("clib: go")
	ErrRunPackageName                      = errors.New("package name")
	ErrSectionHeadersAreOutOfRange         = errs.Wrap(errs.ErrOutOfRange, "section headers are out of range")
	ErrShortCodeSignatureCommand           = errors.New("short code signature command")
	ErrShortDyldInfoCommand                = errors.New("short dyld info command")
	ErrShortLoadCommand                    = errors.New("short load command")
	ErrShortOptionalHeader                 = errors.New("short optional header")
	ErrUnknownPeMagic                      = errs.Wrap(errs.ErrUnknown, "unknown PE magic")
)
