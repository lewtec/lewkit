#ifndef SAMPLE_H
#define SAMPLE_H

#include "extra.h"

#define ANSWER 42
#define FLAG 0x10
#define SUM (1 << 2)

#if defined(NOPE)
typedef int no_flag;
#else
typedef int yes_flag;
#endif

typedef struct Point {
	char tag;
	int x;
	long y;
	double z;
} Point;

typedef union Slot {
	int i;
	void *p;
	double d;
} Slot;

enum Color {
	RED = 1,
	GREEN,
	BLUE = 1 << 2
};

struct Packed {
	char a;
	int b;
} __attribute__((packed));

struct Outer {
	struct {
		char a;
		int b;
	} inner;
	char c;
	void (*hook)(Point *p, int n);
	char name[8];
	int flex[];
};

struct Bit {
	unsigned a : 3;
	unsigned b : 5;
	char c;
};

struct Bit2 {
	unsigned a : 3;
	unsigned b : 30;
	char c;
};

struct Aligned {
	char a;
	int b __attribute__((aligned(8)));
};

struct Hold {
	char a;
	struct {
		int x;
		char y;
	} z;
	short s;
};

struct Arr {
	int *c[2];
	int (*d)[2];
	int (*f[3])(int);
};

struct R {
	unsigned a : 3;
	char b;
} __attribute__((packed));

struct Fwd;
typedef struct Fwd *Handle;

int point_sum(const Point *p, unsigned n);
void *slot_ptr(Slot s);
int variadic(int n, ...);
static int hidden(int a);
int answer_value;

#endif
