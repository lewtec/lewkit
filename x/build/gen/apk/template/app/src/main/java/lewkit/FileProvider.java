package lewkit;

import android.content.ContentProvider;
import android.content.ContentValues;
import android.content.Context;
import android.content.pm.PackageManager;
import android.content.pm.ProviderInfo;
import android.database.Cursor;
import android.database.MatrixCursor;
import android.net.Uri;
import android.os.Environment;
import android.os.ParcelFileDescriptor;
import android.provider.OpenableColumns;
import android.webkit.MimeTypeMap;

import org.xmlpull.v1.XmlPullParser;
import org.xmlpull.v1.XmlPullParserException;

import java.io.File;
import java.io.FileNotFoundException;
import java.io.IOException;
import java.util.ArrayList;
import java.util.List;
import java.util.Locale;

/** Shares cache and files paths as content URIs for the system share sheet. */
public final class FileProvider extends ContentProvider {
    private static final String META = "android.support.FILE_PROVIDER_PATHS";

    public static Uri getUriForFile(Context context, String authority, File file) {
        try {
            Root root = match(loadRoots(context, authority), file);
            String rootPath = root.dir.getCanonicalPath();
            String filePath = file.getCanonicalPath();
            String rel = filePath.substring(rootPath.length());
            if (rel.startsWith("/")) {
                rel = rel.substring(1);
            }
            Uri.Builder b = new Uri.Builder().scheme("content").authority(authority).appendPath(root.name);
            if (!rel.isEmpty()) {
                b.appendPath(rel);
            }
            return b.build();
        } catch (IOException e) {
            throw new IllegalArgumentException(file.getPath(), e);
        }
    }

    @Override
    public boolean onCreate() {
        return true;
    }

    @Override
    public ParcelFileDescriptor openFile(Uri uri, String mode) throws FileNotFoundException {
        File file = resolve(uri);
        int bits = ParcelFileDescriptor.MODE_READ_ONLY;
        if (mode != null && mode.indexOf('w') >= 0) {
            bits = ParcelFileDescriptor.MODE_READ_WRITE | ParcelFileDescriptor.MODE_CREATE;
        }
        return ParcelFileDescriptor.open(file, bits);
    }

    @Override
    public Cursor query(Uri uri, String[] projection, String selection, String[] selectionArgs, String sortOrder) {
        File file;
        try {
            file = resolve(uri);
        } catch (FileNotFoundException e) {
            return null;
        }
        String[] cols = projection;
        if (cols == null) {
            cols = new String[] {OpenableColumns.DISPLAY_NAME, OpenableColumns.SIZE};
        }
        MatrixCursor cursor = new MatrixCursor(cols);
        Object[] row = new Object[cols.length];
        for (int i = 0; i < cols.length; i++) {
            if (OpenableColumns.DISPLAY_NAME.equals(cols[i])) {
                row[i] = file.getName();
            } else if (OpenableColumns.SIZE.equals(cols[i])) {
                row[i] = file.length();
            }
        }
        cursor.addRow(row);
        return cursor;
    }

    @Override
    public String getType(Uri uri) {
        File file;
        try {
            file = resolve(uri);
        } catch (FileNotFoundException e) {
            return null;
        }
        String name = file.getName();
        int dot = name.lastIndexOf('.');
        if (dot >= 0) {
            String mime = MimeTypeMap.getSingleton().getMimeTypeFromExtension(
                    name.substring(dot + 1).toLowerCase(Locale.ROOT));
            if (mime != null) {
                return mime;
            }
        }
        return "application/octet-stream";
    }

    @Override
    public Uri insert(Uri uri, ContentValues values) {
        throw new UnsupportedOperationException();
    }

    @Override
    public int delete(Uri uri, String selection, String[] selectionArgs) {
        throw new UnsupportedOperationException();
    }

    @Override
    public int update(Uri uri, ContentValues values, String selection, String[] selectionArgs) {
        throw new UnsupportedOperationException();
    }

