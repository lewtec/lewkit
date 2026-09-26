package jsonleaven

import "unsafe"

type TSFieldMapEntry struct {
	F0 int16
	F1 byte
	F2 byte
}
type TSFieldMapSlice struct {
	F0 int16
	F1 int16
}
type TSLanguage struct {
	F0  int32
	F1  int32
	F2  int32
	F3  int32
	F4  int32
	F5  int32
	F6  int32
	F7  int32
	F8  int32
	F9  int16
	F10 *int16
	F11 *int16
	F12 *int32
	F13 *TSParseActionEntry
	F14 **byte
	F15 **byte
	F16 *TSFieldMapSlice
	F17 *TSFieldMapEntry
	F18 *TSSymbolMetadata
	F19 *int16
	F20 *int16
	F21 *int16
	F22 *TSLexMode
	F23 func(*TSLexer, int16) bool
	F24 func(*TSLexer, int16) bool
	F25 int16
	F26 anon_2
	F27 *int16
}
type TSLexMode struct {
	F0 int16
	F1 int16
}
type TSLexer struct {
	F0 int32
	F1 int16
	F2 func(*TSLexer, bool)
	F3 func(*TSLexer)
	F4 func(*TSLexer) int32
	F5 func(*TSLexer) bool
	F6 func(*TSLexer) bool
	F7 func(*TSLexer, *byte, ...interface{})
}
type TSSymbolMetadata struct {
	F0 byte
	F1 byte
	F2 byte
}
type anon_0 struct {
	F0 byte
	F1 byte
	F2 int16
	F3 int16
	F4 int16
}
type anon_1 struct {
	F0 byte
	F1 byte
}
type anon_2 struct {
	F0 *byte
	F1 *int16
	F2 func() *byte
	F3 func(*byte)
	F4 func(*byte, *TSLexer, *byte) bool
	F5 func(*byte, *byte) int32
	F6 func(*byte, *byte, int32)
}
type TSParseAction struct {
	F0 anon_0
}
type TSParseActionEntry struct {
	F0 TSParseAction
}

