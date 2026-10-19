#include <stddef.h>
#include <string.h>
/* A grow-free byte buffer and the spans copied into it. */

#define CAP \
	(1 << 8)
#define ALIGN (sizeof(long))
#define EMPTY

struct Buf;

typedef struct Span {
	const char *data;
	long len;
} Span;

typedef struct Buf {
	char *data;
	long len;
	long cap;
	struct Buf *next;
} Buf;

typedef struct Doc {
	Span title;
	struct {
		long major;
		long minor;
	} version;
	Buf *pages[2];
} Doc;

enum Status {
	STATUS_OK = 0,
	STATUS_FULL = 1,
	STATUS_BAD = 4
};

typedef unsigned long Size;
typedef int (*WriteFn)(void *ctx, Span span);

static const char mark = 'x';
static const char *greeting = "hello\n";
static double ratio = 1.25;
static unsigned mask = 0xff;
static Size span_size = sizeof(Span);
static char scratch[CAP];
static char pad[STATUS_BAD];
static Span table[4];
extern volatile long ticks;
int room_a = 1, room_b = 2;

static int log_line(const char *fmt, ...) {
	(void)fmt;
	return STATUS_OK;
}

static long span_len(Span s) {
	if (s.data == 0 || s.len < 0) {
		return 0;
	}
	return s.len;
}

static int write_span(void *ctx, Span span) {
	Buf *buf;
	long need;
	long i;
	buf = (Buf *)ctx;
	if (!buf || span.data == 0) {
		return STATUS_BAD;
	}
	need = buf->len + span.len;
	if (need > buf->cap || need > CAP) {
		return STATUS_FULL;
	}
	i = 0;
	while (i < span.len && i < buf->cap) {
		buf->data[buf->len] = span.data[i];
		buf->len = buf->len + 1;
		i = i + 1;
		if ((i & 1) == 0) {
			continue;
		}
	}
	return STATUS_OK;
}

static int copy_all(Buf *buf, Span *items, long n, WriteFn write) {
	long i;
	int rc;
	rc = STATUS_OK;
	for (i = 0; i < n; i = i + 1) {
		if (items[i].len < 0) {
			continue;
		} else if (write == 0) {
			rc = write_span(buf, items[i]);
		} else {
			rc = write(buf, items[i]);
		}
		if (rc != STATUS_OK) {
			break;
		}
	}
	if (buf != 0 && buf->len > 0) {
		Size local;
		{
			local = sizeof(buf->data);
			rc = (int)local;
		}
		log_line(greeting);
	} else {
		log_line("empty");
	}
	return rc;
}

long buf_room(const Buf *buf) {
	if (buf == 0) {
		return 0;
	}
	return buf->cap - buf->len;
}

const char *buf_get(const Buf *buf, long at) {
	if (buf == 0 || at < 0 || at >= buf->len) {
		return 0;
	}
	return &buf->data[at];
}

static int flag_on(unsigned flags, unsigned bit) {
	return (flags & bit) != 0 || (flags >> 1) == bit;
}

static inline int doc_ok(const Doc *doc) {
	if (doc == 0) {
		return 0;
	}
	return doc->version.major > 0 && doc->pages[0] != 0;
}
