// csmith 2.3.0 (conda-forge), seed 44.
// Fixed-width typedefs stand in for csmith.h.
// Brace initializers are omitted. Compound for-steps are plain assignments.
// A unary minus of a cast is parenthesized so a prefix operator can take it.

typedef signed char int8_t;
typedef unsigned char uint8_t;
typedef short int16_t;
typedef unsigned short uint16_t;
typedef int int32_t;
typedef unsigned int uint32_t;
typedef long long int64_t;
typedef unsigned long long uint64_t;



static long __undefined;


struct S0 {
   int64_t  f0;
   int16_t  f1;
   int32_t  f2;
};

struct S1 {
   uint64_t  f0;
   struct S0  f1;
};

struct S2 {
   uint32_t  f0;
   volatile uint32_t  f1;
   struct S1  f2;
   uint32_t  f3;
};


static uint32_t g_14[1] ;
static struct S1 g_22 ;
static struct S1 *g_21 = &g_22;
static int32_t g_31 = 0xCB2EC6F0L;
static int32_t ** volatile g_33 = (void*)0;
static int32_t *g_35 = &g_31;
static int32_t ** volatile g_34[4] ;
static const struct S2 g_39 ;
static struct S2 g_50 ;
static struct S0 ** const  volatile g_56 = (void*)0;
static struct S0 *g_58 = &g_22.f1;
static struct S0 ** volatile g_57 = &g_58;
static int16_t *g_61 = &g_50.f2.f1.f1;
static int16_t **g_60 = &g_61;
static struct S1 *g_64[1] ;
static const struct S0 ** volatile g_68 = (void*)0;
static int32_t ** volatile g_72[2][4] ;
static int32_t ** volatile g_80 = &g_35;
static int32_t ** volatile g_94 = &g_35;
static struct S1 *g_97 = (void*)0;
static int32_t * volatile g_104 = &g_31;
static const int32_t *g_109 = &g_31;



static int32_t  func_1(void);
static struct S1 * func_2(struct S1  p_3, int16_t  p_4, int16_t  p_5);
static struct S1  func_6(struct S1 * const  p_7, struct S1 * p_8, int32_t  p_9);
static struct S1 * const  func_10(int64_t  p_11, struct S1 * p_12, struct S0  p_13);
static struct S1 * func_15(struct S1 * p_16, struct S1 * p_17);
static struct S1 * func_18(struct S1 * p_19, struct S1 * p_20);




static int32_t  func_1(void)
{ 
    struct S1 **l_23 = &g_21;
    int32_t l_98 = 3L;
    int64_t *l_106 = (void*)0;
    int32_t *l_107 = (void*)0;
    const int32_t *l_108[2][4] ;
    int i, j;
    (*l_23) = func_2(func_6(func_10(g_14[0], func_15(func_18(((*l_23) = g_21), &g_22), g_64[0]), g_39.f2.f1), g_97, l_98), l_98, l_98);
    l_98 = (((void*)0 == l_106) ^ 0UL);
    g_109 = l_108[1][1];
    return (*g_104);
}



static struct S1 * func_2(struct S1  p_3, int16_t  p_4, int16_t  p_5)
{ 
    int32_t l_103 = 0xC4A87E63L;
    struct S1 *l_105 = (void*)0;
    (*g_104) = ((int64_t)p_5 - (int64_t)(0x99C33BC337E5C358LL < l_103));
    return l_105;
}



static struct S1  func_6(struct S1 * const  p_7, struct S1 * p_8, int32_t  p_9)
{ 
    struct S1 l_99 ;
    struct S1 *l_100 = &l_99;
    (*l_100) = ((*p_7) = l_99);
    return (*p_7);
}



static struct S1 * const  func_10(int64_t  p_11, struct S1 * p_12, struct S0  p_13)
{ 
    if (p_13.f1)
    { 
        int32_t *l_93 = &g_31;
        struct S2 *l_96 = &g_50;
        struct S2 **l_95[4][3] ;
        int i, j;
        (*g_94) = l_93;
        l_95[1][2] = l_95[3][0];
        (*g_21) = (*g_21);
    }
    else
    { 
        return p_12;
    }
    return &g_22;
}



static struct S1 * func_15(struct S1 * p_16, struct S1 * p_17)
{ 
    const struct S0 *l_67 = &g_39.f2.f1;
    int32_t l_70 = 0xEF6617FAL;
    int32_t *l_71 = &l_70;
    struct S1 *l_90[1][2];
    struct S1 *l_92 = &g_50.f2;
    int i, j;
    for (i = 0; i < 1; i = i + 1)
    {
        for (j = 0; j < 2; j = j + 1)
            l_90[i][j] = &g_50.f2;
    }
    for (g_22.f0 = 0; (g_22.f0 == 7); g_22.f0 = f0 + 2)
    { 
        const struct S0 **l_69 = &l_67;
        (*l_69) = l_67;
        if (l_70)
            continue;
    }
    l_71 = l_71;
    for (g_50.f2.f1.f2 = (-17); (g_50.f2.f1.f2 == (-19)); g_50.f2.f1.f2 = f2 - 1)
    { 
        int64_t l_86 = (-1L);
        struct S1 **l_91[3];
        int i;
        for (i = 0; i < 3; i = i + 1)
            l_91[i] = &l_90[0][1];
        for (g_22.f1.f1 = 0; (g_22.f1.f1 < 6); g_22.f1.f1 = f1 + 1)
        { 
            uint32_t l_88 = 4294967295UL;
            int16_t **l_89 = &g_61;
            for (g_50.f2.f1.f0 = 7; (g_50.f2.f1.f0 < (-1)); g_50.f2.f1.f0 = f0 - 1)
            { 
                int32_t *l_79[1];
                int64_t *l_87 = &g_22.f1.f0;
                int i;
                for (i = 0; i < 1; i = i + 1)
                    l_79[i] = &l_70;
                (*g_80) = l_79[0];
                (*g_35) = (((uint64_t)(-((int64_t)((*l_87) = ((int16_t)l_86 << (int16_t)((void*)0 != &p_16))))) - (uint64_t)1UL) & l_88);
            }
            l_89 = &g_61;
        }
        g_21 = l_90[0][1];
        g_64[0] = p_17;
    }
    return l_92;
}



