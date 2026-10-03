package lewkit;

import android.content.ContentResolver;
import android.content.Context;
import android.database.Cursor;
import android.graphics.Bitmap;
import android.graphics.BitmapFactory;
import android.graphics.Point;
import android.net.Uri;
import android.os.ParcelFileDescriptor;
import android.provider.DocumentsContract;
import android.util.Log;

import java.io.ByteArrayOutputStream;
import java.io.FileNotFoundException;
import java.io.IOException;
import java.io.OutputStream;
import java.util.ArrayList;
import java.util.List;

/**
 * Reads documents the picker granted.
 * info and list return a catalog. readFd detaches a read-only descriptor.
 * thumbFd detaches a JPEG preview, not the original file.
 * A catalog line is name, size, dir, uri separated by tabs.
 * name and uri escape backslash, newline, carriage return, and tab.
 */
public final class Documents {
    private static final String[] COLUMNS = {
        DocumentsContract.Document.COLUMN_DISPLAY_NAME,
        DocumentsContract.Document.COLUMN_MIME_TYPE,
        DocumentsContract.Document.COLUMN_SIZE,
        DocumentsContract.Document.COLUMN_DOCUMENT_ID,
    };

    private Documents() {}

    public static String info(String uri) throws IOException {
        Resolved resolved = resolve(uri);
        Row row = queryOne(resolved);
        return record(row);
    }

    public static String list(String uri) throws IOException {
        Resolved resolved = resolve(uri);
        if (resolved.tree == null) {
            throw new IOException("not a tree");
        }
        Uri children = DocumentsContract.buildChildDocumentsUriUsingTree(resolved.tree, resolved.docId);
        return catalog(query(resolved.resolver, children, resolved.tree, false));
    }

    public static int readFd(String uri) throws IOException {
        if (uri == null || uri.isEmpty()) {
            throw new IOException("no uri");
        }
        ContentResolver resolver = resolver();
        ParcelFileDescriptor pfd;
        try {
            pfd = resolver.openFileDescriptor(Uri.parse(uri), "r");
        } catch (RuntimeException ex) {
            throw io(ex);
        }
        if (pfd == null) {
            throw new IOException("no file");
        }
        int fd = pfd.detachFd();
        pfd.close();
        return fd;
    }

    public static int thumbFd(String uri) throws IOException {
        if (uri == null || uri.isEmpty()) {
            throw new IOException("no uri");
        }
        ContentResolver resolver = resolver();
        Bitmap bitmap = loadThumb(resolver, Uri.parse(uri));
        if (bitmap == null) {
            throw new IOException("no thumbnail");
        }
        bitmap = fit(bitmap, THUMB);
        try {
            return pipeJpeg(bitmap);
        } finally {
            bitmap.recycle();
        }
    }

    private static Resolved resolve(String raw) throws IOException {
        if (raw == null || raw.isEmpty()) {
            throw new IOException("no uri");
        }
        Uri uri = Uri.parse(raw);
        try {
            return resolveUri(uri);
        } catch (RuntimeException ex) {
            throw io(ex);
        }
    }

    private static Resolved resolveUri(Uri uri) throws IOException {
        boolean tree = DocumentsContract.isTreeUri(uri);
        boolean document = isDocument(uri);
        if (!tree && !document) {
            throw new IOException("not a document");
        }
        Resolved resolved = new Resolved();
        resolved.resolver = resolver();
        resolved.tree = tree ? uri : null;
        if (document) {
            resolved.docId = DocumentsContract.getDocumentId(uri);
        } else {
            resolved.docId = DocumentsContract.getTreeDocumentId(uri);
        }
        if (tree) {
            resolved.doc = DocumentsContract.buildDocumentUriUsingTree(uri, resolved.docId);
        } else {
            resolved.doc = uri;
        }
        return resolved;
    }

