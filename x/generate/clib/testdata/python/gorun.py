"""Python calls for the cgo-free shared library."""

import ctypes


class Job(ctypes.Structure):
    _fields_ = [
        ("n", ctypes.c_int32),
        ("seed", ctypes.c_int32),
    ]


class Vec(ctypes.Structure):
    _fields_ = [
        ("data", ctypes.POINTER(ctypes.c_int32)),
        ("n", ctypes.c_int32),
    ]


class Tally(ctypes.Structure):
    _fields_ = [
        ("sum", ctypes.c_int64),
        ("workers", ctypes.c_int32),
    ]


class Library:
    def __init__(self, path):
        lib = ctypes.CDLL(path)
        lib.fan.argtypes = [Job]
        lib.fan.restype = Tally
        lib.dot.argtypes = [Vec, Vec]
        lib.dot.restype = Tally
        lib.meet.argtypes = [Job]
        lib.meet.restype = Tally
        self._lib = lib

    def fan(self, n, seed):
        return self._lib.fan(Job(int(n), int(seed)))

    def dot(self, a, b):
        if len(a) != len(b):
            raise ValueError("vectors differ in length")
        aa, bb = array(a), array(b)
        return self._lib.dot(as_vec(aa), as_vec(bb))

    def meet(self, n):
        return self._lib.meet(Job(int(n), 0))


def array(values):
    n = len(values)
    if n == 0:
        return None
    return (ctypes.c_int32 * n)(*[int(x) for x in values])


def as_vec(raw):
    if raw is None:
        return Vec()
    return Vec(ctypes.cast(raw, ctypes.POINTER(ctypes.c_int32)), len(raw))


def load(path):
    return Library(path)
