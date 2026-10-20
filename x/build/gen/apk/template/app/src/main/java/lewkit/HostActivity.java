package lewkit;

import android.app.Activity;
import android.content.Intent;
import android.os.Bundle;
import android.view.Window;

/**
 * HostActivity is a window the process tracks.
 * Resume makes it the foreground activity the document picker stacks on.
 * The picker result goes back through FileChooser, which Go already calls.
 */
public class HostActivity extends Activity {
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        // The packed manifest theme is not applied (the activity record
        // stays theme 0), so the default theme draws a title bar. No title
        // wins over that action bar when the decor is created.
        requestWindowFeature(Window.FEATURE_NO_TITLE);
        super.onCreate(savedInstanceState);
    }

    @Override
    protected void onResume() {
        super.onResume();
        Host.noteForeground(this, true);
    }

    @Override
    protected void onPause() {
        Host.noteForeground(this, false);
        super.onPause();
    }

    @SuppressWarnings("deprecation")
    @Override
    protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        if (FileChooser.onResult(this, requestCode, resultCode, data)) {
            return;
        }
        super.onActivityResult(requestCode, resultCode, data);
    }

    @SuppressWarnings("deprecation")
    @Override
    public void onRequestPermissionsResult(int requestCode, String[] permissions, int[] grantResults) {
        if (Notify.onResult(requestCode, grantResults)) {
            return;
        }
        super.onRequestPermissionsResult(requestCode, permissions, grantResults);
    }

    @Override
    protected void onDestroy() {
        FileChooser.hostGone(this);
        super.onDestroy();
    }
}
