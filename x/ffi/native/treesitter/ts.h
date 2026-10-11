/* Subset of libtree-sitter used by package treesitter. */
typedef struct {
	uint32_t Context[4];
	uintptr_t ID;
	uintptr_t Tree;
} RawNode;

uintptr_t ts_parser_new(void);
void ts_parser_delete(uintptr_t parser);
bool ts_parser_set_language(uintptr_t parser, uintptr_t lang);
uintptr_t ts_parser_parse_string(uintptr_t parser, uintptr_t old, char *str, uint32_t length);
void ts_tree_delete(uintptr_t tree);
RawNode ts_tree_root_node(uintptr_t tree);
uintptr_t ts_node_type(RawNode node);
uint32_t ts_node_start_byte(RawNode node);
uint32_t ts_node_end_byte(RawNode node);
uint32_t ts_node_child_count(RawNode node);
RawNode ts_node_child(RawNode node, uint32_t index);
uintptr_t ts_node_field_name_for_child(RawNode node, uint32_t index);
uint32_t ts_node_named_child_count(RawNode node);
RawNode ts_node_named_child(RawNode node, uint32_t index);
uintptr_t ts_node_string(RawNode node);
bool ts_node_is_null(RawNode node);
bool ts_node_is_named(RawNode node);
bool ts_node_is_extra(RawNode node);
bool ts_node_is_error(RawNode node);
bool ts_node_has_error(RawNode node);
