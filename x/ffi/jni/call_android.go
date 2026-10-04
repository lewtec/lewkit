//go:build android && cgo

package jni

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

const (
	sigString        = "()Ljava/lang/String;"
	sigClassLoader   = "()Ljava/lang/ClassLoader;"
	sigLoadClass     = "(Ljava/lang/String;)Ljava/lang/Class;"
	sigMethods       = "()[Ljava/lang/reflect/Method;"
	sigCtors         = "()[Ljava/lang/reflect/Constructor;"
	sigClasses       = "()[Ljava/lang/Class;"
	sigClass         = "()Ljava/lang/Class;"
	sigMods          = "()I"
	sigInvoke        = "(Ljava/lang/Object;[Ljava/lang/Object;)Ljava/lang/Object;"
	sigNewInstance   = "([Ljava/lang/Object;)Ljava/lang/Object;"
	sigCause         = "()Ljava/lang/Throwable;"
	sigBoolValueOf   = "(Z)Ljava/lang/Boolean;"
	sigIntValueOf    = "(I)Ljava/lang/Integer;"
	sigLongValueOf   = "(J)Ljava/lang/Long;"
	sigFloatValueOf  = "(F)Ljava/lang/Float;"
	sigDoubleValueOf = "(D)Ljava/lang/Double;"
)

var (
	errUnbound   = errors.New("jni class loader is not bound")
	errNullCtor  = errors.New("constructor returned null")
	errAttach    = errors.New("jni attach")
	errNoLoader  = errors.New("class loader is null")
	errNoMethods = errors.New("no methods")

	bindMu     sync.Mutex
	bound      atomic.Pointer[vm]
	currentEnv atomic.Value
)

// vm is the bound process: one app ClassLoader and the boot method ids.
type vm struct {
	errIDs
	loader          uintptr
	object          uintptr
	booleanClass    uintptr
	integerClass    uintptr
	longClass       uintptr
	floatClass      uintptr
	doubleClass     uintptr
	getClassLoader  uintptr
	loadClass       uintptr
	getMethods      uintptr
	getConstructors uintptr
	getName         uintptr
	methodName      uintptr
	methodParams    uintptr
	methodReturn    uintptr
	methodModifiers uintptr
	invoke          uintptr
	ctorParams      uintptr
	newInstance     uintptr
	boolValueOf     uintptr
	intValueOf      uintptr
	longValueOf     uintptr
	floatValueOf    uintptr
	doubleValueOf   uintptr
	booleanValue    uintptr
	intValue        uintptr
	longValue       uintptr
	floatValue      uintptr
	doubleValue     uintptr
}

type held struct {
	candidate
	obj uintptr
}

type scanSpec struct {
	params    uintptr
	nameID    uintptr
	returnID  uintptr
	modsID    uintptr
	fixedName string
	fixedRet  string
	checkMods bool
}

// SetCurrentEnv supplies the JNIEnv for the calling goroutine.
// The function attaches the goroutine when it is not already on a Java thread.
func SetCurrentEnv(fn func() uintptr) {
	if fn == nil {
		return
	}
	currentEnv.Store(fn)
}

func bind(raw uintptr, anchor string) error {
	bindMu.Lock()
	defer bindMu.Unlock()
	if bound.Load() != nil {
		return nil
	}
	e := openEnv(raw)
	if e.tab == 0 {
		return errNoEnv
	}
	if err := e.push(256); err != nil {
		return err
	}
	defer e.pop()
	vm, err := e.bootstrap(anchor)
	if err != nil {
		return err
	}
	javaRaw.Store(raw)
	bound.Store(vm)
	return nil
}

func callStatic(className, method string, args ...any) (any, error) {
	return dispatch(call{class: className, name: method, static: true, args: args})
}

func callNew(className string, args ...any) (*Ref, error) {
	v, err := dispatch(call{class: className, name: "<init>", ctor: true, args: args})
	if err != nil {
		return nil, err
	}
	ref, ok := v.(*Ref)
	if !ok || ref == nil {
		return nil, errNullCtor
	}
	return ref, nil
}

func callRef(r *Ref, method string, args ...any) (any, error) {
	return dispatch(call{recv: r.ptr, name: method, args: args})
}

