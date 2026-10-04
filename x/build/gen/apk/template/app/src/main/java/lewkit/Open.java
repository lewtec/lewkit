package lewkit;

import android.content.ActivityNotFoundException;
import android.content.Context;
import android.content.Intent;
import android.net.Uri;
import android.webkit.MimeTypeMap;

import java.io.File;
import java.util.Locale;

/**
 * Opens a URL or a file in another app.
 * A path has to sit under the file provider roots. Go calls open.
 */
public final class Open {
    public static final int OPENED = 0;
    public static final int BAD = 1;
    public static final int MISSING = 2;
    public static final int OUTSIDE = 3;

    private Open() {}

    public static int open(String target) {
        Context app = Host.app;
        if (app == null) {
            return MISSING;
        }
        String text = target == null ? "" : target.trim();
        if (text.isEmpty()) {
            return BAD;
        }
        try {
            Intent intent = view(app, text);
            if (intent == null) {
                return text.startsWith("/") ? MISSING : BAD;
            }
            intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
            app.startActivity(intent);
            return OPENED;
        } catch (ActivityNotFoundException e) {
            return MISSING;
        } catch (IllegalArgumentException e) {
            return OUTSIDE;
        } catch (RuntimeException e) {
            return MISSING;
        }
    }

    private static Intent view(Context app, String text) {
        if (text.startsWith("/")) {
            return file(app, new File(text));
        }
        Uri uri = Uri.parse(text);
        String scheme = uri.getScheme();
        if (scheme == null || scheme.isEmpty()) {
            return null;
        }
        if ("file".equalsIgnoreCase(scheme)) {
            String path = uri.getPath();
            if (path == null || path.isEmpty()) {
                return null;
            }
            return file(app, new File(path));
        }
        Intent intent = new Intent(Intent.ACTION_VIEW);
        if ("content".equalsIgnoreCase(scheme)) {
            String mime = app.getContentResolver().getType(uri);
            if (mime == null || mime.isEmpty()) {
                mime = "*/*";
            }
            intent.setDataAndType(uri, mime);
            intent.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
            return intent;
        }
        intent.setData(uri);
        return intent;
    }

    private static Intent file(Context app, File file) {
        if (file == null || !file.isFile()) {
            return null;
        }
        String authority = app.getPackageName() + ".fileprovider";
        Uri uri = FileProvider.getUriForFile(app, authority, file);
        Intent intent = new Intent(Intent.ACTION_VIEW);
        intent.setDataAndType(uri, mime(file));
        intent.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
        return intent;
    }

    private static String mime(File file) {
        String name = file.getName();
        int dot = name.lastIndexOf('.');
        if (dot >= 0) {
            String found = MimeTypeMap.getSingleton().getMimeTypeFromExtension(
                    name.substring(dot + 1).toLowerCase(Locale.ROOT));
            if (found != null) {
                return found;
            }
        }
        return "application/octet-stream";
    }
}