static struct S1 * func_18(struct S1 * p_19, struct S1 * p_20)
{ 
    int16_t l_28 = 0L;
    int16_t *l_29 = &g_22.f1.f1;
    int32_t *l_30 = &g_31;
    struct S2 *l_49 = &g_50;
    int32_t **l_62 = &l_30;
    int32_t *l_63[2];
    int i;
    for (i = 0; i < 2; i = i + 1)
        l_63[i] = &g_31;
    if (((*l_30) = ((int16_t)((*l_29) = ((((int32_t)(-1L) - (int32_t)(-1L)) & (((l_28 ^ (-2L)) ^ 0xC0105807L) >= l_28)) == l_28)) * (int16_t)l_28)))
    { 
        struct S1 l_32[4] ;
        const struct S2 *l_38 = &g_39;
        const struct S2 **l_37 = &l_38;
        int32_t l_48 = 0L;
        int i;
        for (g_22.f1.f0 = 0; (g_22.f1.f0 <= 3); g_22.f1.f0 = f0 + 1)
        { 
            int32_t **l_36 = &l_30;
            int i;
            p_19 = &l_32[g_22.f1.f0];
            (*l_36) = l_30;
            return &g_22;
        }
        (*l_37) = (void*)0;
        (*g_35) = (-((uint16_t)(((((*l_30) ^ (((int64_t)(((((int16_t)(&p_20 == &g_21) >> (int16_t)(!((int64_t)(0x153EL | 0x70B1L) % (int64_t)(*l_30)))) && l_48) <= (*l_30)) || 6UL) + (int64_t)l_48) < (*l_30))) <= (*l_30)) | 8L) || 0xAAC5L)));
    }
    else
    { 
        struct S2 *l_51[2][2] ;
        int16_t **l_59 = &l_29;
        int i, j;
        l_51[0][1] = l_49;
        for (g_31 = 23; (g_31 < (-1)); g_31 = g_31 - 1)
        { 
            uint64_t l_54 = 5UL;
            if (l_54)
                break;
            if (l_54)
            { 
                return &g_22;
            }
            else
            { 
                struct S0 *l_55 = &g_50.f2.f1;
                (*g_57) = l_55;
                g_60 = l_59;
            }
        }
    }
    l_63[1] = ((*l_62) = l_30);
    return &g_22;
}





int main (int argc, char* argv[])
{
    int i, j;
    int print_hash_value = 0;
    if (argc == 2 && strcmp(argv[1], "1") == 0) print_hash_value = 1;
    platform_main_begin();
    crc32_gentab();
    func_1();
    for (i = 0; i < 1; i = i + 1)
    {
        transparent_crc(g_14[i], "g_14[i]", print_hash_value);
        if (print_hash_value) printf("index = [%d]\n", i);

    }
    transparent_crc(g_22.f0, "g_22.f0", print_hash_value);
    transparent_crc(g_22.f1.f0, "g_22.f1.f0", print_hash_value);
    transparent_crc(g_22.f1.f1, "g_22.f1.f1", print_hash_value);
    transparent_crc(g_22.f1.f2, "g_22.f1.f2", print_hash_value);
    transparent_crc(g_31, "g_31", print_hash_value);
    transparent_crc(g_39.f0, "g_39.f0", print_hash_value);
    transparent_crc(g_39.f1, "g_39.f1", print_hash_value);
    transparent_crc(g_39.f2.f0, "g_39.f2.f0", print_hash_value);
    transparent_crc(g_39.f2.f1.f0, "g_39.f2.f1.f0", print_hash_value);
    transparent_crc(g_39.f2.f1.f1, "g_39.f2.f1.f1", print_hash_value);
    transparent_crc(g_39.f2.f1.f2, "g_39.f2.f1.f2", print_hash_value);
    transparent_crc(g_39.f3, "g_39.f3", print_hash_value);
    transparent_crc(g_50.f0, "g_50.f0", print_hash_value);
    transparent_crc(g_50.f1, "g_50.f1", print_hash_value);
    transparent_crc(g_50.f2.f0, "g_50.f2.f0", print_hash_value);
    transparent_crc(g_50.f2.f1.f0, "g_50.f2.f1.f0", print_hash_value);
    transparent_crc(g_50.f2.f1.f1, "g_50.f2.f1.f1", print_hash_value);
    transparent_crc(g_50.f2.f1.f2, "g_50.f2.f1.f2", print_hash_value);
    transparent_crc(g_50.f3, "g_50.f3", print_hash_value);
    platform_main_end(crc32_context ^ 0xFFFFFFFFUL, print_hash_value);
    return 0;
}
