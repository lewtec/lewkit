package lewkit;

import java.lang.reflect.InvocationHandler;
import java.lang.reflect.Method;
import java.lang.reflect.Proxy;

// GoProxy is the one Java object Go uses to implement an interface.
// nativeInvoke is the only JNI entry added for callbacks.
public final class GoProxy implements InvocationHandler {
    private final long id;

    private GoProxy(long id) {
        this.id = id;
    }

    public static Object create(Class<?> iface, long id) {
        ClassLoader loader = iface.getClassLoader();
        if (loader == null) {
            loader = GoProxy.class.getClassLoader();
        }
        return Proxy.newProxyInstance(loader, new Class<?>[] {iface}, new GoProxy(id));
    }

    @Override
    public Object invoke(Object proxy, Method method, Object[] args) {
        String name = method.getName();
        if ("equals".equals(name) && args != null && args.length == 1) {
            return Boolean.valueOf(proxy == args[0]);
        }
        if ("hashCode".equals(name) && (args == null || args.length == 0)) {
            return Integer.valueOf(System.identityHashCode(proxy));
        }
        if ("toString".equals(name) && (args == null || args.length == 0)) {
            return proxy.getClass().getName() + "@" + Integer.toHexString(System.identityHashCode(proxy));
        }
        if (args == null) {
            args = new Object[0];
        }
        return nativeInvoke(id, name, args);
    }

    private static native Object nativeInvoke(long id, String name, Object[] args);
}
