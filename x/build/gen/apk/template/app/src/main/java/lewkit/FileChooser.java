package lewkit;

import android.app.Activity;
import android.content.ActivityNotFoundException;
import android.content.ClipData;
import android.content.Context;
import android.content.Intent;
import android.net.Uri;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.provider.DocumentsContract;
import android.webkit.MimeTypeMap;

import java.util.ArrayList;
import java.util.List;
import java.util.Locale;
import java.util.function.BiConsumer;

/**
 * Shows the system document picker once.
 * Go sets the listener, then open. The listener receives content URIs, or
 * "canceled", "no activity", or "no picker".
 */
public final class FileChooser extends Activity {
    private static final int REQUEST = 1;

    private static final String EXTRA_TITLE = "title";
    private static final String EXTRA_DIRECTORY = "directory";
    private static final String EXTRA_NAME = "name";
    private static final String EXTRA_EXTENSIONS = "extensions";
    private static final String EXTRA_MULTIPLE = "multiple";
    private static final String EXTRA_FOLDER = "folder";
    private static final String EXTRA_SAVE = "save";

    private static volatile BiConsumer<String, String> listener;
    private static volatile FileChooser current;

    private BiConsumer<String, String> mine;
    private boolean delivered;
    private boolean write;

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
            FileChooser open = current;
            if (open != null) {
                open.finish();
                return;
            }
            report("canceled", "");
        });
    }

    private static void launch(
            String title,
            String directory,
            String name,
            String extensions,
            boolean multiple,
            boolean folder,
            boolean save) {
        if (started(Host.foreground, title, directory, name, extensions, multiple, folder, save, false)) {
            return;
        }
        if (started(Host.app, title, directory, name, extensions, multiple, folder, save, true)) {
            return;
        }
        report("no activity", "");
    }

    private static boolean started(
            Context ctx,
            String title,
            String directory,
            String name,
            String extensions,
            boolean multiple,
            boolean folder,
            boolean save,
            boolean newTask) {
        if (ctx == null) {
            return false;
        }
        Intent intent = chooserIntent(ctx, title, directory, name, extensions, multiple, folder, save);
        if (newTask) {
            intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
        }
        try {
            ctx.startActivity(intent);
            return true;
        } catch (RuntimeException ex) {
            // This context cannot start the chooser. The caller tries the next one.
            return false;
        }
    }

    private static Intent chooserIntent(
            Context ctx,
            String title,
            String directory,
            String name,
            String extensions,
            boolean multiple,
            boolean folder,
            boolean save) {
        Intent intent = new Intent(ctx, FileChooser.class);
        intent.putExtra(EXTRA_TITLE, title == null ? "" : title);
        intent.putExtra(EXTRA_DIRECTORY, directory == null ? "" : directory);
        intent.putExtra(EXTRA_NAME, name == null ? "" : name);
        intent.putExtra(EXTRA_EXTENSIONS, extensions == null ? "" : extensions);
        intent.putExtra(EXTRA_MULTIPLE, multiple);
        intent.putExtra(EXTRA_FOLDER, folder);
        intent.putExtra(EXTRA_SAVE, save);
        return intent;
    }

    private static void report(String error, String paths) {
        BiConsumer<String, String> cb = listener;
        if (cb == null) {
            return;
        }
        cb.accept(error, paths);
    }

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        current = this;
        mine = listener;
        if (savedInstanceState != null) {
            write = savedInstanceState.getBoolean("write");
            if (savedInstanceState.getBoolean("started")) {
                return;
            }
        }
        showPicker();
    }

    @Override
    protected void onSaveInstanceState(Bundle outState) {
        super.onSaveInstanceState(outState);
        outState.putBoolean("started", true);
        outState.putBoolean("write", write);
    }

    @SuppressWarnings("deprecation")
    private void showPicker() {
        try {
            startActivityForResult(pickerIntent(), REQUEST);
        } catch (ActivityNotFoundException err) {
            finishWith("no picker", "");
        }
    }

    private Intent pickerIntent() {
        Intent src = getIntent();
        String title = src.getStringExtra(EXTRA_TITLE);
        String directory = src.getStringExtra(EXTRA_DIRECTORY);
        String name = src.getStringExtra(EXTRA_NAME);
        String extensions = src.getStringExtra(EXTRA_EXTENSIONS);
        boolean multiple = src.getBooleanExtra(EXTRA_MULTIPLE, false);
        boolean folder = src.getBooleanExtra(EXTRA_FOLDER, false);
        boolean save = src.getBooleanExtra(EXTRA_SAVE, false);
        write = folder || save;

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

    @SuppressWarnings("deprecation")
    @Override
    protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        if (requestCode != REQUEST) {
            super.onActivityResult(requestCode, resultCode, data);
            return;
        }
        if (resultCode != RESULT_OK || data == null) {
            finishWith("canceled", "");
            return;
        }
        List<Uri> uris = readUris(data);
        if (uris.isEmpty()) {
            finishWith("canceled", "");
            return;
        }
        for (Uri uri : uris) {
            keep(uri);
        }
        finishWith("", join(uris));
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

    private void keep(Uri uri) {
        int mode = Intent.FLAG_GRANT_READ_URI_PERMISSION;
        if (write) {
            mode |= Intent.FLAG_GRANT_WRITE_URI_PERMISSION;
        }
        try {
            getContentResolver().takePersistableUriPermission(uri, mode);
            return;
        } catch (SecurityException err) {
            if (!write) {
                return;
            }
        }
        try {
            getContentResolver().takePersistableUriPermission(uri, Intent.FLAG_GRANT_READ_URI_PERMISSION);
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

    private void finishWith(String error, String paths) {
        if (delivered) {
            return;
        }
        delivered = true;
        BiConsumer<String, String> cb = mine;
        if (cb != null) {
            cb.accept(error, paths);
        }
        finish();
    }

    @Override
    protected void onDestroy() {
        if (current == this) {
            current = null;
        }
        if (!delivered && !isChangingConfigurations()) {
            delivered = true;
            BiConsumer<String, String> cb = mine;
            if (cb != null) {
                cb.accept("canceled", "");
            }
        }
        super.onDestroy();
    }
}