func classObject(name string) (*Ref, error) {
	if bound.Load() == nil {
		return nil, errUnbound
	}
	v, err := withJava(func(e env) (any, error) {
		vm := bound.Load()
		if vm == nil {
			return nil, errUnbound
		}
		if err := e.push(32); err != nil {
			return nil, err
		}
		defer e.pop()
		local, err := e.load(vm, name)
		if err != nil {
			return nil, err
		}
		return e.asRef(vm, local)
	})
	if err != nil {
		return nil, err
	}
	ref, ok := v.(*Ref)
	if !ok || ref == nil {
		return nil, errNullClass
	}
	return ref, nil
}

func staticField(className, name string) (any, error) {
	cls, err := classObject(className)
	if err != nil {
		return nil, err
	}
	defer cls.Release()
	return readField(cls, nil, name)
}

func instanceField(r *Ref, name string) (any, error) {
	got, err := callRef(r, "getClass")
	if err != nil {
		return nil, err
	}
	cls, ok := got.(*Ref)
	if !ok || cls == nil {
		return nil, errNullClass
	}
	defer cls.Release()
	return readField(cls, r, name)
}

func readField(cls *Ref, recv *Ref, name string) (any, error) {
	got, err := callRef(cls, "getField", name)
	if err != nil {
		return nil, err
	}
	field, ok := got.(*Ref)
	if !ok || field == nil {
		return nil, errNullClass
	}
	defer field.Release()
	var target any
	if recv != nil {
		target = recv
	}
	return callRef(field, "get", target)
}

func release(r *Ref) bool {
	_, err := withJava(func(e env) (any, error) {
		dropProxy(r.ptr)
		e.deleteGlobal(r.ptr)
		return nil, nil
	})
	return err == nil
}

// withJava runs fn on the Bind thread. The JNIEnv is the one Bind saved.
func withJava(fn func(env) (any, error)) (any, error) {
	var (
		v   any
		err error
	)
	runOnJava(func() {
		e, openErr := envHere()
		if openErr != nil {
			err = openErr
			return
		}
		v, err = fn(e)
	})
	return v, err
}

func envHere() (env, error) {
	raw := javaRaw.Load()
	if raw == 0 {
		fn, ok := currentEnv.Load().(func() uintptr)
		if !ok || fn == nil {
			return env{}, errNoEnv
		}
		raw = fn()
		if raw == 0 {
			return env{}, errAttach
		}
	}
	e := openEnv(raw)
	if e.tab == 0 {
		return env{}, errNoEnv
	}
	return e, nil
}

