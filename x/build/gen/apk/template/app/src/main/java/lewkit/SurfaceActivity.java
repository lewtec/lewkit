package lewkit;

import android.content.res.Configuration;
import android.graphics.Insets;
import android.graphics.PixelFormat;
import android.os.Build;
import android.os.Bundle;
import android.view.DisplayCutout;
import android.view.MotionEvent;
import android.view.SurfaceHolder;
import android.view.SurfaceView;
import android.view.View;
import android.view.WindowInsets;

/** A Vulkan surface window. The Go app presents into the native window. */
public final class SurfaceActivity extends HostActivity implements SurfaceHolder.Callback {
    private boolean sent;
    private boolean alive;
    private int lastW;
    private int lastH;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        Host.boot(this);
        SurfaceView view = new SurfaceView(this);
        view.setZOrderOnTop(true);
        view.setClickable(true);
        view.setOnTouchListener((v, event) -> {
            int action;
            switch (event.getActionMasked()) {
                case MotionEvent.ACTION_DOWN:
                    action = 0;
                    break;
                case MotionEvent.ACTION_UP:
                case MotionEvent.ACTION_CANCEL:
                    action = 1;
                    break;
                case MotionEvent.ACTION_MOVE:
                    action = 2;
                    break;
                default:
                    return false;
            }
            Host.pointer((int) event.getX(), (int) event.getY(), action);
            return true;
        });
        setContentView(view);
        view.setOnApplyWindowInsetsListener((v, insets) -> {
            reportDead(v, insets);
            return insets;
        });
        view.getHolder().setFormat(PixelFormat.OPAQUE);
        view.getHolder().addCallback(this);
    }

    /** Bars and cutouts are pixels of this view that are not fully shown. */
    private static void reportDead(View view, WindowInsets insets) {
        int width = view.getWidth();
        int height = view.getHeight();
        if (width < 1 || height < 1 || insets == null) {
            return;
        }
        int left = insets.getSystemWindowInsetLeft();
        int top = insets.getSystemWindowInsetTop();
        int right = insets.getSystemWindowInsetRight();
        int bottom = insets.getSystemWindowInsetBottom();
        if (Build.VERSION.SDK_INT >= 30) {
            Insets bars = insets.getInsets(WindowInsets.Type.systemBars());
            Insets cut = insets.getInsets(WindowInsets.Type.displayCutout());
            left = Math.max(bars.left, cut.left);
            top = Math.max(bars.top, cut.top);
            right = Math.max(bars.right, cut.right);
            bottom = Math.max(bars.bottom, cut.bottom);
        } else if (Build.VERSION.SDK_INT >= 28) {
            DisplayCutout cutout = insets.getDisplayCutout();
            if (cutout != null) {
                left = Math.max(left, cutout.getSafeInsetLeft());
                top = Math.max(top, cutout.getSafeInsetTop());
                right = Math.max(right, cutout.getSafeInsetRight());
                bottom = Math.max(bottom, cutout.getSafeInsetBottom());
            }
        }
        Host.obscure(left, top, right, bottom, width, height);
    }

    @Override
    public void onConfigurationChanged(Configuration newConfig) {
        super.onConfigurationChanged(newConfig);
        View content = findViewById(android.R.id.content);
        if (content != null) {
            content.requestLayout();
            content.requestApplyInsets();
        }
    }

    private void requestDead() {
        View content = findViewById(android.R.id.content);
        if (content != null) {
            content.requestApplyInsets();
        }
    }

    @Override
    public void surfaceCreated(SurfaceHolder holder) {}

    @Override
    public void surfaceChanged(SurfaceHolder holder, int format, int width, int height) {
        if (width < 64 || height < 64) {
            return;
        }
        boolean same = sent && alive && width == lastW && height == lastH;
        lastW = width;
        lastH = height;
        alive = true;
        if (!sent) {
            sent = true;
            Host.nativeWindow(holder.getSurface(), width, height);
            requestDead();
            return;
        }
        if (same) {
            return;
        }
        Host.nativeWindow(holder.getSurface(), width, height);
        requestDead();
    }

    @Override
    public void surfaceDestroyed(SurfaceHolder holder) {
        alive = false;
        Host.surfaceLost();
    }
}