var tree_sitter_json_language TSLanguage = TSLanguage{14, 25, 0, 15, 0, 32, 7, 2, 2, 4, &(*[7][25]int16)(unsafe.Pointer(&ts_parse_table))[0][0], &ts_small_parse_table[0], &ts_small_parse_table_map[0], &(*[96]TSParseActionEntry)(unsafe.Pointer(&ts_parse_actions))[0], &ts_symbol_names[0], &ts_field_names[0], &ts_field_map_slices[0], &ts_field_map_entries[0], &ts_symbol_metadata[0], &ts_symbol_map[0], &ts_non_terminal_alias_map[0], &ts_alias_sequences[0][0], &(*[32]TSLexMode)(unsafe.Pointer(&ts_lex_modes))[0], ts_lex, nil, 0, anon_2{}, &ts_primary_state_ids[0]}
var ts_small_parse_table [360]int16 = [360]int16{2, 3, 1, 14, 37, 11, 0, 1, 2, 3, 5, 6, 7, 10, 11, 12, 13, 2, 3, 1, 14, 39, 11, 0, 1, 2, 3, 5, 6, 7, 10, 11, 12, 13, 2, 3, 1, 14, 41, 11, 0, 1, 2, 3, 5, 6, 7, 10, 11, 12, 13, 7, 3, 1, 14, 7, 1, 1, 9, 1, 5, 11, 1, 7, 29, 1, 16, 8, 3, 17, 19, 20, 13, 4, 10, 11, 12, 13, 2, 3, 1, 14, 43, 11, 0, 1, 2, 3, 5, 6, 7, 10, 11, 12, 13, 2, 3, 1, 14, 45, 11, 0, 1, 2, 3, 5, 6, 7, 10, 11, 12, 13, 7, 3, 1, 14, 7, 1, 1, 9, 1, 5, 11, 1, 7, 28, 1, 16, 8, 3, 17, 19, 20, 13, 4, 10, 11, 12, 13, 2, 3, 1, 14, 47, 11, 0, 1, 2, 3, 5, 6, 7, 10, 11, 12, 13, 2, 3, 1, 14, 49, 11, 0, 1, 2, 3, 5, 6, 7, 10, 11, 12, 13, 5, 3, 1, 14, 11, 1, 7, 51, 1, 3, 20, 1, 18, 31, 1, 20, 4, 53, 1, 7, 57, 1, 14, 18, 1, 21, 55, 2, 8, 9, 4, 57, 1, 14, 59, 1, 7, 19, 1, 21, 61, 2, 8, 9, 4, 57, 1, 14, 63, 1, 7, 19, 1, 21, 65, 2, 8, 9, 4, 3, 1, 14, 68, 1, 2, 70, 1, 3, 22, 1, 23, 4, 3, 1, 14, 72, 1, 2, 74, 1, 6, 24, 1, 24, 4, 3, 1, 14, 68, 1, 2, 76, 1, 3, 25, 1, 23, 4, 3, 1, 14, 11, 1, 7, 27, 1, 18, 31, 1, 20, 4, 3, 1, 14, 72, 1, 2, 78, 1, 6, 26, 1, 24, 4, 3, 1, 14, 80, 1, 2, 83, 1, 3, 25, 1, 23, 4, 3, 1, 14, 85, 1, 2, 88, 1, 6, 26, 1, 24, 2, 3, 1, 14, 83, 2, 2, 3, 2, 3, 1, 14, 90, 2, 2, 3, 2, 3, 1, 14, 88, 2, 2, 6, 2, 3, 1, 14, 92, 1, 0, 2, 3, 1, 14, 94, 1, 4}
var ts_small_parse_table_map [25]int32 = [25]int32{0, 17, 34, 51, 78, 95, 112, 139, 156, 173, 189, 203, 217, 231, 244, 257, 270, 283, 296, 309, 322, 330, 338, 346, 353}
var ts_symbol_names [25]*byte = [25]*byte{&_str[0], &_str_3[0], &_str_4[0], &_str_5[0], &_str_6[0], &_str_7[0], &_str_8[0], &_str_9[0], &_str_10[0], &_str_11[0], &_str_12[0], &_str_13[0], &_str_14[0], &_str_15[0], &_str_16[0], &_str_17[0], &_str_18[0], &_str_19[0], &_str_20[0], &_str_21[0], &_str_22[0], &_str_23[0], &_str_24[0], &_str_25[0], &_str_26[0]}
var ts_field_names [3]*byte = [3]*byte{nil, &_str_27[0], &_str_28[0]}
var ts_field_map_slices [2]TSFieldMapSlice = [2]TSFieldMapSlice{TSFieldMapSlice{}, TSFieldMapSlice{0, 2}}
var ts_field_map_entries [2]TSFieldMapEntry = [2]TSFieldMapEntry{TSFieldMapEntry{1, 0, 0}, TSFieldMapEntry{2, 2, 0}}
var ts_symbol_metadata [25]TSSymbolMetadata = [25]TSSymbolMetadata{TSSymbolMetadata{0, 1, 0}, TSSymbolMetadata{1, 0, 0}, TSSymbolMetadata{1, 0, 0}, TSSymbolMetadata{1, 0, 0}, TSSymbolMetadata{1, 0, 0}, TSSymbolMetadata{1, 0, 0}, TSSymbolMetadata{1, 0, 0}, TSSymbolMetadata{1, 0, 0}, TSSymbolMetadata{1, 1, 0}, TSSymbolMetadata{1, 1, 0}, TSSymbolMetadata{1, 1, 0}, TSSymbolMetadata{1, 1, 0}, TSSymbolMetadata{1, 1, 0}, TSSymbolMetadata{1, 1, 0}, TSSymbolMetadata{1, 1, 0}, TSSymbolMetadata{1, 1, 0}, TSSymbolMetadata{0, 1, 1}, TSSymbolMetadata{1, 1, 0}, TSSymbolMetadata{1, 1, 0}, TSSymbolMetadata{1, 1, 0}, TSSymbolMetadata{1, 1, 0}, TSSymbolMetadata{}, TSSymbolMetadata{}, TSSymbolMetadata{}, TSSymbolMetadata{}}
var ts_symbol_map [25]int16 = [25]int16{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24}
var ts_non_terminal_alias_map [1]int16 = [1]int16{}
var ts_alias_sequences [2][4]int16 = [2][4]int16{}
var ts_primary_state_ids [32]int16 = [32]int16{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31}
var ts_parse_table struct {
	F0 struct {
		F0 [15]int16
		F1 [10]int16
	}
	F1 [25]int16
	F2 [25]int16
	F3 [25]int16
	F4 [25]int16
	F5 struct {
		F0 [15]int16
		F1 [10]int16
	}
	F6 struct {
		F0 [15]int16
		F1 [10]int16
	}
} = struct {
	F0 struct {
		F0 [15]int16
		F1 [10]int16
	}
	F1 [25]int16
	F2 [25]int16
	F3 [25]int16
	F4 [25]int16
	F5 struct {
		F0 [15]int16
		F1 [10]int16
	}
	F6 struct {
		F0 [15]int16
		F1 [10]int16
	}
}{struct {
	F0 [15]int16
	F1 [10]int16
}{[15]int16{1, 1, 1, 1, 1, 1, 1, 1, 0, 1, 1, 1, 1, 1, 3}, [10]int16{}}, [25]int16{5, 7, 0, 0, 0, 9, 0, 11, 0, 0, 13, 13, 13, 13, 3, 30, 2, 8, 0, 8, 8, 0, 2, 0, 0}, [25]int16{15, 7, 0, 0, 0, 9, 0, 11, 0, 0, 13, 13, 13, 13, 3, 0, 3, 8, 0, 8, 8, 0, 3, 0, 0}, [25]int16{17, 19, 0, 0, 0, 22, 0, 25, 0, 0, 28, 28, 28, 28, 3, 0, 3, 8, 0, 8, 8, 0, 3, 0, 0}, [25]int16{0, 7, 0, 0, 0, 9, 31, 11, 0, 0, 13, 13, 13, 13, 3, 0, 21, 8, 0, 8, 8, 0, 0, 0, 0}, struct {
	F0 [15]int16
	F1 [10]int16
}{[15]int16{33, 33, 33, 33, 33, 33, 33, 33, 0, 0, 33, 33, 33, 33, 3}, [10]int16{}}, struct {
	F0 [15]int16
	F1 [10]int16
}{[15]int16{35, 35, 35, 35, 35, 35, 35, 35, 0, 0, 35, 35, 35, 35, 3}, [10]int16{}}}
var ts_parse_actions struct {
	F0 struct {
		F0 anon_1
		F1 [6]byte
	}
	F1 struct {
		F0 anon_1
		F1 [6]byte
	}
	F2 struct {
		F0 struct {
			F0 byte
			F1 [7]byte
		}
	}
	F3 struct {
		F0 anon_1
		F1 [6]byte
	}
	F4 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F5 struct {
		F0 anon_1
		F1 [6]byte
	}
	F6 TSParseActionEntry
	F7 struct {
		F0 anon_1
		F1 [6]byte
	}
	F8 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F9 struct {
		F0 anon_1
		F1 [6]byte
	}
	F10 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F11 struct {
		F0 anon_1
		F1 [6]byte
	}
	F12 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F13 struct {
		F0 anon_1
		F1 [6]byte
	}
	F14 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F15 struct {
		F0 anon_1
		F1 [6]byte
	}
	F16 TSParseActionEntry
	F17 struct {
		F0 anon_1
		F1 [6]byte
	}
	F18 TSParseActionEntry
	F19 struct {
		F0 anon_1
		F1 [6]byte
	}
	F20 TSParseActionEntry
	F21 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F22 struct {
		F0 anon_1
		F1 [6]byte
	}
	F23 TSParseActionEntry
	F24 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F25 struct {
		F0 anon_1
		F1 [6]byte
	}
	F26 TSParseActionEntry
	F27 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F28 struct {
		F0 anon_1
		F1 [6]byte
	}
	F29 TSParseActionEntry
	F30 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F31 struct {
		F0 anon_1
		F1 [6]byte
	}
	F32 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F33 struct {
		F0 anon_1
		F1 [6]byte
	}
	F34 TSParseActionEntry
	F35 struct {
		F0 anon_1
		F1 [6]byte
	}
	F36 TSParseActionEntry
	F37 struct {
		F0 anon_1
		F1 [6]byte
	}
	F38 TSParseActionEntry
	F39 struct {
		F0 anon_1
		F1 [6]byte
	}
	F40 TSParseActionEntry
	F41 struct {
		F0 anon_1
		F1 [6]byte
	}
	F42 TSParseActionEntry
	F43 struct {
		F0 anon_1
		F1 [6]byte
	}
	F44 TSParseActionEntry
	F45 struct {
		F0 anon_1
		F1 [6]byte
	}
	F46 TSParseActionEntry
	F47 struct {
		F0 anon_1
		F1 [6]byte
	}
	F48 TSParseActionEntry
	F49 struct {
		F0 anon_1
		F1 [6]byte
	}
	F50 TSParseActionEntry
	F51 struct {
		F0 anon_1
		F1 [6]byte
	}
	F52 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F53 struct {
		F0 anon_1
		F1 [6]byte
	}
	F54 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F55 struct {
		F0 anon_1
		F1 [6]byte
	}
	F56 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F57 struct {
		F0 anon_1
		F1 [6]byte
	}
	F58 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F59 struct {
		F0 anon_1
		F1 [6]byte
	}
	F60 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F61 struct {
		F0 anon_1
		F1 [6]byte
	}
	F62 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F63 struct {
		F0 anon_1
		F1 [6]byte
	}
	F64 TSParseActionEntry
	F65 struct {
		F0 anon_1
		F1 [6]byte
	}
	F66 TSParseActionEntry
	F67 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F68 struct {
		F0 anon_1
		F1 [6]byte
	}
	F69 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F70 struct {
		F0 anon_1
		F1 [6]byte
	}
	F71 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F72 struct {
		F0 anon_1
		F1 [6]byte
	}
	F73 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F74 struct {
		F0 anon_1
		F1 [6]byte
	}
	F75 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F76 struct {
		F0 anon_1
		F1 [6]byte
	}
	F77 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F78 struct {
		F0 anon_1
		F1 [6]byte
	}
	F79 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F80 struct {
		F0 anon_1
		F1 [6]byte
	}
	F81 TSParseActionEntry
	F82 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F83 struct {
		F0 anon_1
		F1 [6]byte
	}
	F84 TSParseActionEntry
	F85 struct {
		F0 anon_1
		F1 [6]byte
	}
	F86 TSParseActionEntry
	F87 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F88 struct {
		F0 anon_1
		F1 [6]byte
	}
	F89 TSParseActionEntry
	F90 struct {
		F0 anon_1
		F1 [6]byte
	}
	F91 TSParseActionEntry
	F92 struct {
		F0 anon_1
		F1 [6]byte
	}
	F93 struct {
		F0 struct {
			F0 byte
			F1 [7]byte
		}
	}
	F94 struct {
		F0 anon_1
		F1 [6]byte
	}
	F95 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
} = struct {
	F0 struct {
		F0 anon_1
		F1 [6]byte
	}
	F1 struct {
		F0 anon_1
		F1 [6]byte
	}
	F2 struct {
		F0 struct {
			F0 byte
			F1 [7]byte
		}
	}
	F3 struct {
		F0 anon_1
		F1 [6]byte
	}
	F4 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F5 struct {
		F0 anon_1
		F1 [6]byte
	}
	F6 TSParseActionEntry
	F7 struct {
		F0 anon_1
		F1 [6]byte
	}
	F8 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F9 struct {
		F0 anon_1
		F1 [6]byte
	}
	F10 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F11 struct {
		F0 anon_1
		F1 [6]byte
	}
	F12 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F13 struct {
		F0 anon_1
		F1 [6]byte
	}
	F14 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F15 struct {
		F0 anon_1
		F1 [6]byte
	}
	F16 TSParseActionEntry
	F17 struct {
		F0 anon_1
		F1 [6]byte
	}
	F18 TSParseActionEntry
	F19 struct {
		F0 anon_1
		F1 [6]byte
	}
	F20 TSParseActionEntry
	F21 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F22 struct {
		F0 anon_1
		F1 [6]byte
	}
	F23 TSParseActionEntry
	F24 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F25 struct {
		F0 anon_1
		F1 [6]byte
	}
	F26 TSParseActionEntry
	F27 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F28 struct {
		F0 anon_1
		F1 [6]byte
	}
	F29 TSParseActionEntry
	F30 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F31 struct {
		F0 anon_1
		F1 [6]byte
	}
	F32 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F33 struct {
		F0 anon_1
		F1 [6]byte
	}
	F34 TSParseActionEntry
	F35 struct {
		F0 anon_1
		F1 [6]byte
	}
	F36 TSParseActionEntry
	F37 struct {
		F0 anon_1
		F1 [6]byte
	}
	F38 TSParseActionEntry
	F39 struct {
		F0 anon_1
		F1 [6]byte
	}
	F40 TSParseActionEntry
	F41 struct {
		F0 anon_1
		F1 [6]byte
	}
	F42 TSParseActionEntry
	F43 struct {
		F0 anon_1
		F1 [6]byte
	}
	F44 TSParseActionEntry
	F45 struct {
		F0 anon_1
		F1 [6]byte
	}
	F46 TSParseActionEntry
	F47 struct {
		F0 anon_1
		F1 [6]byte
	}
	F48 TSParseActionEntry
	F49 struct {
		F0 anon_1
		F1 [6]byte
	}
	F50 TSParseActionEntry
	F51 struct {
		F0 anon_1
		F1 [6]byte
	}
	F52 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F53 struct {
		F0 anon_1
		F1 [6]byte
	}
	F54 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F55 struct {
		F0 anon_1
		F1 [6]byte
	}
	F56 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F57 struct {
		F0 anon_1
		F1 [6]byte
	}
	F58 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F59 struct {
		F0 anon_1
		F1 [6]byte
	}
	F60 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F61 struct {
		F0 anon_1
		F1 [6]byte
	}
	F62 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F63 struct {
		F0 anon_1
		F1 [6]byte
	}
	F64 TSParseActionEntry
	F65 struct {
		F0 anon_1
		F1 [6]byte
	}
	F66 TSParseActionEntry
	F67 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F68 struct {
		F0 anon_1
		F1 [6]byte
	}
	F69 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F70 struct {
		F0 anon_1
		F1 [6]byte
	}
	F71 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F72 struct {
		F0 anon_1
		F1 [6]byte
	}
	F73 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F74 struct {
		F0 anon_1
		F1 [6]byte
	}
	F75 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F76 struct {
		F0 anon_1
		F1 [6]byte
	}
	F77 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F78 struct {
		F0 anon_1
		F1 [6]byte
	}
	F79 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F80 struct {
		F0 anon_1
		F1 [6]byte
	}
	F81 TSParseActionEntry
	F82 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F83 struct {
		F0 anon_1
		F1 [6]byte
	}
	F84 TSParseActionEntry
	F85 struct {
		F0 anon_1
		F1 [6]byte
	}
	F86 TSParseActionEntry
	F87 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
	F88 struct {
		F0 anon_1
		F1 [6]byte
	}
	F89 TSParseActionEntry
	F90 struct {
		F0 anon_1
		F1 [6]byte
	}
	F91 TSParseActionEntry
	F92 struct {
		F0 anon_1
		F1 [6]byte
	}
	F93 struct {
		F0 struct {
			F0 byte
			F1 [7]byte
		}
	}
	F94 struct {
		F0 anon_1
		F1 [6]byte
	}
	F95 struct {
		F0 struct {
			F0 struct {
				F0 byte
				F1 int16
				F2 byte
				F3 byte
			}
			F1 [2]byte
		}
	}
}{struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{}, [6]byte{}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 0}, [6]byte{}}, struct {
	F0 struct {
		F0 byte
		F1 [7]byte
	}
}{struct {
	F0 byte
	F1 [7]byte
}{3, [7]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 0, 1, 0}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 0, 15, 0, 0}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 16, 0, 0}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 4, 0, 0}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 17, 0, 0}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 8, 0, 0}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 1, 15, 0, 0}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 2, 22, 0, 0}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{2, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 2, 22, 0, 0}}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 16, 0, 1}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{2, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 2, 22, 0, 0}}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 4, 0, 1}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{2, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 2, 22, 0, 0}}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 17, 0, 1}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{2, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 2, 22, 0, 0}}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 8, 0, 1}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 9, 0, 0}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 2, 20, 0, 0}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 3, 20, 0, 0}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 2, 17, 0, 0}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 1, 16, 0, 0}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 2, 19, 0, 0}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 3, 17, 0, 0}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 4, 17, 0, 0}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 3, 19, 0, 0}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 4, 19, 0, 0}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 7, 0, 0}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 0}, [6]byte{}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 5, 0, 0}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 18, 0, 0}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 0}, [6]byte{}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 0, 1, 0}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 0}, [6]byte{}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 6, 0, 0}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 19, 0, 0}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 0}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 2, 21, 0, 0}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{2, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 2, 21, 0, 0}}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 19, 0, 1}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 23, 0, 0}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 11, 0, 0}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 10, 0, 0}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 14, 0, 0}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 12, 0, 0}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 15, 0, 0}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{2, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 2, 23, 0, 0}}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 23, 0, 1}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 2, 23, 0, 0}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{2, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 2, 24, 0, 0}}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 10, 0, 1}, [2]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 2, 24, 0, 0}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, TSParseActionEntry{TSParseAction{anon_0{1, 3, 18, 0, 1}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, struct {
	F0 struct {
		F0 byte
		F1 [7]byte
	}
}{struct {
	F0 byte
	F1 [7]byte
}{2, [7]byte{}}}, struct {
	F0 anon_1
	F1 [6]byte
}{anon_1{1, 1}, [6]byte{}}, struct {
	F0 struct {
		F0 struct {
			F0 byte
			F1 int16
			F2 byte
			F3 byte
		}
		F1 [2]byte
	}
}{struct {
	F0 struct {
		F0 byte
		F1 int16
		F2 byte
		F3 byte
	}
	F1 [2]byte
}{struct {
	F0 byte
	F1 int16
	F2 byte
	F3 byte
}{0, 13, 0, 0}, [2]byte{}}}}
var _str [4]byte = [4]byte{101, 110, 100, 0}
var _str_3 [2]byte = [2]byte{123, 0}
var _str_4 [2]byte = [2]byte{44, 0}
var _str_5 [2]byte = [2]byte{125, 0}
var _str_6 [2]byte = [2]byte{58, 0}
var _str_7 [2]byte = [2]byte{91, 0}
var _str_8 [2]byte = [2]byte{93, 0}
var _str_9 [2]byte = [2]byte{34, 0}
var _str_10 [15]byte = [15]byte{115, 116, 114, 105, 110, 103, 95, 99, 111, 110, 116, 101, 110, 116, 0}
var _str_11 [16]byte = [16]byte{101, 115, 99, 97, 112, 101, 95, 115, 101, 113, 117, 101, 110, 99, 101, 0}
var _str_12 [7]byte = [7]byte{110, 117, 109, 98, 101, 114, 0}
var _str_13 [5]byte = [5]byte{116, 114, 117, 101, 0}
var _str_14 [6]byte = [6]byte{102, 97, 108, 115, 101, 0}
var _str_15 [5]byte = [5]byte{110, 117, 108, 108, 0}
var _str_16 [8]byte = [8]byte{99, 111, 109, 109, 101, 110, 116, 0}
var _str_17 [9]byte = [9]byte{100, 111, 99, 117, 109, 101, 110, 116, 0}
var _str_18 [7]byte = [7]byte{95, 118, 97, 108, 117, 101, 0}
var _str_19 [7]byte = [7]byte{111, 98, 106, 101, 99, 116, 0}
var _str_20 [5]byte = [5]byte{112, 97, 105, 114, 0}
var _str_21 [6]byte = [6]byte{97, 114, 114, 97, 121, 0}
var _str_22 [7]byte = [7]byte{115, 116, 114, 105, 110, 103, 0}
var _str_23 [16]byte = [16]byte{95, 115, 116, 114, 105, 110, 103, 95, 99, 111, 110, 116, 101, 110, 116, 0}
var _str_24 [17]byte = [17]byte{100, 111, 99, 117, 109, 101, 110, 116, 95, 114, 101, 112, 101, 97, 116, 49, 0}
var _str_25 [15]byte = [15]byte{111, 98, 106, 101, 99, 116, 95, 114, 101, 112, 101, 97, 116, 49, 0}
var _str_26 [14]byte = [14]byte{97, 114, 114, 97, 121, 95, 114, 101, 112, 101, 97, 116, 49, 0}
var _str_27 [4]byte = [4]byte{107, 101, 121, 0}
var _str_28 [6]byte = [6]byte{118, 97, 108, 117, 101, 0}
var ts_lex_modes struct {
	F0 [20]TSLexMode
	F1 [12]TSLexMode
} = struct {
	F0 [20]TSLexMode
	F1 [12]TSLexMode
}{[20]TSLexMode{TSLexMode{}, TSLexMode{}, TSLexMode{}, TSLexMode{}, TSLexMode{}, TSLexMode{}, TSLexMode{}, TSLexMode{}, TSLexMode{}, TSLexMode{}, TSLexMode{}, TSLexMode{}, TSLexMode{}, TSLexMode{}, TSLexMode{}, TSLexMode{}, TSLexMode{}, TSLexMode{1, 0}, TSLexMode{1, 0}, TSLexMode{1, 0}}, [12]TSLexMode{}}
var ts_lex_map [28]int16 = [28]int16{34, 28, 44, 23, 45, 7, 47, 3, 48, 35, 58, 25, 91, 26, 92, 18, 93, 27, 102, 8, 110, 17, 116, 14, 123, 22, 125, 24}
var ts_lex_map_30 [18]int16 = [18]int16{34, 34, 47, 34, 92, 34, 98, 34, 102, 34, 110, 34, 114, 34, 116, 34, 117, 34}
var ts_lex_map_31 [26]int16 = [26]int16{34, 28, 44, 23, 45, 7, 47, 3, 48, 35, 58, 25, 91, 26, 93, 27, 102, 8, 110, 17, 116, 14, 123, 22, 125, 24}

func tree_sitter_json() *TSLanguage {
	return &tree_sitter_json_language
}
func ts_lex(lexer *TSLexer, state int16) bool {
	var v0, v2, v4, v6, v8, v102, v103, v105, v107, v108, v110, v112, v113, v115, v117, v118, v120, v122, v123, v125, v127, v128, v130, v132, v133, v135, v137, v138, v140, v142, v143, v145, v153, v154, v156, v164, v165, v167, v174, v175, v177, v189, v190, v192, v198, v199, v201, v203, v204, v206, v211, v212, v214, v221, v222, v224, v230, v231, v233, v237, v238, v240, v242, v243, v245, v247, v248, v250, v252, v253, v255, v257, v258, v260 *TSLexer
	var lexer_addr **TSLexer
	var tobool, call, tobool3, cmp, cmp7, cmp14, cmp16, cmp18, cmp22, cmp25, tobool29, cmp31, cmp35, cmp39, cmp43, cmp47, cmp50, cmp53, cmp57, tobool61, cmp63, cmp67, cmp71, cmp74, cmp77, tobool81, cmp83, cmp87, tobool91, cmp93, cmp97, cmp101, tobool105, cmp107, cmp111, tobool115, cmp117, cmp121, cmp124, tobool128, cmp130, cmp134, cmp137, tobool141, cmp143, tobool147, cmp149, tobool153, cmp155, tobool159, cmp161, tobool165, cmp167, tobool171, cmp173, tobool177, cmp179, tobool183, cmp185, tobool189, cmp191, tobool195, cmp197, tobool201, cmp206, cmp212, tobool222, cmp224, cmp227, tobool231, tobool233, cmp239, cmp245, cmp255, cmp258, cmp261, cmp265, cmp268, tobool272, tobool274, tobool278, tobool282, tobool286, tobool290, tobool294, tobool298, tobool302, cmp306, cmp310, cmp314, cmp317, cmp320, cmp323, tobool327, cmp331, cmp335, cmp339, cmp342, cmp345, cmp348, tobool352, cmp356, cmp360, cmp363, cmp366, cmp369, tobool373, cmp377, cmp381, cmp384, cmp387, cmp390, cmp394, cmp397, cmp400, cmp403, cmp406, tobool410, cmp414, cmp417, cmp420, cmp423, tobool427, tobool431, cmp435, cmp439, cmp442, tobool446, cmp450, cmp454, cmp457, cmp461, cmp464, tobool468, cmp472, cmp475, cmp479, cmp482, tobool486, cmp490, cmp493, tobool497, tobool501, tobool505, tobool509, tobool513, cmp517, cmp520, tobool524, v264 bool
	var v7 func(*TSLexer) bool
	var eof2 *func(*TSLexer) bool
	var retval *bool
	var v9, v13, v16, v79, v82, v91, v94 int16
	var state_addr, arrayidx, arrayidx11, arrayidx210, arrayidx217, arrayidx243, arrayidx250, result_symbol, result_symbol276, result_symbol280, result_symbol284, result_symbol288, result_symbol292, result_symbol296, result_symbol300, result_symbol304, result_symbol329, result_symbol354, result_symbol375, result_symbol412, result_symbol429, result_symbol433, result_symbol448, result_symbol470, result_symbol488, result_symbol499, result_symbol503, result_symbol507, result_symbol511, result_symbol515 *int16
	var v5, conv, v11, v12, conv6, v14, v15, add, v17, add13, v18, v19, v20, v21, v22, v24, v25, v26, v27, v28, v29, v30, v31, v33, v34, v35, v36, v37, v39, v40, v42, v43, v44, v46, v47, v49, v50, v51, v53, v54, v55, v57, v59, v61, v63, v65, v67, v69, v71, v73, v75, v77, v78, conv211, v80, v81, add215, v83, add220, v85, v86, v89, v90, conv244, v92, v93, add248, v95, add253, v96, v97, v98, v99, v100, v146, v147, v148, v149, v150, v151, v157, v158, v159, v160, v161, v162, v168, v169, v170, v171, v172, v178, v179, v180, v181, v182, v183, v184, v185, v186, v187, v193, v194, v195, v196, v207, v208, v209, v215, v216, v217, v218, v219, v225, v226, v227, v228, v234, v235, v261, v262 int32
	var lookahead, i, i203, i236, lookahead1 *int32
	var conv4, idxprom, idxprom10, conv205, idxprom209, idxprom216, conv238, idxprom242, idxprom249 int64
	var v3, frombool, v10, v23, v32, v38, v41, v45, v48, v52, v56, v58, v60, v62, v64, v66, v68, v70, v72, v74, v76, v84, v87, v88, v101, v106, v111, v116, v121, v126, v131, v136, v141, v152, v163, v173, v188, v197, v202, v210, v220, v229, v236, v241, v246, v251, v256, v263 byte
	var result, skip, eof *byte
	var v104, v109, v114, v119, v124, v129, v134, v139, v144, v155, v166, v176, v191, v200, v205, v213, v223, v232, v239, v244, v249, v254, v259 func(*TSLexer)
	var mark_end, mark_end277, mark_end281, mark_end285, mark_end289, mark_end293, mark_end297, mark_end301, mark_end305, mark_end330, mark_end355, mark_end376, mark_end413, mark_end430, mark_end434, mark_end449, mark_end471, mark_end489, mark_end500, mark_end504, mark_end508, mark_end512, mark_end516 *func(*TSLexer)
	var v1 func(*TSLexer, bool)
	var advance *func(*TSLexer, bool)
	_, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _, _ = retval, lexer_addr, state_addr, result, skip, eof, lookahead, i, i203, i236, v0, advance, v1, v2, v3, tobool, v4, lookahead1, v5, v6, eof2, v7, v8, call, frombool, v9, conv, v10, tobool3, v11, conv4, cmp, v12, idxprom, arrayidx, v13, conv6, v14, cmp7, v15, add, idxprom10, arrayidx11, v16, v17, add13, v18, cmp14, v19, cmp16, v20, cmp18, v21, cmp22, v22, cmp25, v23, tobool29, v24, cmp31, v25, cmp35, v26, cmp39, v27, cmp43, v28, cmp47, v29, cmp50, v30, cmp53, v31, cmp57, v32, tobool61, v33, cmp63, v34, cmp67, v35, cmp71, v36, cmp74, v37, cmp77, v38, tobool81, v39, cmp83, v40, cmp87, v41, tobool91, v42, cmp93, v43, cmp97, v44, cmp101, v45, tobool105, v46, cmp107, v47, cmp111, v48, tobool115, v49, cmp117, v50, cmp121, v51, cmp124, v52, tobool128, v53, cmp130, v54, cmp134, v55, cmp137, v56, tobool141, v57, cmp143, v58, tobool147, v59, cmp149, v60, tobool153, v61, cmp155, v62, tobool159, v63, cmp161, v64, tobool165, v65, cmp167, v66, tobool171, v67, cmp173, v68, tobool177, v69, cmp179, v70, tobool183, v71, cmp185, v72, tobool189, v73, cmp191, v74, tobool195, v75, cmp197, v76, tobool201, v77, conv205, cmp206, v78, idxprom209, arrayidx210, v79, conv211, v80, cmp212, v81, add215, idxprom216, arrayidx217, v82, v83, add220, v84, tobool222, v85, cmp224, v86, cmp227, v87, tobool231, v88, tobool233, v89, conv238, cmp239, v90, idxprom242, arrayidx243, v91, conv244, v92, cmp245, v93, add248, idxprom249, arrayidx250, v94, v95, add253, v96, cmp255, v97, cmp258, v98, cmp261, v99, cmp265, v100, cmp268, v101, tobool272, v102, result_symbol, v103, mark_end, v104, v105, v106, tobool274, v107, result_symbol276, v108, mark_end277, v109, v110, v111, tobool278, v112, result_symbol280, v113, mark_end281, v114, v115, v116, tobool282, v117, result_symbol284, v118, mark_end285, v119, v120, v121, tobool286, v122, result_symbol288, v123, mark_end289, v124, v125, v126, tobool290, v127, result_symbol292, v128, mark_end293, v129, v130, v131, tobool294, v132, result_symbol296, v133, mark_end297, v134, v135, v136, tobool298, v137, result_symbol300, v138, mark_end301, v139, v140, v141, tobool302, v142, result_symbol304, v143, mark_end305, v144, v145, v146, cmp306, v147, cmp310, v148, cmp314, v149, cmp317, v150, cmp320, v151, cmp323, v152, tobool327, v153, result_symbol329, v154, mark_end330, v155, v156, v157, cmp331, v158, cmp335, v159, cmp339, v160, cmp342, v161, cmp345, v162, cmp348, v163, tobool352, v164, result_symbol354, v165, mark_end355, v166, v167, v168, cmp356, v169, cmp360, v170, cmp363, v171, cmp366, v172, cmp369, v173, tobool373, v174, result_symbol375, v175, mark_end376, v176, v177, v178, cmp377, v179, cmp381, v180, cmp384, v181, cmp387, v182, cmp390, v183, cmp394, v184, cmp397, v185, cmp400, v186, cmp403, v187, cmp406, v188, tobool410, v189, result_symbol412, v190, mark_end413, v191, v192, v193, cmp414, v194, cmp417, v195, cmp420, v196, cmp423, v197, tobool427, v198, result_symbol429, v199, mark_end430, v200, v201, v202, tobool431, v203, result_symbol433, v204, mark_end434, v205, v206, v207, cmp435, v208, cmp439, v209, cmp442, v210, tobool446, v211, result_symbol448, v212, mark_end449, v213, v214, v215, cmp450, v216, cmp454, v217, cmp457, v218, cmp461, v219, cmp464, v220, tobool468, v221, result_symbol470, v222, mark_end471, v223, v224, v225, cmp472, v226, cmp475, v227, cmp479, v228, cmp482, v229, tobool486, v230, result_symbol488, v231, mark_end489, v232, v233, v234, cmp490, v235, cmp493, v236, tobool497, v237, result_symbol499, v238, mark_end500, v239, v240, v241, tobool501, v242, result_symbol503, v243, mark_end504, v244, v245, v246, tobool505, v247, result_symbol507, v248, mark_end508, v249, v250, v251, tobool509, v252, result_symbol511, v253, mark_end512, v254, v255, v256, tobool513, v257, result_symbol515, v258, mark_end516, v259, v260, v261, cmp517, v262, cmp520, v263, tobool524, v264

	retval = new(bool)
	lexer_addr = new(*TSLexer)
	state_addr = new(int16)
	result = new(byte)
	skip = new(byte)
	eof = new(byte)
	lookahead = new(int32)
	i = new(int32)
	i203 = new(int32)
	i236 = new(int32)
	*lexer_addr = lexer
	*state_addr = state
	*result = 0
	*skip = 0
	*eof = 0
	goto start

next_state:
	v0 = *lexer_addr
	advance = &v0.F2
	v1 = *advance
	v2 = *lexer_addr
	v3 = *skip
	tobool = (v3 & 1) != 0
	v1(v2, tobool)
	goto start

start:
	*skip = 0
	v4 = *lexer_addr
	lookahead1 = &v4.F0
	v5 = *lookahead1
	*lookahead = v5
	v6 = *lexer_addr
	eof2 = &v6.F6
	v7 = *eof2
	v8 = *lexer_addr
	call = v7(v8)
	if call {
		frombool = 1
	} else {
		frombool = 0
	}
	*eof = frombool
	v9 = *state_addr
	conv = int32(uint32(uint16(v9)))
	switch conv {
	case 0:
		goto sw_bb
	case 1:
		goto sw_bb30
	case 2:
		goto sw_bb62
	case 3:
		goto sw_bb82
	case 4:
		goto sw_bb92
	case 5:
		goto sw_bb106
	case 6:
		goto sw_bb116
	case 7:
		goto sw_bb129
	case 8:
		goto sw_bb142
	case 9:
		goto sw_bb148
	case 10:
		goto sw_bb154
	case 11:
		goto sw_bb160
	case 12:
		goto sw_bb166
	case 13:
		goto sw_bb172
	case 14:
		goto sw_bb178
	case 15:
		goto sw_bb184
	case 16:
		goto sw_bb190
	case 17:
		goto sw_bb196
	case 18:
		goto sw_bb202
	case 19:
		goto sw_bb223
	case 20:
		goto sw_bb232
	case 21:
		goto sw_bb273
	case 22:
		goto sw_bb275
	case 23:
		goto sw_bb279
	case 24:
		goto sw_bb283
	case 25:
		goto sw_bb287
	case 26:
		goto sw_bb291
	case 27:
		goto sw_bb295
	case 28:
		goto sw_bb299
	case 29:
		goto sw_bb303
	case 30:
		goto sw_bb328
	case 31:
		goto sw_bb353
	case 32:
		goto sw_bb374
	case 33:
		goto sw_bb411
	case 34:
		goto sw_bb428
	case 35:
		goto sw_bb432
	case 36:
		goto sw_bb447
	case 37:
		goto sw_bb469
	case 38:
		goto sw_bb487
	case 39:
		goto sw_bb498
	case 40:
		goto sw_bb502
	case 41:
		goto sw_bb506
	case 42:
		goto sw_bb510
	case 43:
		goto sw_bb514
	default:
		goto sw_default
	}

sw_bb:
	v10 = *eof
	tobool3 = (v10 & 1) != 0
	if tobool3 {
		goto if_then
	} else {
		goto if_end
	}

if_then:
	*state_addr = 21
	goto next_state

if_end:
	*i = 0
	goto for_cond

for_cond:
	v11 = *i
	conv4 = int64(uint64(uint32(v11)))
	cmp = uint64(conv4) < uint64(28)
	if cmp {
		goto for_body
	} else {
		goto for_end
	}

for_body:
	v12 = *i
	idxprom = int64(uint64(uint32(v12)))
	arrayidx = &ts_lex_map[idxprom]
	v13 = *arrayidx
	conv6 = int32(uint32(uint16(v13)))
	v14 = *lookahead
	cmp7 = conv6 == v14
	if cmp7 {
		goto if_then9
	} else {
		goto if_end12
	}

if_then9:
	v15 = *i
	add = v15 + 1
	idxprom10 = int64(uint64(uint32(add)))
	arrayidx11 = &ts_lex_map[idxprom10]
	v16 = *arrayidx11
	*state_addr = v16
	goto next_state

if_end12:
	goto for_inc

for_inc:
	v17 = *i
	add13 = v17 + 2
	*i = add13
	goto for_cond

for_end:
	v18 = *lookahead
	cmp14 = 9 <= v18
	if cmp14 {
		goto land_lhs_true
	} else {
		goto lor_lhs_false
	}

land_lhs_true:
	v19 = *lookahead
	cmp16 = v19 <= 13
	if cmp16 {
		goto if_then20
	} else {
		goto lor_lhs_false
	}

lor_lhs_false:
	v20 = *lookahead
	cmp18 = v20 == 32
	if cmp18 {
		goto if_then20
	} else {
		goto if_end21
	}

if_then20:
	*skip = 1
	*state_addr = 20
	goto next_state

if_end21:
	v21 = *lookahead
	cmp22 = 49 <= v21
	if cmp22 {
		goto land_lhs_true24
	} else {
		goto if_end28
	}

land_lhs_true24:
	v22 = *lookahead
	cmp25 = v22 <= 57
	if cmp25 {
		goto if_then27
	} else {
		goto if_end28
	}

if_then27:
	*state_addr = 36
	goto next_state

if_end28:
	v23 = *result
	tobool29 = (v23 & 1) != 0
	*retval = tobool29
	goto _return

sw_bb30:
	v24 = *lookahead
	cmp31 = v24 == 10
	if cmp31 {
		goto if_then33
	} else {
		goto if_end34
	}

if_then33:
	*skip = 1
	*state_addr = 2
	goto next_state

if_end34:
	v25 = *lookahead
	cmp35 = v25 == 34
	if cmp35 {
		goto if_then37
	} else {
		goto if_end38
	}

if_then37:
	*state_addr = 28
	goto next_state

if_end38:
	v26 = *lookahead
	cmp39 = v26 == 47
	if cmp39 {
		goto if_then41
	} else {
		goto if_end42
	}

if_then41:
	*state_addr = 29
	goto next_state

if_end42:
	v27 = *lookahead
	cmp43 = v27 == 92
	if cmp43 {
		goto if_then45
	} else {
		goto if_end46
	}

if_then45:
	*state_addr = 18
	goto next_state

if_end46:
	v28 = *lookahead
	cmp47 = 9 <= v28
	if cmp47 {
		goto land_lhs_true49
	} else {
		goto lor_lhs_false52
	}

land_lhs_true49:
	v29 = *lookahead
	cmp50 = v29 <= 13
	if cmp50 {
		goto if_then55
	} else {
		goto lor_lhs_false52
	}

lor_lhs_false52:
	v30 = *lookahead
	cmp53 = v30 == 32
	if cmp53 {
		goto if_then55
	} else {
		goto if_end56
	}

if_then55:
	*state_addr = 32
	goto next_state

if_end56:
	v31 = *lookahead
	cmp57 = v31 != 0
	if cmp57 {
		goto if_then59
	} else {
		goto if_end60
	}

if_then59:
	*state_addr = 33
	goto next_state

if_end60:
	v32 = *result
	tobool61 = (v32 & 1) != 0
	*retval = tobool61
	goto _return

sw_bb62:
	v33 = *lookahead
	cmp63 = v33 == 34
	if cmp63 {
		goto if_then65
	} else {
		goto if_end66
	}

if_then65:
	*state_addr = 28
	goto next_state

if_end66:
	v34 = *lookahead
	cmp67 = v34 == 47
	if cmp67 {
		goto if_then69
	} else {
		goto if_end70
	}

if_then69:
	*state_addr = 3
	goto next_state

if_end70:
	v35 = *lookahead
	cmp71 = 9 <= v35
	if cmp71 {
		goto land_lhs_true73
	} else {
		goto lor_lhs_false76
	}

land_lhs_true73:
	v36 = *lookahead
	cmp74 = v36 <= 13
	if cmp74 {
		goto if_then79
	} else {
		goto lor_lhs_false76
	}

lor_lhs_false76:
	v37 = *lookahead
	cmp77 = v37 == 32
	if cmp77 {
		goto if_then79
	} else {
		goto if_end80
	}

if_then79:
	*skip = 1
	*state_addr = 2
	goto next_state

if_end80:
	v38 = *result
	tobool81 = (v38 & 1) != 0
	*retval = tobool81
	goto _return

sw_bb82:
	v39 = *lookahead
	cmp83 = v39 == 42
	if cmp83 {
		goto if_then85
	} else {
		goto if_end86
	}

if_then85:
	*state_addr = 5
	goto next_state

if_end86:
	v40 = *lookahead
	cmp87 = v40 == 47
	if cmp87 {
		goto if_then89
	} else {
		goto if_end90
	}

if_then89:
	*state_addr = 43
	goto next_state

if_end90:
	v41 = *result
	tobool91 = (v41 & 1) != 0
	*retval = tobool91
	goto _return

sw_bb92:
	v42 = *lookahead
	cmp93 = v42 == 42
	if cmp93 {
		goto if_then95
	} else {
		goto if_end96
	}

if_then95:
	*state_addr = 4
	goto next_state

if_end96:
	v43 = *lookahead
	cmp97 = v43 == 47
	if cmp97 {
		goto if_then99
	} else {
		goto if_end100
	}

if_then99:
	*state_addr = 42
	goto next_state

if_end100:
	v44 = *lookahead
	cmp101 = v44 != 0
	if cmp101 {
		goto if_then103
	} else {
		goto if_end104
	}

if_then103:
	*state_addr = 5
	goto next_state

if_end104:
	v45 = *result
	tobool105 = (v45 & 1) != 0
	*retval = tobool105
	goto _return

sw_bb106:
	v46 = *lookahead
	cmp107 = v46 == 42
	if cmp107 {
		goto if_then109
	} else {
		goto if_end110
	}

if_then109:
	*state_addr = 4
	goto next_state

if_end110:
	v47 = *lookahead
	cmp111 = v47 != 0
	if cmp111 {
		goto if_then113
	} else {
		goto if_end114
	}

if_then113:
	*state_addr = 5
	goto next_state

if_end114:
	v48 = *result
	tobool115 = (v48 & 1) != 0
	*retval = tobool115
	goto _return

sw_bb116:
	v49 = *lookahead
	cmp117 = v49 == 45
	if cmp117 {
		goto if_then119
	} else {
		goto if_end120
	}

if_then119:
	*state_addr = 19
	goto next_state

if_end120:
	v50 = *lookahead
	cmp121 = 48 <= v50
	if cmp121 {
		goto land_lhs_true123
	} else {
		goto if_end127
	}

land_lhs_true123:
	v51 = *lookahead
	cmp124 = v51 <= 57
	if cmp124 {
		goto if_then126
	} else {
		goto if_end127
	}

if_then126:
	*state_addr = 38
	goto next_state

if_end127:
	v52 = *result
	tobool128 = (v52 & 1) != 0
	*retval = tobool128
	goto _return

sw_bb129:
	v53 = *lookahead
	cmp130 = v53 == 48
	if cmp130 {
		goto if_then132
	} else {
		goto if_end133
	}

if_then132:
	*state_addr = 35
	goto next_state

if_end133:
	v54 = *lookahead
	cmp134 = 49 <= v54
	if cmp134 {
		goto land_lhs_true136
	} else {
		goto if_end140
	}

land_lhs_true136:
	v55 = *lookahead
	cmp137 = v55 <= 57
	if cmp137 {
		goto if_then139
	} else {
		goto if_end140
	}

if_then139:
	*state_addr = 36
	goto next_state

if_end140:
	v56 = *result
	tobool141 = (v56 & 1) != 0
	*retval = tobool141
	goto _return

sw_bb142:
	v57 = *lookahead
	cmp143 = v57 == 97
	if cmp143 {
		goto if_then145
	} else {
		goto if_end146
	}

if_then145:
	*state_addr = 11
	goto next_state

if_end146:
	v58 = *result
	tobool147 = (v58 & 1) != 0
	*retval = tobool147
	goto _return

sw_bb148:
	v59 = *lookahead
	cmp149 = v59 == 101
	if cmp149 {
		goto if_then151
	} else {
		goto if_end152
	}

if_then151:
	*state_addr = 39
	goto next_state

if_end152:
	v60 = *result
	tobool153 = (v60 & 1) != 0
	*retval = tobool153
	goto _return

sw_bb154:
	v61 = *lookahead
	cmp155 = v61 == 101
	if cmp155 {
		goto if_then157
	} else {
		goto if_end158
	}

if_then157:
	*state_addr = 40
	goto next_state

if_end158:
	v62 = *result
	tobool159 = (v62 & 1) != 0
	*retval = tobool159
	goto _return

sw_bb160:
	v63 = *lookahead
	cmp161 = v63 == 108
	if cmp161 {
		goto if_then163
	} else {
		goto if_end164
	}

if_then163:
	*state_addr = 15
	goto next_state

if_end164:
	v64 = *result
	tobool165 = (v64 & 1) != 0
	*retval = tobool165
	goto _return

sw_bb166:
	v65 = *lookahead
	cmp167 = v65 == 108
	if cmp167 {
		goto if_then169
	} else {
		goto if_end170
	}

if_then169:
	*state_addr = 41
	goto next_state

if_end170:
	v66 = *result
	tobool171 = (v66 & 1) != 0
	*retval = tobool171
	goto _return

sw_bb172:
	v67 = *lookahead
	cmp173 = v67 == 108
	if cmp173 {
		goto if_then175
	} else {
		goto if_end176
	}

if_then175:
	*state_addr = 12
	goto next_state

if_end176:
	v68 = *result
	tobool177 = (v68 & 1) != 0
	*retval = tobool177
	goto _return

sw_bb178:
	v69 = *lookahead
	cmp179 = v69 == 114
	if cmp179 {
		goto if_then181
	} else {
		goto if_end182
	}

if_then181:
	*state_addr = 16
	goto next_state

if_end182:
	v70 = *result
	tobool183 = (v70 & 1) != 0
	*retval = tobool183
	goto _return

sw_bb184:
	v71 = *lookahead
	cmp185 = v71 == 115
	if cmp185 {
		goto if_then187
	} else {
		goto if_end188
	}

if_then187:
	*state_addr = 10
	goto next_state

if_end188:
	v72 = *result
	tobool189 = (v72 & 1) != 0
	*retval = tobool189
	goto _return

sw_bb190:
	v73 = *lookahead
	cmp191 = v73 == 117
	if cmp191 {
		goto if_then193
	} else {
		goto if_end194
	}

if_then193:
	*state_addr = 9
	goto next_state

if_end194:
	v74 = *result
	tobool195 = (v74 & 1) != 0
	*retval = tobool195
	goto _return

sw_bb196:
	v75 = *lookahead
	cmp197 = v75 == 117
	if cmp197 {
		goto if_then199
	} else {
		goto if_end200
	}

if_then199:
	*state_addr = 13
	goto next_state

if_end200:
	v76 = *result
	tobool201 = (v76 & 1) != 0
	*retval = tobool201
	goto _return

sw_bb202:
	*i203 = 0
	goto for_cond204

for_cond204:
	v77 = *i203
	conv205 = int64(uint64(uint32(v77)))
	cmp206 = uint64(conv205) < uint64(18)
	if cmp206 {
		goto for_body208
	} else {
		goto for_end221
	}

for_body208:
	v78 = *i203
	idxprom209 = int64(uint64(uint32(v78)))
	arrayidx210 = &ts_lex_map_30[idxprom209]
	v79 = *arrayidx210
	conv211 = int32(uint32(uint16(v79)))
	v80 = *lookahead
	cmp212 = conv211 == v80
	if cmp212 {
		goto if_then214
	} else {
		goto if_end218
	}

if_then214:
	v81 = *i203
	add215 = v81 + 1
	idxprom216 = int64(uint64(uint32(add215)))
	arrayidx217 = &ts_lex_map_30[idxprom216]
	v82 = *arrayidx217
	*state_addr = v82
	goto next_state

if_end218:
	goto for_inc219

for_inc219:
	v83 = *i203
	add220 = v83 + 2
	*i203 = add220
	goto for_cond204

for_end221:
	v84 = *result
	tobool222 = (v84 & 1) != 0
	*retval = tobool222
	goto _return

sw_bb223:
	v85 = *lookahead
	cmp224 = 48 <= v85
	if cmp224 {
		goto land_lhs_true226
	} else {
		goto if_end230
	}

land_lhs_true226:
	v86 = *lookahead
	cmp227 = v86 <= 57
	if cmp227 {
		goto if_then229
	} else {
		goto if_end230
	}

if_then229:
	*state_addr = 38
	goto next_state

if_end230:
	v87 = *result
	tobool231 = (v87 & 1) != 0
	*retval = tobool231
	goto _return

sw_bb232:
	v88 = *eof
	tobool233 = (v88 & 1) != 0
	if tobool233 {
		goto if_then234
	} else {
		goto if_end235
	}

if_then234:
	*state_addr = 21
	goto next_state

if_end235:
	*i236 = 0
	goto for_cond237

for_cond237:
	v89 = *i236
	conv238 = int64(uint64(uint32(v89)))
	cmp239 = uint64(conv238) < uint64(26)
	if cmp239 {
		goto for_body241
	} else {
		goto for_end254
	}

for_body241:
	v90 = *i236
	idxprom242 = int64(uint64(uint32(v90)))
	arrayidx243 = &ts_lex_map_31[idxprom242]
	v91 = *arrayidx243
	conv244 = int32(uint32(uint16(v91)))
	v92 = *lookahead
	cmp245 = conv244 == v92
	if cmp245 {
		goto if_then247
	} else {
		goto if_end251
	}

if_then247:
	v93 = *i236
	add248 = v93 + 1
	idxprom249 = int64(uint64(uint32(add248)))
	arrayidx250 = &ts_lex_map_31[idxprom249]
	v94 = *arrayidx250
	*state_addr = v94
	goto next_state

if_end251:
	goto for_inc252

for_inc252:
	v95 = *i236
	add253 = v95 + 2
	*i236 = add253
	goto for_cond237

for_end254:
	v96 = *lookahead
	cmp255 = 9 <= v96
	if cmp255 {
		goto land_lhs_true257
	} else {
		goto lor_lhs_false260
	}

land_lhs_true257:
	v97 = *lookahead
	cmp258 = v97 <= 13
	if cmp258 {
		goto if_then263
	} else {
		goto lor_lhs_false260
	}

lor_lhs_false260:
	v98 = *lookahead
	cmp261 = v98 == 32
	if cmp261 {
		goto if_then263
	} else {
		goto if_end264
	}

if_then263:
	*skip = 1
	*state_addr = 20
	goto next_state

if_end264:
	v99 = *lookahead
	cmp265 = 49 <= v99
	if cmp265 {
		goto land_lhs_true267
	} else {
		goto if_end271
	}

land_lhs_true267:
	v100 = *lookahead
	cmp268 = v100 <= 57
	if cmp268 {
		goto if_then270
	} else {
		goto if_end271
	}

if_then270:
	*state_addr = 36
	goto next_state

if_end271:
	v101 = *result
	tobool272 = (v101 & 1) != 0
	*retval = tobool272
	goto _return

sw_bb273:
	*result = 1
	v102 = *lexer_addr
	result_symbol = &v102.F1
	*result_symbol = 0
	v103 = *lexer_addr
	mark_end = &v103.F3
	v104 = *mark_end
	v105 = *lexer_addr
	v104(v105)
	v106 = *result
	tobool274 = (v106 & 1) != 0
	*retval = tobool274
	goto _return

sw_bb275:
	*result = 1
	v107 = *lexer_addr
	result_symbol276 = &v107.F1
	*result_symbol276 = 1
	v108 = *lexer_addr
	mark_end277 = &v108.F3
	v109 = *mark_end277
	v110 = *lexer_addr
	v109(v110)
	v111 = *result
	tobool278 = (v111 & 1) != 0
	*retval = tobool278
	goto _return

sw_bb279:
	*result = 1
	v112 = *lexer_addr
	result_symbol280 = &v112.F1
	*result_symbol280 = 2
	v113 = *lexer_addr
	mark_end281 = &v113.F3
	v114 = *mark_end281
	v115 = *lexer_addr
	v114(v115)
	v116 = *result
	tobool282 = (v116 & 1) != 0
	*retval = tobool282
	goto _return

sw_bb283:
	*result = 1
	v117 = *lexer_addr
	result_symbol284 = &v117.F1
	*result_symbol284 = 3
	v118 = *lexer_addr
	mark_end285 = &v118.F3
	v119 = *mark_end285
	v120 = *lexer_addr
	v119(v120)
	v121 = *result
	tobool286 = (v121 & 1) != 0
	*retval = tobool286
	goto _return

sw_bb287:
	*result = 1
	v122 = *lexer_addr
	result_symbol288 = &v122.F1
	*result_symbol288 = 4
	v123 = *lexer_addr
	mark_end289 = &v123.F3
	v124 = *mark_end289
	v125 = *lexer_addr
	v124(v125)
	v126 = *result
	tobool290 = (v126 & 1) != 0
	*retval = tobool290
	goto _return

sw_bb291:
	*result = 1
	v127 = *lexer_addr
	result_symbol292 = &v127.F1
	*result_symbol292 = 5
	v128 = *lexer_addr
	mark_end293 = &v128.F3
	v129 = *mark_end293
	v130 = *lexer_addr
	v129(v130)
	v131 = *result
	tobool294 = (v131 & 1) != 0
	*retval = tobool294
	goto _return

sw_bb295:
	*result = 1
	v132 = *lexer_addr
	result_symbol296 = &v132.F1
	*result_symbol296 = 6
	v133 = *lexer_addr
	mark_end297 = &v133.F3
	v134 = *mark_end297
	v135 = *lexer_addr
	v134(v135)
	v136 = *result
	tobool298 = (v136 & 1) != 0
	*retval = tobool298
	goto _return

sw_bb299:
	*result = 1
	v137 = *lexer_addr
	result_symbol300 = &v137.F1
	*result_symbol300 = 7
	v138 = *lexer_addr
	mark_end301 = &v138.F3
	v139 = *mark_end301
	v140 = *lexer_addr
	v139(v140)
	v141 = *result
	tobool302 = (v141 & 1) != 0
	*retval = tobool302
	goto _return

sw_bb303:
	*result = 1
	v142 = *lexer_addr
	result_symbol304 = &v142.F1
	*result_symbol304 = 8
	v143 = *lexer_addr
	mark_end305 = &v143.F3
	v144 = *mark_end305
	v145 = *lexer_addr
	v144(v145)
	v146 = *lookahead
	cmp306 = v146 == 42
	if cmp306 {
		goto if_then308
	} else {
		goto if_end309
	}

if_then308:
	*state_addr = 31
	goto next_state

if_end309:
	v147 = *lookahead
	cmp310 = v147 == 47
	if cmp310 {
		goto if_then312
	} else {
		goto if_end313
	}

if_then312:
	*state_addr = 33
	goto next_state

if_end313:
	v148 = *lookahead
	cmp314 = v148 != 0
	if cmp314 {
		goto land_lhs_true316
	} else {
		goto if_end326
	}

land_lhs_true316:
	v149 = *lookahead
	cmp317 = v149 != 10
	if cmp317 {
		goto land_lhs_true319
	} else {
		goto if_end326
	}

land_lhs_true319:
	v150 = *lookahead
	cmp320 = v150 != 34
	if cmp320 {
		goto land_lhs_true322
	} else {
		goto if_end326
	}

land_lhs_true322:
	v151 = *lookahead
	cmp323 = v151 != 92
	if cmp323 {
		goto if_then325
	} else {
		goto if_end326
	}

if_then325:
	*state_addr = 33
	goto next_state

if_end326:
	v152 = *result
	tobool327 = (v152 & 1) != 0
	*retval = tobool327
	goto _return

sw_bb328:
	*result = 1
	v153 = *lexer_addr
	result_symbol329 = &v153.F1
	*result_symbol329 = 8
	v154 = *lexer_addr
	mark_end330 = &v154.F3
	v155 = *mark_end330
	v156 = *lexer_addr
	v155(v156)
	v157 = *lookahead
	cmp331 = v157 == 42
	if cmp331 {
		goto if_then333
	} else {
		goto if_end334
	}

if_then333:
	*state_addr = 30
	goto next_state

if_end334:
	v158 = *lookahead
	cmp335 = v158 == 47
	if cmp335 {
		goto if_then337
	} else {
		goto if_end338
	}

if_then337:
	*state_addr = 33
	goto next_state

if_end338:
	v159 = *lookahead
	cmp339 = v159 != 0
	if cmp339 {
		goto land_lhs_true341
	} else {
		goto if_end351
	}

land_lhs_true341:
	v160 = *lookahead
	cmp342 = v160 != 10
	if cmp342 {
		goto land_lhs_true344
	} else {
		goto if_end351
	}

land_lhs_true344:
	v161 = *lookahead
	cmp345 = v161 != 34
	if cmp345 {
		goto land_lhs_true347
	} else {
		goto if_end351
	}

land_lhs_true347:
	v162 = *lookahead
	cmp348 = v162 != 92
	if cmp348 {
		goto if_then350
	} else {
		goto if_end351
	}

if_then350:
	*state_addr = 31
	goto next_state

if_end351:
	v163 = *result
	tobool352 = (v163 & 1) != 0
	*retval = tobool352
	goto _return

sw_bb353:
	*result = 1
	v164 = *lexer_addr
	result_symbol354 = &v164.F1
	*result_symbol354 = 8
	v165 = *lexer_addr
	mark_end355 = &v165.F3
	v166 = *mark_end355
	v167 = *lexer_addr
	v166(v167)
	v168 = *lookahead
	cmp356 = v168 == 42
	if cmp356 {
		goto if_then358
	} else {
		goto if_end359
	}

if_then358:
	*state_addr = 30
	goto next_state

if_end359:
	v169 = *lookahead
	cmp360 = v169 != 0
	if cmp360 {
		goto land_lhs_true362
	} else {
		goto if_end372
	}

land_lhs_true362:
	v170 = *lookahead
	cmp363 = v170 != 10
	if cmp363 {
		goto land_lhs_true365
	} else {
		goto if_end372
	}

land_lhs_true365:
	v171 = *lookahead
	cmp366 = v171 != 34
	if cmp366 {
		goto land_lhs_true368
	} else {
		goto if_end372
	}

land_lhs_true368:
	v172 = *lookahead
	cmp369 = v172 != 92
	if cmp369 {
		goto if_then371
	} else {
		goto if_end372
	}

if_then371:
	*state_addr = 31
	goto next_state

if_end372:
	v173 = *result
	tobool373 = (v173 & 1) != 0
	*retval = tobool373
	goto _return

sw_bb374:
	*result = 1
	v174 = *lexer_addr
	result_symbol375 = &v174.F1
	*result_symbol375 = 8
	v175 = *lexer_addr
	mark_end376 = &v175.F3
	v176 = *mark_end376
	v177 = *lexer_addr
	v176(v177)
	v178 = *lookahead
	cmp377 = v178 == 47
	if cmp377 {
		goto if_then379
	} else {
		goto if_end380
	}

if_then379:
	*state_addr = 29
	goto next_state

if_end380:
	v179 = *lookahead
	cmp381 = v179 == 9
	if cmp381 {
		goto if_then392
	} else {
		goto lor_lhs_false383
	}

lor_lhs_false383:
	v180 = *lookahead
	cmp384 = 11 <= v180
	if cmp384 {
		goto land_lhs_true386
	} else {
		goto lor_lhs_false389
	}

land_lhs_true386:
	v181 = *lookahead
	cmp387 = v181 <= 13
	if cmp387 {
		goto if_then392
	} else {
		goto lor_lhs_false389
	}

lor_lhs_false389:
	v182 = *lookahead
	cmp390 = v182 == 32
	if cmp390 {
		goto if_then392
	} else {
		goto if_end393
	}

if_then392:
	*state_addr = 32
	goto next_state

if_end393:
	v183 = *lookahead
	cmp394 = v183 != 0
	if cmp394 {
		goto land_lhs_true396
	} else {
		goto if_end409
	}

land_lhs_true396:
	v184 = *lookahead
	cmp397 = v184 < 9
	if cmp397 {
		goto land_lhs_true402
	} else {
		goto lor_lhs_false399
	}

lor_lhs_false399:
	v185 = *lookahead
	cmp400 = 13 < v185
	if cmp400 {
		goto land_lhs_true402
	} else {
		goto if_end409
	}

land_lhs_true402:
	v186 = *lookahead
	cmp403 = v186 != 34
	if cmp403 {
		goto land_lhs_true405
	} else {
		goto if_end409
	}

land_lhs_true405:
	v187 = *lookahead
	cmp406 = v187 != 92
	if cmp406 {
		goto if_then408
	} else {
		goto if_end409
	}

if_then408:
	*state_addr = 33
	goto next_state

if_end409:
	v188 = *result
	tobool410 = (v188 & 1) != 0
	*retval = tobool410
	goto _return

sw_bb411:
	*result = 1
	v189 = *lexer_addr
	result_symbol412 = &v189.F1
	*result_symbol412 = 8
	v190 = *lexer_addr
	mark_end413 = &v190.F3
	v191 = *mark_end413
	v192 = *lexer_addr
	v191(v192)
	v193 = *lookahead
	cmp414 = v193 != 0
	if cmp414 {
		goto land_lhs_true416
	} else {
		goto if_end426
	}

land_lhs_true416:
	v194 = *lookahead
	cmp417 = v194 != 10
	if cmp417 {
		goto land_lhs_true419
	} else {
		goto if_end426
	}

land_lhs_true419:
	v195 = *lookahead
	cmp420 = v195 != 34
	if cmp420 {
		goto land_lhs_true422
	} else {
		goto if_end426
	}

land_lhs_true422:
	v196 = *lookahead
	cmp423 = v196 != 92
	if cmp423 {
		goto if_then425
	} else {
		goto if_end426
	}

if_then425:
	*state_addr = 33
	goto next_state

if_end426:
	v197 = *result
	tobool427 = (v197 & 1) != 0
	*retval = tobool427
	goto _return

sw_bb428:
	*result = 1
	v198 = *lexer_addr
	result_symbol429 = &v198.F1
	*result_symbol429 = 9
	v199 = *lexer_addr
	mark_end430 = &v199.F3
	v200 = *mark_end430
	v201 = *lexer_addr
	v200(v201)
	v202 = *result
	tobool431 = (v202 & 1) != 0
	*retval = tobool431
	goto _return

sw_bb432:
	*result = 1
	v203 = *lexer_addr
	result_symbol433 = &v203.F1
	*result_symbol433 = 10
	v204 = *lexer_addr
	mark_end434 = &v204.F3
	v205 = *mark_end434
	v206 = *lexer_addr
	v205(v206)
	v207 = *lookahead
	cmp435 = v207 == 46
	if cmp435 {
		goto if_then437
	} else {
		goto if_end438
	}

if_then437:
	*state_addr = 37
	goto next_state

if_end438:
	v208 = *lookahead
	cmp439 = v208 == 69
	if cmp439 {
		goto if_then444
	} else {
		goto lor_lhs_false441
	}

lor_lhs_false441:
	v209 = *lookahead
	cmp442 = v209 == 101
	if cmp442 {
		goto if_then444
	} else {
		goto if_end445
	}

if_then444:
	*state_addr = 6
	goto next_state

if_end445:
	v210 = *result
	tobool446 = (v210 & 1) != 0
	*retval = tobool446
	goto _return

sw_bb447:
	*result = 1
	v211 = *lexer_addr
	result_symbol448 = &v211.F1
	*result_symbol448 = 10
	v212 = *lexer_addr
	mark_end449 = &v212.F3
	v213 = *mark_end449
	v214 = *lexer_addr
	v213(v214)
	v215 = *lookahead
	cmp450 = v215 == 46
	if cmp450 {
		goto if_then452
	} else {
		goto if_end453
	}

if_then452:
	*state_addr = 37
	goto next_state

if_end453:
	v216 = *lookahead
	cmp454 = v216 == 69
	if cmp454 {
		goto if_then459
	} else {
		goto lor_lhs_false456
	}

lor_lhs_false456:
	v217 = *lookahead
	cmp457 = v217 == 101
	if cmp457 {
		goto if_then459
	} else {
		goto if_end460
	}

if_then459:
	*state_addr = 6
	goto next_state

if_end460:
	v218 = *lookahead
	cmp461 = 48 <= v218
	if cmp461 {
		goto land_lhs_true463
	} else {
		goto if_end467
	}

land_lhs_true463:
	v219 = *lookahead
	cmp464 = v219 <= 57
	if cmp464 {
		goto if_then466
	} else {
		goto if_end467
	}

if_then466:
	*state_addr = 36
	goto next_state

if_end467:
	v220 = *result
	tobool468 = (v220 & 1) != 0
	*retval = tobool468
	goto _return

sw_bb469:
	*result = 1
	v221 = *lexer_addr
	result_symbol470 = &v221.F1
	*result_symbol470 = 10
	v222 = *lexer_addr
	mark_end471 = &v222.F3
	v223 = *mark_end471
	v224 = *lexer_addr
	v223(v224)
	v225 = *lookahead
	cmp472 = v225 == 69
	if cmp472 {
		goto if_then477
	} else {
		goto lor_lhs_false474
	}

lor_lhs_false474:
	v226 = *lookahead
	cmp475 = v226 == 101
	if cmp475 {
		goto if_then477
	} else {
		goto if_end478
	}

if_then477:
	*state_addr = 6
	goto next_state

if_end478:
	v227 = *lookahead
	cmp479 = 48 <= v227
	if cmp479 {
		goto land_lhs_true481
	} else {
		goto if_end485
	}

land_lhs_true481:
	v228 = *lookahead
	cmp482 = v228 <= 57
	if cmp482 {
		goto if_then484
	} else {
		goto if_end485
	}

if_then484:
	*state_addr = 37
	goto next_state

if_end485:
	v229 = *result
	tobool486 = (v229 & 1) != 0
	*retval = tobool486
	goto _return

sw_bb487:
	*result = 1
	v230 = *lexer_addr
	result_symbol488 = &v230.F1
	*result_symbol488 = 10
	v231 = *lexer_addr
	mark_end489 = &v231.F3
	v232 = *mark_end489
	v233 = *lexer_addr
	v232(v233)
	v234 = *lookahead
	cmp490 = 48 <= v234
	if cmp490 {
		goto land_lhs_true492
	} else {
		goto if_end496
	}

land_lhs_true492:
	v235 = *lookahead
	cmp493 = v235 <= 57
	if cmp493 {
		goto if_then495
	} else {
		goto if_end496
	}

if_then495:
	*state_addr = 38
	goto next_state

if_end496:
	v236 = *result
	tobool497 = (v236 & 1) != 0
	*retval = tobool497
	goto _return

sw_bb498:
	*result = 1
	v237 = *lexer_addr
	result_symbol499 = &v237.F1
	*result_symbol499 = 11
	v238 = *lexer_addr
	mark_end500 = &v238.F3
	v239 = *mark_end500
	v240 = *lexer_addr
	v239(v240)
	v241 = *result
	tobool501 = (v241 & 1) != 0
	*retval = tobool501
	goto _return

sw_bb502:
	*result = 1
	v242 = *lexer_addr
	result_symbol503 = &v242.F1
	*result_symbol503 = 12
	v243 = *lexer_addr
	mark_end504 = &v243.F3
	v244 = *mark_end504
	v245 = *lexer_addr
	v244(v245)
	v246 = *result
	tobool505 = (v246 & 1) != 0
	*retval = tobool505
	goto _return

sw_bb506:
	*result = 1
	v247 = *lexer_addr
	result_symbol507 = &v247.F1
	*result_symbol507 = 13
	v248 = *lexer_addr
	mark_end508 = &v248.F3
	v249 = *mark_end508
	v250 = *lexer_addr
	v249(v250)
	v251 = *result
	tobool509 = (v251 & 1) != 0
	*retval = tobool509
	goto _return

sw_bb510:
	*result = 1
	v252 = *lexer_addr
	result_symbol511 = &v252.F1
	*result_symbol511 = 14
	v253 = *lexer_addr
	mark_end512 = &v253.F3
	v254 = *mark_end512
	v255 = *lexer_addr
	v254(v255)
	v256 = *result
	tobool513 = (v256 & 1) != 0
	*retval = tobool513
	goto _return

sw_bb514:
	*result = 1
	v257 = *lexer_addr
	result_symbol515 = &v257.F1
	*result_symbol515 = 14
	v258 = *lexer_addr
	mark_end516 = &v258.F3
	v259 = *mark_end516
	v260 = *lexer_addr
	v259(v260)
	v261 = *lookahead
	cmp517 = v261 != 0
	if cmp517 {
		goto land_lhs_true519
	} else {
		goto if_end523
	}

land_lhs_true519:
	v262 = *lookahead
	cmp520 = v262 != 10
	if cmp520 {
		goto if_then522
	} else {
		goto if_end523
	}

if_then522:
	*state_addr = 43
	goto next_state

if_end523:
	v263 = *result
	tobool524 = (v263 & 1) != 0
	*retval = tobool524
	goto _return

sw_default:
	*retval = false
	goto _return

_return:
	v264 = *retval
	return v264
}
