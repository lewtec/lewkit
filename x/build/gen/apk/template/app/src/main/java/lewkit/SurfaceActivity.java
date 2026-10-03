package lewkit;

import android.content.res.Configuration;
import android.graphics.PixelFormat;
import android.os.Bundle;
import android.view.MotionEvent;
import android.view.SurfaceHolder;
import android.view.SurfaceView;
import android.view.View;

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
        view.getHolder().setFormat(PixelFormat.OPAQUE);
        view.getHolder().addCallback(this);
    }

    @Override
    public void onConfigurationChanged(Configuration newConfig) {
        super.onConfigurationChanged(newConfig);
        View content = findViewById(android.R.id.content);
        if (content != null) {
            content.requestLayout();
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
            return;
        }
        if (same) {
            return;
        }
        Host.nativeWindow(holder.getSurface(), width, height);
    }

    @Override
    public void surfaceDestroyed(SurfaceHolder holder) {
        alive = false;
        Host.surfaceLost();
    }
}
