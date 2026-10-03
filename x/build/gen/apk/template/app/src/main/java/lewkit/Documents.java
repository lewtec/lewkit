package lewkit;

import android.content.ContentResolver;
import android.content.Context;
import android.database.Cursor;
import android.net.Uri;
import android.os.ParcelFileDescriptor;
import android.provider.DocumentsContract;

import java.io.IOException;
import java.util.ArrayList;
import java.util.List;

/**
 * Reads documents the picker granted.
 * info and list return a catalog. readFd detaches a read-only descriptor.
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