func (e env) bootstrap(anchor string) (*vm, error) {
	state := &vm{}
	var gs []uintptr
	fail := func(err error) (*vm, error) {
		for _, g := range gs {
			e.deleteGlobal(g)
		}
		return nil, err
	}
	keep := func(local uintptr) (uintptr, error) {
		g, err := e.global(local, state.errIDs)
		if err != nil {
			return 0, err
		}
		gs = append(gs, g)
		return g, nil
	}

	var (
		err                                                      error
		objectCls, boolCls, intCls, longCls, floatCls, doubleCls uintptr
	)
	objectCls, state.toString, err = e.pair("java.lang.Object", "toString", sigString, false, state.errIDs)
	if err != nil {
		return fail(err)
	}
	_, state.getCause, err = e.pair("java.lang.Throwable", "getCause", sigCause, false, state.errIDs)
	if err != nil {
		return fail(err)
	}
	_, state.getClassLoader, err = e.pair("java.lang.Class", "getClassLoader", sigClassLoader, false, state.errIDs)
	if err != nil {
		return fail(err)
	}
	_, state.loadClass, err = e.pair("java.lang.ClassLoader", "loadClass", sigLoadClass, false, state.errIDs)
	if err != nil {
		return fail(err)
	}
	_, state.getMethods, err = e.pair("java.lang.Class", "getMethods", sigMethods, false, state.errIDs)
	if err != nil {
		return fail(err)
	}
	_, state.getConstructors, err = e.pair("java.lang.Class", "getConstructors", sigCtors, false, state.errIDs)
	if err != nil {
		return fail(err)
	}
	_, state.getName, err = e.pair("java.lang.Class", "getName", sigString, false, state.errIDs)
	if err != nil {
		return fail(err)
	}
	_, state.methodName, err = e.pair("java.lang.reflect.Method", "getName", sigString, false, state.errIDs)
	if err != nil {
		return fail(err)
	}
	_, state.methodParams, err = e.pair("java.lang.reflect.Method", "getParameterTypes", sigClasses, false, state.errIDs)
	if err != nil {
		return fail(err)
	}
	_, state.methodReturn, err = e.pair("java.lang.reflect.Method", "getReturnType", sigClass, false, state.errIDs)
	if err != nil {
		return fail(err)
	}
	_, state.methodModifiers, err = e.pair("java.lang.reflect.Method", "getModifiers", sigMods, false, state.errIDs)
	if err != nil {
		return fail(err)
	}
	_, state.invoke, err = e.pair("java.lang.reflect.Method", "invoke", sigInvoke, false, state.errIDs)
	if err != nil {
		return fail(err)
	}
	_, state.ctorParams, err = e.pair("java.lang.reflect.Constructor", "getParameterTypes", sigClasses, false, state.errIDs)
	if err != nil {
		return fail(err)
	}
	_, state.newInstance, err = e.pair("java.lang.reflect.Constructor", "newInstance", sigNewInstance, false, state.errIDs)
	if err != nil {
		return fail(err)
	}
	boolCls, state.boolValueOf, err = e.pair("java.lang.Boolean", "valueOf", sigBoolValueOf, true, state.errIDs)
	if err != nil {
		return fail(err)
	}
	_, state.booleanValue, err = e.pair("java.lang.Boolean", "booleanValue", "()Z", false, state.errIDs)
	if err != nil {
		return fail(err)
	}
	intCls, state.intValueOf, err = e.pair("java.lang.Integer", "valueOf", sigIntValueOf, true, state.errIDs)
	if err != nil {
		return fail(err)
	}
	longCls, state.longValueOf, err = e.pair("java.lang.Long", "valueOf", sigLongValueOf, true, state.errIDs)
	if err != nil {
		return fail(err)
	}
	floatCls, state.floatValueOf, err = e.pair("java.lang.Float", "valueOf", sigFloatValueOf, true, state.errIDs)
	if err != nil {
		return fail(err)
	}
	doubleCls, state.doubleValueOf, err = e.pair("java.lang.Double", "valueOf", sigDoubleValueOf, true, state.errIDs)
	if err != nil {
		return fail(err)
	}
	_, state.intValue, err = e.pair("java.lang.Number", "intValue", "()I", false, state.errIDs)
	if err != nil {
		return fail(err)
	}
	_, state.longValue, err = e.pair("java.lang.Number", "longValue", "()J", false, state.errIDs)
	if err != nil {
		return fail(err)
	}
	_, state.floatValue, err = e.pair("java.lang.Number", "floatValue", "()F", false, state.errIDs)
	if err != nil {
		return fail(err)
	}
	_, state.doubleValue, err = e.pair("java.lang.Number", "doubleValue", "()D", false, state.errIDs)
	if err != nil {
		return fail(err)
	}

	anchorCls, err := e.find(anchor, state.errIDs)
	if err != nil {
		return fail(err)
	}
	loader, err := e.callObject(idxCallObjectA, e.self, anchorCls, state.getClassLoader, nil, state.errIDs)
	if err != nil {
		return fail(err)
	}
	if loader == 0 {
		return fail(errNoLoader)
	}
	if state.loader, err = keep(loader); err != nil {
		return fail(err)
	}
	if state.object, err = keep(objectCls); err != nil {
		return fail(err)
	}
	if state.booleanClass, err = keep(boolCls); err != nil {
		return fail(err)
	}
	if state.integerClass, err = keep(intCls); err != nil {
		return fail(err)
	}
	if state.longClass, err = keep(longCls); err != nil {
		return fail(err)
	}
	if state.floatClass, err = keep(floatCls); err != nil {
		return fail(err)
	}
	if state.doubleClass, err = keep(doubleCls); err != nil {
		return fail(err)
	}
	return state, nil
}

