package lewkit;

import android.app.Activity;
import android.content.Intent;

/**
 * HostActivity is a window the process tracks.
 * Resume makes it the foreground activity the document picker stacks on.
 * The picker result goes back through FileChooser, which Go already calls.
 */
public class HostActivity extends Activity {
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

    @Override
    protected void onDestroy() {
        FileChooser.hostGone(this);
        super.onDestroy();
    }
}
