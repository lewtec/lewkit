package lewkit;

import android.app.Activity;
import android.content.Context;
import android.content.Intent;
import android.os.Handler;
import android.os.Looper;
import android.view.Surface;

public final class Host {
    public static volatile Context app;
    public static volatile Ready onReady;
    public static volatile Fail onFail;
    public static volatile Activity foreground;

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
            if (fg != null) {
                fg.startActivity(new Intent(fg, SurfaceActivity.class));
                return;
            }
            Context ctx = app;
            if (ctx == null) {
                return;
            }
            ctx.startActivity(
                    new Intent(ctx, SurfaceActivity.class).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK));
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
