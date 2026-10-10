/* Required libgobject entry points. g_object_new is the fixed 4-argument shape this package calls.
   The 6-argument call stays a separate registration because a C prototype has one signature. */
uint64_t g_signal_connect_data(uintptr_t obj, char *name, uintptr_t handler, uintptr_t data, uintptr_t destroy, uint32_t flags);
void g_object_unref(uintptr_t obj);
void g_value_init(uintptr_t value, uintptr_t gtype);
void g_value_set_boolean(uintptr_t value, int32_t v);
void g_value_unset(uintptr_t value);
void g_object_set_property(uintptr_t object, char *name, uintptr_t value);
uintptr_t g_object_new(uintptr_t object_type, char *property, uintptr_t value, uintptr_t terminator);