    // isTreeUri is true for a nested document too. The nested id is the
    // document id, not the tree id from getTreeDocumentId.
    private static boolean isDocument(Uri uri) {
        List<String> paths = uri.getPathSegments();
        if (paths.size() == 2 && "document".equals(paths.get(0))) {
            return true;
        }
        return paths.size() >= 4 && "tree".equals(paths.get(0)) && "document".equals(paths.get(2));
    }

    private static Row queryOne(Resolved resolved) throws IOException {
        List<Row> rows = query(resolved.resolver, resolved.doc, resolved.tree, true);
        if (rows.isEmpty()) {
            throw new IOException("no document");
        }
        return rows.get(0);
    }

    private static List<Row> query(ContentResolver resolver, Uri uri, Uri tree, boolean single) throws IOException {
        Cursor cursor;
        try {
            cursor = resolver.query(uri, COLUMNS, null, null, null);
        } catch (RuntimeException ex) {
            throw io(ex);
        }
        if (cursor == null) {
            throw new IOException("no cursor");
        }
        try {
            int name = cursor.getColumnIndex(DocumentsContract.Document.COLUMN_DISPLAY_NAME);
            int mime = cursor.getColumnIndex(DocumentsContract.Document.COLUMN_MIME_TYPE);
            int size = cursor.getColumnIndex(DocumentsContract.Document.COLUMN_SIZE);
            int id = cursor.getColumnIndex(DocumentsContract.Document.COLUMN_DOCUMENT_ID);
            List<Row> out = new ArrayList<>();
            while (cursor.moveToNext()) {
                String docId = id >= 0 && !cursor.isNull(id) ? cursor.getString(id) : "";
                Row row = new Row();
                row.name = name >= 0 && !cursor.isNull(name) ? cursor.getString(name) : "";
                if (row.name.isEmpty()) {
                    row.name = docId;
                }
                String mimeType = mime >= 0 && !cursor.isNull(mime) ? cursor.getString(mime) : "";
                row.dir = DocumentsContract.Document.MIME_TYPE_DIR.equals(mimeType);
                row.size = size >= 0 && !cursor.isNull(size) ? cursor.getLong(size) : 0;
                if (tree != null && !docId.isEmpty()) {
                    row.uri = DocumentsContract.buildDocumentUriUsingTree(tree, docId).toString();
                } else if (single) {
                    row.uri = uri.toString();
                } else {
                    continue;
                }
                out.add(row);
            }
            return out;
        } finally {
            cursor.close();
        }
    }

    private static final int THUMB = 384;

    private static Bitmap loadThumb(ContentResolver resolver, Uri uri) throws IOException {
        Bitmap bitmap = null;
        FileNotFoundException missed = null;
        try {
            bitmap = DocumentsContract.getDocumentThumbnail(resolver, uri, new Point(THUMB, THUMB), null);
        } catch (FileNotFoundException ex) {
            missed = ex;
        } catch (RuntimeException ex) {
            throw io(ex);
        }
        if (bitmap != null) {
            return bitmap;
        }
        bitmap = sample(resolver, uri, THUMB);
        if (bitmap == null && missed != null) {
            throw new IOException(missed.getMessage(), missed);
        }
        return bitmap;
    }

    private static Bitmap sample(ContentResolver resolver, Uri uri, int max) throws IOException {
        BitmapFactory.Options bounds = new BitmapFactory.Options();
        bounds.inJustDecodeBounds = true;
        if (!decode(resolver, uri, bounds)) {
            return null;
        }
        if (bounds.outWidth <= 0 || bounds.outHeight <= 0) {
            return null;
        }
        int sampleSize = 1;
        while (bounds.outWidth / sampleSize > max || bounds.outHeight / sampleSize > max) {
            int next = sampleSize * 2;
            if (next <= sampleSize) {
                break;
            }
            sampleSize = next;
        }
        BitmapFactory.Options opts = new BitmapFactory.Options();
        opts.inSampleSize = sampleSize;
        return decodeBitmap(resolver, uri, opts);
    }