func (e env) pair(class, name, sig string, static bool, ids errIDs) (uintptr, uintptr, error) {
	cls, err := e.find(class, ids)
	if err != nil {
		return 0, 0, err
	}
	slot := idxGetMethodID
	if static {
		slot = idxGetStaticMethodID
	}
	mid, err := e.methodID(slot, cls, name, sig, ids)
	if err != nil {
		return 0, 0, err
	}
	return cls, mid, nil
}

type call struct {
	class  string
	recv   uintptr
	name   string
	static bool
	ctor   bool
	args   []any
}

func dispatch(in call) (any, error) {
	vals := make([]value, len(in.args))
	for i, arg := range in.args {
		v, err := classify(arg)
		if err != nil {
			return nil, err
		}
		vals[i] = v
	}
	if bound.Load() == nil {
		return nil, errUnbound
	}
	return withJava(func(e env) (any, error) {
		vm := bound.Load()
		if vm == nil {
			return nil, errUnbound
		}
		if err := e.push(256); err != nil {
			return nil, err
		}
		defer e.pop()

		var cls uintptr
		var err error
		if in.ctor || in.static {
			cls, err = e.load(vm, in.class)
		} else {
			cls, err = e.objectClass(in.recv, vm.errIDs)
		}
		if err != nil {
			return nil, err
		}

		spec := methodSpec(vm)
		list := vm.getMethods
		wantStatic := in.static
		if in.ctor {
			spec = ctorSpec(vm, dotted(in.class))
			list = vm.getConstructors
			wantStatic = false
		}
		helds, err := e.scan(cls, list, vm, spec, in.name, wantStatic, len(vals))
		if err != nil {
			return nil, err
		}
		defer freeHeld(e, helds)

		as, err := e.argList(vm, vals, helds)
		if err != nil {
			return nil, err
		}
		idx, err := pick(candsOf(helds), selector{name: in.name, static: wantStatic, args: as})
		if err != nil {
			return nil, err
		}
		boxed := make([]uintptr, len(vals))
		for i, v := range vals {
			p, err := e.box(vm, v)
			if err != nil {
				return nil, err
			}
			boxed[i] = p
		}
		arr, err := e.objectArray(vm, boxed)
		if err != nil {
			return nil, err
		}
		var result uintptr
		if in.ctor {
			result, err = e.callObject(idxCallObjectA, e.self, helds[idx].obj, vm.newInstance, []jvalue{jobj(arr)}, vm.errIDs)
		} else {
			receiver := uintptr(0)
			if !in.static {
				receiver = in.recv
			}
			result, err = e.callObject(idxCallObjectA, e.self, helds[idx].obj, vm.invoke, []jvalue{jobj(receiver), jobj(arr)}, vm.errIDs)
		}
		if err != nil {
			return nil, err
		}
		return e.unbox(vm, result, helds[idx].ret, in.ctor)
	})
}

func methodSpec(vm *vm) scanSpec {
	return scanSpec{
		params:    vm.methodParams,
		nameID:    vm.methodName,
		returnID:  vm.methodReturn,
		modsID:    vm.methodModifiers,
		checkMods: true,
	}
}

func ctorSpec(vm *vm, ret string) scanSpec {
	return scanSpec{
		params:    vm.ctorParams,
		fixedName: "<init>",
		fixedRet:  ret,
	}
}

func (e env) scan(cls, listID uintptr, vm *vm, spec scanSpec, want string, wantStatic bool, nargs int) ([]held, error) {
	if err := e.push(16); err != nil {
		return nil, err
	}
	defer e.pop()
	arr, err := e.callObject(idxCallObjectA, e.self, cls, listID, nil, vm.errIDs)
	if err != nil {
		return nil, err
	}
	if arr == 0 {
		return nil, errNoMethods
	}
	n, err := e.arrayLen(arr, vm.errIDs)
	if err != nil {
		return nil, err
	}
	var out []held
	for i := int32(0); i < n; i++ {
		h, keep, err := e.readOne(arr, i, vm, spec, want, wantStatic, nargs)
		if err != nil {
			freeHeld(e, out)
			return nil, err
		}
		if keep {
			out = append(out, h)
		}
	}
	return out, nil
}

