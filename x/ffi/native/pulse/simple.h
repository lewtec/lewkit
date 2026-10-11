/* Subset of libpulse-simple used by package pulse. */
typedef struct {
	int32_t format;
	uint32_t rate;
	uint8_t channels;
} sample_spec;

typedef struct {
	uint32_t max;
	uint32_t target;
	uint32_t prebuf;
	uint32_t minreq;
	uint32_t frag;
} buffer_attr;

uintptr_t pa_simple_new(uintptr_t server, uintptr_t name, int32_t dir, uintptr_t dev, uintptr_t stream, sample_spec *ss, uintptr_t map, buffer_attr *attr, int32_t *error);
void pa_simple_free(uintptr_t s);
int32_t pa_simple_write(uintptr_t s, uintptr_t data, uintptr_t bytes, int32_t *error);
int32_t pa_simple_drain(uintptr_t s, int32_t *error);
