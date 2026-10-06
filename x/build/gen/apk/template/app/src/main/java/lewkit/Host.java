package lewkit;

import android.app.Activity;
import android.app.AlertDialog;
import android.content.Context;
import android.content.Intent;
import android.os.Handler;
import android.os.Looper;
import android.view.Surface;

import java.io.File;

public final class Host {
    public static volatile Context app;
    public static volatile Ready onReady;
    public static volatile Ready onPage;
    public static volatile Fail onFail;
    public static volatile Activity foreground;
    private static boolean booted;
    private static boolean surfaceOpening;

    public interface Ready {
        void call(String url);
    }

    public interface Fail {
        void call(String message);
    }

    private Host() {}

    public static void load() {
        System.loadLibrary("eletrocromo");
    }

    // boot loads the Go library once and starts it. The calling activity is foreground.
    public static synchronized void boot(Context context) {
        if (context == null) {
            return;
        }
        app = context.getApplicationContext();
        if (context instanceof Activity) {
            noteForeground((Activity) context, true);
        }
        if (booted) {
            return;
        }
        load();
        booted = true;
        File file = new File(context.getCacheDir(), "eletrocromo-ready");
        file.delete();
        Thread go = new Thread(() -> start(file.getAbsolutePath()), "lewkit-go");
        go.setDaemon(true);
        go.start();
    }

    public static void noteForeground(Activity activity, boolean visible) {
        if (visible) {
            foreground = activity;
        } else if (foreground == activity) {
            foreground = null;
        }
    }

    public static void openSurface() {
        if (Looper.myLooper() == Looper.getMainLooper()) {
            openSurfaceNow();
            return;
        }
        new Handler(Looper.getMainLooper()).post(Host::openSurfaceNow);
    }

    private static void openSurfaceNow() {
        if (foreground instanceof SurfaceActivity || surfaceOpening) {
            return;
        }
        Context ctx = app != null ? app : foreground;
        if (ctx == null) {
            return;
        }
        surfaceOpening = true;
        ctx.startActivity(new Intent(ctx, SurfaceActivity.class).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK));
        // The first window is up. The splash activity replaces itself with it.
        Ready cb = onReady;
        if (cb != null) {
            onReady = null;
            onFail = null;
            cb.call("");
        }
    }

    // ready delivers the first window to the splash. A URL after the splash
    // has closed opens another page. An empty URL is the native surface.
    public static void ready(String url) {
        Ready splash = onReady;
        if (splash != null) {
            onReady = null;
            onFail = null;
            new Handler(Looper.getMainLooper()).post(() -> splash.call(url));
            return;
        }
        if (url == null || url.isEmpty()) {
            return;
        }
        Ready page = onPage;
        if (page == null) {
            return;
        }
        new Handler(Looper.getMainLooper()).post(() -> page.call(url));
    }

    public static void fail(String message) {
        String text = message == null || message.isEmpty() ? "The app stopped." : message;
        new Handler(Looper.getMainLooper()).post(() -> showFail(text));
    }

    private static void showFail(String text) {
        Fail cb = onFail;
        if (cb != null) {
            cb.call(text);
        }
        Activity activity = foreground;
        if (activity == null || activity.isFinishing()) {
            return;
        }
        new AlertDialog.Builder(activity)
                .setTitle("Error")
                .setMessage(text)
                .setPositiveButton(android.R.string.ok, null)
                .show();
    }

    public static native void start(String readyFile);

    public static native long nativeWindow(Surface surface, int width, int height);

    public static native void pointer(int x, int y, int action);

    public static native void resize(int width, int height);

    public static native void surfaceLost();

    public static native void obscure(int left, int top, int right, int bottom, int width, int height);
}