func (e env) readOne(arr uintptr, i int32, vm *vm, spec scanSpec, want string, wantStatic bool, nargs int) (held, bool, error) {
	if err := e.push(256); err != nil {
		return held{}, false, err
	}
	defer e.pop()
	m, err := e.element(arr, i, vm.errIDs)
	if err != nil || m == 0 {
		return held{}, false, err
	}
	name := spec.fixedName
	if spec.nameID != 0 {
		name, err = e.javaString(m, spec.nameID, vm)
		if err != nil {
			return held{}, false, err
		}
	}
	if name != want {
		return held{}, false, nil
	}
	if spec.checkMods {
		mods, err := e.callInt(m, spec.modsID, nil, vm.errIDs)
		if err != nil {
			return held{}, false, err
		}
		if !keepMember(mods, wantStatic) {
			return held{}, false, nil
		}
	}
	params, err := e.readParams(m, spec.params, vm)
	if err != nil {
		return held{}, false, err
	}
	if len(params) != nargs {
		return held{}, false, nil
	}
	ret := spec.fixedRet
	if spec.returnID != 0 {
		rc, err := e.callObject(idxCallObjectA, e.self, m, spec.returnID, nil, vm.errIDs)
		if err != nil {
			return held{}, false, err
		}
		ret, err = e.className(rc, vm)
		if err != nil {
			return held{}, false, err
		}
	}
	g, err := e.global(m, vm.errIDs)
	if err != nil {
		return held{}, false, err
	}
	return held{
		candidate: candidate{name: name, static: wantStatic, params: params, ret: ret},
		obj:       g,
	}, true, nil
}

func (e env) readParams(obj, paramsID uintptr, vm *vm) ([]string, error) {
	arr, err := e.callObject(idxCallObjectA, e.self, obj, paramsID, nil, vm.errIDs)
	if err != nil {
		return nil, err
	}
	if arr == 0 {
		return nil, nil
	}
	n, err := e.arrayLen(arr, vm.errIDs)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, n)
	for i := int32(0); i < n; i++ {
		cls, err := e.element(arr, i, vm.errIDs)
		if err != nil {
			return nil, err
		}
		name, err := e.className(cls, vm)
		if err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, nil
}

func (e env) argList(vm *vm, vals []value, helds []held) ([]arg, error) {
	cache := map[string]uintptr{}
	out := make([]arg, len(vals))
	for i, v := range vals {
		a := arg{kind: v.kind, class: v.class}
		if v.kind == kindRef {
			from, err := e.objectClass(v.ref, vm.errIDs)
			if err != nil {
				return nil, err
			}
			fit := map[string]bool{}
			for _, h := range helds {
				for _, p := range h.params {
					if isPrimitive(p) {
						continue
					}
					if _, seen := fit[p]; seen {
						continue
					}
					pc, err := e.cachedClass(vm, p, cache)
					if err != nil {
						return nil, err
					}
					fit[p] = e.isAssign(from, pc)
				}
			}
			a.assignable = func(param string) bool { return fit[param] }
		}
		out[i] = a
	}
	return out, nil
}

func (e env) cachedClass(vm *vm, name string, cache map[string]uintptr) (uintptr, error) {
	if cls, ok := cache[name]; ok {
		return cls, nil
	}
	cls, err := e.load(vm, name)
	if err != nil {
		return 0, err
	}
	cache[name] = cls
	return cls, nil
}

func (e env) load(vm *vm, name string) (uintptr, error) {
	js, err := e.newString(dotted(name), vm.errIDs)
	if err != nil {
		return 0, err
	}
	cls, err := e.callObject(idxCallObjectA, e.self, vm.loader, vm.loadClass, []jvalue{jobj(js)}, vm.errIDs)
	if err != nil {
		return 0, err
	}
	if cls == 0 {
		return 0, fmt.Errorf("%w: %s", errClass, dotted(name))
	}
	return cls, nil
}

