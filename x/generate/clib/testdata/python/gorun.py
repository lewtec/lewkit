"""Python calls for the cgo-free shared library."""

import ctypes


class Library:
    def __init__(self, path):
        lib = ctypes.CDLL(path)
        lib.fan.argtypes = [ctypes.c_int32, ctypes.c_int32]
        lib.fan.restype = ctypes.c_int64
        lib.dot.argtypes = [
            ctypes.POINTER(ctypes.c_int32),
            ctypes.POINTER(ctypes.c_int32),
            ctypes.c_int32,
        ]
        lib.dot.restype = ctypes.c_int64
        lib.meet.argtypes = [ctypes.c_int32]
        lib.meet.restype = ctypes.c_int32
        self._lib = lib

    def fan(self, n, seed):
        return int(self._lib.fan(int(n), int(seed)))

    def dot(self, a, b):
        if len(a) != len(b):
            raise ValueError("vectors differ in length")
        n = len(a)
        if n == 0:
            return int(self._lib.dot(None, None, 0))
        aa = (ctypes.c_int32 * n)(*[int(x) for x in a])
        bb = (ctypes.c_int32 * n)(*[int(x) for x in b])
        return int(self._lib.dot(aa, bb, n))

    def meet(self, n):
        return int(self._lib.meet(int(n)))


def load(path):
    return Library(path)
