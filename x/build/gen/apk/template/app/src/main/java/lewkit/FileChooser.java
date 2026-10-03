package lewkit;

import android.app.Activity;
import android.content.ActivityNotFoundException;
import android.content.ClipData;
import android.content.Intent;
import android.net.Uri;
import android.os.Handler;
import android.os.Looper;
import android.provider.DocumentsContract;
import android.webkit.MimeTypeMap;

import java.util.ArrayList;
import java.util.List;
import java.util.Locale;
import java.util.function.BiConsumer;

/**
 * Shows the system document picker over the foreground activity.
 * A file, a folder, or a save name is the activity result. The picker pops
 * itself and the activity underneath stays. Go sets the listener, then open.
 * The listener receives content URIs, or "canceled", "no activity", or "no picker".
 */
public final class FileChooser {
    public static final int REQUEST = 1;

    private static volatile BiConsumer<String, String> listener;
    private static Activity pickerHost;
    private static boolean pending;
    private static boolean write;

    private FileChooser() {}

    public static void setListener(BiConsumer<String, String> next) {
        listener = next;
    }

    public static void open(
            String title,
            String directory,
            String name,
            String extensions,
            boolean multiple,
            boolean folder,
            boolean save) {
        new Handler(Looper.getMainLooper()).post(
                () -> launch(title, directory, name, extensions, multiple, folder, save));
    }

    public static void cancel() {
        new Handler(Looper.getMainLooper()).post(() -> {
            Activity activity = pickerHost;
            if (activity != null && pending) {
                activity.finishActivity(REQUEST);
            }
            deliver("canceled", "");
        });
    }

    /**
     * onResult accepts the picker result for the activity that launched it.
     * A true result means this request was the document picker.
     */
    public static boolean onResult(Activity activity, int requestCode, int resultCode, Intent data) {
        if (requestCode != REQUEST) {
            return false;
        }
        if (!pending) {
            return true;
        }
        if (resultCode != Activity.RESULT_OK || data == null) {
            deliver("canceled", "");
            return true;
        }
        List<Uri> uris = readUris(data);
        if (uris.isEmpty()) {
            deliver("canceled", "");
            return true;
        }
        for (Uri uri : uris) {
            keep(activity, uri);
        }
        deliver("", join(uris));
        return true;
    }

    /** hostGone cancels a picker whose activity is finishing. */
    public static void hostGone(Activity activity) {
        if (!pending || activity == null || pickerHost != activity || activity.isChangingConfigurations()) {
            return;
        }
        deliver("canceled", "");
    }

    @SuppressWarnings("deprecation")
    private static void launch(
            String title,
            String directory,
            String name,
            String extensions,
            boolean multiple,
            boolean folder,
            boolean save) {
        if (pending) {
            return;
        }
        Activity activity = Host.foreground;
        if (activity == null || activity.isFinishing()) {
            deliver("no activity", "");
            return;
        }
        write = folder || save;
        pending = true;
        pickerHost = activity;
        try {
            activity.startActivityForResult(
                    pickerIntent(title, directory, name, extensions, multiple, folder, save), REQUEST);
        } catch (ActivityNotFoundException | RuntimeException ex) {
            deliver("no picker", "");
        }
    }

    private static Intent pickerIntent(
            String title,
            String directory,
            String name,
            String extensions,
            boolean multiple,
            boolean folder,
            boolean save) {
        Intent intent;
        if (folder) {
            intent = new Intent(Intent.ACTION_OPEN_DOCUMENT_TREE);
        } else if (save) {
            intent = new Intent(Intent.ACTION_CREATE_DOCUMENT);
            intent.addCategory(Intent.CATEGORY_OPENABLE);
            applyType(intent, extensions);
            if (name != null && !name.isEmpty()) {
                intent.putExtra(Intent.EXTRA_TITLE, name);
            }
        } else {
            intent = new Intent(Intent.ACTION_OPEN_DOCUMENT);
            intent.addCategory(Intent.CATEGORY_OPENABLE);
            applyType(intent, extensions);
            if (multiple) {
                intent.putExtra(Intent.EXTRA_ALLOW_MULTIPLE, true);
            }
        }
        if (title != null && !title.isEmpty()) {
            intent.putExtra(DocumentsContract.EXTRA_PROMPT, title);
        }
        if (directory != null && directory.startsWith("content:")) {
            intent.putExtra(DocumentsContract.EXTRA_INITIAL_URI, Uri.parse(directory));
        }
        int flags = Intent.FLAG_GRANT_READ_URI_PERMISSION | Intent.FLAG_GRANT_PERSISTABLE_URI_PERMISSION;
        if (write) {
            flags |= Intent.FLAG_GRANT_WRITE_URI_PERMISSION;
        }
        if (folder) {
            flags |= Intent.FLAG_GRANT_PREFIX_URI_PERMISSION;
        }
        intent.addFlags(flags);
        return intent;
    }

    private static void applyType(Intent intent, String extensions) {
        String[] mimes = mimes(extensions);
        if (mimes.length == 1) {
            intent.setType(mimes[0]);
            return;
        }
        intent.setType("*/*");
        if (mimes.length > 1) {
            intent.putExtra(Intent.EXTRA_MIME_TYPES, mimes);
        }
    }

    private static String[] mimes(String extensions) {
        if (extensions == null || extensions.trim().isEmpty()) {
            return new String[0];
        }
        MimeTypeMap map = MimeTypeMap.getSingleton();
        ArrayList<String> out = new ArrayList<>();
        for (String part : extensions.split(",")) {
            String ext = part.trim().toLowerCase(Locale.ROOT);
            if (ext.isEmpty()) {
                continue;
            }
            String mime = map.getMimeTypeFromExtension(ext);
            if (mime == null || out.contains(mime)) {
                continue;
            }
            out.add(mime);
        }
        return out.toArray(new String[0]);
    }

    private static List<Uri> readUris(Intent data) {
        ArrayList<Uri> out = new ArrayList<>();
        ClipData clip = data.getClipData();
        if (clip != null) {
            for (int i = 0; i < clip.getItemCount(); i++) {
                Uri uri = clip.getItemAt(i).getUri();
                if (uri != null) {
                    out.add(uri);
                }
            }
        }
        if (out.isEmpty() && data.getData() != null) {
            out.add(data.getData());
        }
        return out;
    }

    private static void keep(Activity activity, Uri uri) {
        if (activity == null) {
            return;
        }
        int mode = Intent.FLAG_GRANT_READ_URI_PERMISSION;
        if (write) {
            mode |= Intent.FLAG_GRANT_WRITE_URI_PERMISSION;
        }
        try {
            activity.getContentResolver().takePersistableUriPermission(uri, mode);
            return;
        } catch (SecurityException err) {
            if (!write) {
                return;
            }
        }
        try {
            activity.getContentResolver().takePersistableUriPermission(uri, Intent.FLAG_GRANT_READ_URI_PERMISSION);
        } catch (SecurityException err) {
            // Some providers refuse a durable grant. The URI still works for this session.
        }
    }

    private static String join(List<Uri> uris) {
        StringBuilder b = new StringBuilder();
        for (int i = 0; i < uris.size(); i++) {
            if (i > 0) {
                b.append('\n');
            }
            b.append(uris.get(i).toString());
        }
        return b.toString();
    }

    private static void deliver(String error, String paths) {
        pending = false;
        pickerHost = null;
        BiConsumer<String, String> cb = listener;
        if (cb == null) {
            return;
        }
        cb.accept(error, paths);
    }
}