func (e env) box(vm *vm, v value) (uintptr, error) {
	var (
		p   uintptr
		err error
	)
	switch v.kind {
	case kindNil:
		return 0, nil
	case kindRef:
		return v.ref, nil
	case kindString:
		return e.newString(v.text, vm.errIDs)
	case kindBytes:
		return e.newByteArray(v.raw, vm.errIDs)
	case kindBool:
		p, err = e.callObject(idxCallStaticObjectA, e.self, vm.booleanClass, vm.boolValueOf, []jvalue{jbool(v.b)}, vm.errIDs)
	case kindInt:
		p, err = e.callObject(idxCallStaticObjectA, e.self, vm.integerClass, vm.intValueOf, []jvalue{ji32(v.i32)}, vm.errIDs)
	case kindLong:
		p, err = e.callObject(idxCallStaticObjectA, e.self, vm.longClass, vm.longValueOf, []jvalue{ji64(v.i64)}, vm.errIDs)
	case kindFloat:
		p, err = e.callObject(idxCallStaticObjectA, e.self, vm.floatClass, vm.floatValueOf, []jvalue{jf32(v.f32)}, vm.errIDs)
	case kindDouble:
		p, err = e.callObject(idxCallStaticObjectA, e.self, vm.doubleClass, vm.doubleValueOf, []jvalue{jf64(v.f64)}, vm.errIDs)
	default:
		return 0, errJava
	}
	if err != nil {
		return 0, err
	}
	if p == 0 {
		return 0, errJava
	}
	return p, nil
}

func (e env) objectArray(vm *vm, items []uintptr) (uintptr, error) {
	if len(items) == 0 {
		return 0, nil
	}
	arr, err := e.newObjectArray(int32(len(items)), vm.object, vm.errIDs)
	if err != nil {
		return 0, err
	}
	for i, it := range items {
		if err := e.setElement(arr, int32(i), it, vm.errIDs); err != nil {
			return 0, err
		}
	}
	return arr, nil
}

func (e env) unbox(vm *vm, obj uintptr, declared string, forceRef bool) (any, error) {
	if forceRef {
		if obj == 0 {
			return nil, errNullCtor
		}
		return e.asRef(vm, obj)
	}
	if declared == "void" || obj == 0 {
		return nil, nil
	}
	cls, err := e.objectClass(obj, vm.errIDs)
	if err != nil {
		return nil, err
	}
	name, err := e.className(cls, vm)
	if err != nil {
		return nil, err
	}
	switch name {
	case "java.lang.String":
		return e.readString(obj)
	case "java.lang.Boolean":
		return e.callBool(obj, vm.booleanValue, vm.errIDs)
	case "java.lang.Byte", "java.lang.Short", "java.lang.Integer":
		n, err := e.callInt(obj, vm.intValue, nil, vm.errIDs)
		if err != nil {
			return nil, err
		}
		return int(n), nil
	case "java.lang.Long":
		return e.callLong(obj, vm.longValue, vm.errIDs)
	case "java.lang.Float":
		return e.callFloat(obj, vm.floatValue, vm.errIDs)
	case "java.lang.Double":
		return e.callDouble(obj, vm.doubleValue, vm.errIDs)
	case "java.lang.Character":
		return e.javaString(obj, vm.toString, vm)
	default:
		return e.asRef(vm, obj)
	}
}

func (e env) asRef(vm *vm, obj uintptr) (*Ref, error) {
	cls, err := e.objectClass(obj, vm.errIDs)
	if err != nil {
		return nil, err
	}
	name, err := e.className(cls, vm)
	if err != nil {
		return nil, err
	}
	g, err := e.global(obj, vm.errIDs)
	if err != nil {
		return nil, err
	}
	return &Ref{ptr: g, class: name}, nil
}

func (e env) className(cls uintptr, vm *vm) (string, error) {
	return e.javaString(cls, vm.getName, vm)
}

func (e env) javaString(obj, mid uintptr, vm *vm) (string, error) {
	s, err := e.callObject(idxCallObjectA, e.self, obj, mid, nil, vm.errIDs)
	if err != nil {
		return "", err
	}
	return e.readString(s)
}

func candsOf(hs []held) []candidate {
	out := make([]candidate, len(hs))
	for i, h := range hs {
		out[i] = h.candidate
	}
	return out
}

func freeHeld(e env, hs []held) {
	for _, h := range hs {
		e.deleteGlobal(h.obj)
	}
}