    private File resolve(Uri uri) throws FileNotFoundException {
        List<String> segs = uri.getPathSegments();
        if (segs.isEmpty()) {
            throw new FileNotFoundException("missing path");
        }
        Context context = getContext();
        if (context == null) {
            throw new FileNotFoundException("no context");
        }
        Root root = null;
        try {
            for (Root item : loadRoots(context, uri.getAuthority())) {
                if (item.name.equals(segs.get(0))) {
                    root = item;
                    break;
                }
            }
        } catch (IllegalArgumentException e) {
            throw new FileNotFoundException(e.getMessage());
        }
        if (root == null) {
            throw new FileNotFoundException(segs.get(0));
        }
        StringBuilder rel = new StringBuilder();
        for (int i = 1; i < segs.size(); i++) {
            if (rel.length() > 0) {
                rel.append('/');
            }
            rel.append(segs.get(i));
        }
        File file = rel.length() == 0 ? root.dir : new File(root.dir, rel.toString());
        if (!under(root.dir, file)) {
            throw new FileNotFoundException(uri.toString());
        }
        return file;
    }

    private static Root match(List<Root> roots, File file) throws IOException {
        Root best = null;
        int bestLen = -1;
        for (Root root : roots) {
            if (root.dir == null || !under(root.dir, file)) {
                continue;
            }
            int len = root.dir.getCanonicalPath().length();
            if (len > bestLen) {
                best = root;
                bestLen = len;
            }
        }
        if (best == null) {
            throw new IllegalArgumentException("file is outside shared paths");
        }
        return best;
    }

    private static boolean under(File root, File file) {
        try {
            String rootPath = root.getCanonicalPath();
            String filePath = file.getCanonicalPath();
            if (filePath.equals(rootPath)) {
                return true;
            }
            String prefix = rootPath.endsWith("/") ? rootPath : rootPath + "/";
            return filePath.startsWith(prefix);
        } catch (IOException e) {
            return false;
        }
    }

    private static List<Root> loadRoots(Context context, String authority) {
        ProviderInfo info = context.getPackageManager().resolveContentProvider(authority, PackageManager.GET_META_DATA);
        if (info == null) {
            throw new IllegalArgumentException("missing provider " + authority);
        }
        android.content.res.XmlResourceParser parser = info.loadXmlMetaData(context.getPackageManager(), META);
        if (parser == null) {
            throw new IllegalArgumentException("missing file paths");
        }
        try {
            List<Root> roots = new ArrayList<>();
            int event;
            while ((event = parser.next()) != XmlPullParser.END_DOCUMENT) {
                if (event != XmlPullParser.START_TAG) {
                    continue;
                }
                File base = baseDir(context, parser.getName());
                if (base == null) {
                    continue;
                }
                String name = parser.getAttributeValue(null, "name");
                String path = parser.getAttributeValue(null, "path");
                if (name == null || name.isEmpty()) {
                    continue;
                }
                File dir = base;
                if (path != null && !path.isEmpty() && !".".equals(path)) {
                    dir = new File(base, path);
                }
                roots.add(new Root(name, dir));
            }
            return roots;
        } catch (XmlPullParserException | IOException e) {
            throw new IllegalArgumentException("file paths", e);
        } finally {
            parser.close();
        }
    }

    private static File baseDir(Context context, String tag) {
        switch (tag) {
            case "files-path":
                return context.getFilesDir();
            case "cache-path":
                return context.getCacheDir();
            case "external-files-path":
                return context.getExternalFilesDir(null);
            case "external-cache-path":
                return context.getExternalCacheDir();
            case "external-path":
                return Environment.getExternalStorageDirectory();
            default:
                return null;
        }
    }

    private static final class Root {
        final String name;
        final File dir;

        Root(String name, File dir) {
            this.name = name;
            this.dir = dir;
        }
    }
}