    private static boolean decode(ContentResolver resolver, Uri uri, BitmapFactory.Options opts) throws IOException {
        ParcelFileDescriptor pfd = openRead(resolver, uri);
        if (pfd == null) {
            return false;
        }
        try {
            BitmapFactory.decodeFileDescriptor(pfd.getFileDescriptor(), null, opts);
            return true;
        } finally {
            pfd.close();
        }
    }

    private static Bitmap decodeBitmap(ContentResolver resolver, Uri uri, BitmapFactory.Options opts) throws IOException {
        ParcelFileDescriptor pfd = openRead(resolver, uri);
        if (pfd == null) {
            return null;
        }
        try {
            return BitmapFactory.decodeFileDescriptor(pfd.getFileDescriptor(), null, opts);
        } finally {
            pfd.close();
        }
    }

    private static ParcelFileDescriptor openRead(ContentResolver resolver, Uri uri) throws IOException {
        try {
            return resolver.openFileDescriptor(uri, "r");
        } catch (RuntimeException ex) {
            throw io(ex);
        }
    }

    private static Bitmap fit(Bitmap bitmap, int max) {
        int w = bitmap.getWidth();
        int h = bitmap.getHeight();
        if (w <= max && h <= max) {
            return bitmap;
        }
        float scale = Math.min(max / (float) w, max / (float) h);
        int nw = Math.max(1, Math.round(w * scale));
        int nh = Math.max(1, Math.round(h * scale));
        Bitmap fitted = Bitmap.createScaledBitmap(bitmap, nw, nh, true);
        if (fitted != bitmap) {
            bitmap.recycle();
        }
        return fitted;
    }

    private static int pipeJpeg(Bitmap bitmap) throws IOException {
        ByteArrayOutputStream bytes = new ByteArrayOutputStream();
        if (!bitmap.compress(Bitmap.CompressFormat.JPEG, 80, bytes)) {
            throw new IOException("thumbnail");
        }
        byte[] data = bytes.toByteArray();
        ParcelFileDescriptor[] pipe;
        try {
            pipe = ParcelFileDescriptor.createPipe();
        } catch (RuntimeException ex) {
            throw io(ex);
        }
        int fd = pipe[0].detachFd();
        final ParcelFileDescriptor write = pipe[1];
        Thread writer = new Thread(new Runnable() {
            @Override
            public void run() {
                writeJpeg(write, data);
            }
        });
        writer.setDaemon(true);
        writer.start();
        return fd;
    }

    private static void writeJpeg(ParcelFileDescriptor write, byte[] data) {
        try (OutputStream out = new ParcelFileDescriptor.AutoCloseOutputStream(write)) {
            out.write(data);
        } catch (IOException ex) {
            String message = ex.getMessage();
            if (message == null || message.isEmpty()) {
                message = "thumbnail";
            }
            Log.w("lewkit", message);
        }
    }

    private static String catalog(List<Row> rows) {
        StringBuilder b = new StringBuilder();
        for (int i = 0; i < rows.size(); i++) {
            if (i > 0) {
                b.append('\n');
            }
            b.append(record(rows.get(i)));
        }
        return b.toString();
    }

    private static String record(Row row) {
        return field(row.name) + "\t" + row.size + "\t" + (row.dir ? "1" : "0") + "\t" + field(row.uri);
    }

    private static String field(String s) {
        if (s == null) {
            return "";
        }
        return s.replace("\\", "\\\\").replace("\n", "\\n").replace("\r", "\\r").replace("\t", "\\t");
    }

    private static ContentResolver resolver() throws IOException {
        Context ctx = Host.app;
        if (ctx == null) {
            throw new IOException("no context");
        }
        return ctx.getContentResolver();
    }

    private static IOException io(RuntimeException ex) {
        String message = ex.getMessage();
        if (message == null || message.isEmpty()) {
            message = ex.getClass().getSimpleName();
        }
        return new IOException(message, ex);
    }

    private static final class Resolved {
        ContentResolver resolver;
        Uri tree;
        Uri doc;
        String docId;
    }

    private static final class Row {
        String name;
        long size;
        boolean dir;
        String uri;
    }
}
