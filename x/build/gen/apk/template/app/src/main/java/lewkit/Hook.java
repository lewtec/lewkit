package lewkit;

// Hook is the one native Java calls to reach Go.
// A cgo build exports call. A cgo-free build registers it from JNI_OnLoad,
// so the host methods do not each need a cgo export.
public final class Hook {
    private Hook() {}

    public static native long call(String name, Object[] args);
}
