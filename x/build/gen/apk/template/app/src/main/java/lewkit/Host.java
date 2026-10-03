package lewkit;

import android.app.Activity;
import android.content.Context;
import android.content.Intent;
import android.os.Handler;
import android.os.Looper;
import android.view.Surface;

import java.io.File;

public final class Host {
    public static volatile Context app;
    public static volatile Ready onReady;
    public static volatile Fail onFail;
    public static volatile Activity foreground;
    private static boolean booted;

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
        new Handler(Looper.getMainLooper()).post(() -> {
            Activity fg = foreground;
            if (fg instanceof SurfaceActivity) {
                return;
            }
            Context ctx = fg != null ? fg : app;
            if (ctx == null) {
                return;
            }
            Intent intent = new Intent(ctx, SurfaceActivity.class);
            if (fg == null) {
                intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
            }
            ctx.startActivity(intent);
        });
    }

    public static void ready(String url) {
        Ready cb = onReady;
        if (cb == null) {
            return;
        }
        new Handler(Looper.getMainLooper()).post(() -> cb.call(url));
    }

    public static void fail(String message) {
        Fail cb = onFail;
        if (cb == null) {
            return;
        }
        new Handler(Looper.getMainLooper()).post(() -> cb.call(message));
    }

    public static native void start(String readyFile);

    public static native long nativeWindow(Surface surface, int width, int height);

    public static native void pointer(int x, int y, int action);

    public static native void resize(int width, int height);

    public static native void surfaceLost();
}
