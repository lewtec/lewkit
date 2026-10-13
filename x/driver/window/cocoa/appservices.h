/* TransformProcessType from ApplicationServices. */
typedef struct {
	uint32_t high;
	uint32_t low;
} ProcessSerial;

int32_t TransformProcessType(ProcessSerial *psn, uint32_t state);
