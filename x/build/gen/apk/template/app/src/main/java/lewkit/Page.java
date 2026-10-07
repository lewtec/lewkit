package lewkit;

import android.os.Handler;
import android.os.Looper;
import android.webkit.CookieManager;
import android.webkit.JavascriptInterface;
import android.webkit.WebResourceRequest;
import android.webkit.WebResourceResponse;

import java.io.ByteArrayInputStream;
import java.nio.charset.StandardCharsets;
import java.util.ArrayList;
import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.List;
import java.util.Locale;
import java.util.Map;

// Page is the activity WebView for one Go window.
// The page bytes stay in this process. This class does not listen and does
// not start the packaged service.
public final class Page {
    public static final String ORIGIN = "https://appassets.androidplatform.net";
    static final String PAGE_HEADER = "X-Lewkit-Page";

    public interface Opener {
        void call(int id);
    }

    public interface Bridge {
        byte[] fetch(String method, String url, String headers, String body);

        void message(String text);

        void closed();

        void evalResult(String token, String value);
    }

    public interface Session {
        void load(String url);

        void show(String baseUrl, String data, String mime, String encoding, String historyUrl);

        void run(String script, String token);

        void dark(boolean on);

        void finish();
    }

    public static final class Js {
        private final int id;

        public Js(int id) {
            this.id = id;
        }

        @JavascriptInterface
        public void postMessage(String text) {
            message(id, text);
        }

        @JavascriptInterface
        public void submit(String method, String url, String contentType, String body) {
            Page.submit(id, method, url, contentType, body);
        }
    }

    public static volatile Opener onOpen;

    private static final Object lock = new Object();
    private static final Map<Integer, Bridge> bridges = new HashMap<>();
    private static final Map<Integer, Session> sessions = new HashMap<>();
    private static final Map<Integer, Boolean> dark = new HashMap<>();

    private Page() {}

    // open registers bridge and asks the activity to show it.
    // An empty ready URL dismisses the splash without publishing an address.
    public static boolean open(int id, Bridge bridge) {
        if (id == 0 || bridge == null || onOpen == null) {
            return false;
        }
        synchronized (lock) {
            bridges.put(id, bridge);
        }
        main(() -> {
            Opener opener = onOpen;
            if (opener != null) {
                opener.call(id);
            }
        });
        Host.ready("");
        return true;
    }

    public static void attach(int id, Session session) {
        if (id == 0 || session == null) {
            return;
        }
        Boolean theme;
        synchronized (lock) {
            if (!bridges.containsKey(id)) {
                session.finish();
                return;
            }
            sessions.put(id, session);
            theme = dark.get(id);
        }
        session.load(ORIGIN + "/");
        if (theme != null) {
            session.dark(theme);
        }
    }

    public static void detach(int id) {
        if (id == 0) {
            return;
        }
        synchronized (lock) {
            sessions.remove(id);
        }
        drop(id);
    }

    public static void close(int id) {
        main(() -> {
            Session session;
            synchronized (lock) {
                session = sessions.remove(id);
            }
            drop(id);
            if (session != null) {
                session.finish();
            }
        });
    }

    public static void evaluate(int id, String token, String script) {
        Session session;
        synchronized (lock) {
            session = sessions.get(id);
        }
        if (session == null) {
            result(id, token, "");
            return;
        }
        session.run(script, token);
    }

    public static void preferDark(int id, boolean on) {
        Session session;
        synchronized (lock) {
            dark.put(id, on);
            session = sessions.get(id);
        }
        if (session != null) {
            session.dark(on);
        }
    }

    public static WebResourceResponse intercept(int id, WebResourceRequest request) {
        if (request == null || request.getUrl() == null) {
            return null;
        }
        String url = request.getUrl().toString();
        if (!isOrigin(url)) {
            return null;
        }
        String method = request.getMethod();
        if (method == null || method.isEmpty()) {
            method = "GET";
        }
        Bridge bridge = bridge(id);
        if (bridge == null) {
            return notFound();
        }
        byte[] packed = bridge.fetch(method, url, headerText(request.getRequestHeaders()), "");
        if (packed == null) {
            return notFound();
        }
        Parsed parsed = parse(packed);
        applyCookies(url, parsed.cookies);
        return parsed.toResponse();
    }

    static void submit(int id, String method, String url, String contentType, String body) {
        Bridge bridge = bridge(id);
        Session session;
        synchronized (lock) {
            session = sessions.get(id);
        }
        if (bridge == null || session == null || url == null) {
            return;
        }
        String headers = "";
        if (contentType != null && !contentType.isEmpty()) {
            headers = "Content-Type: " + contentType + "\n";
        }
        byte[] packed = bridge.fetch(method == null ? "POST" : method, url, headers, body == null ? "" : body);
        if (packed == null) {
            return;
        }
        Parsed parsed = parse(packed);
        applyCookies(url, parsed.cookies);
        if (parsed.status >= 300 && parsed.status < 400) {
            String location = parsed.headers.get("Location");
            if (location != null && !location.isEmpty()) {
                session.load(location);
                return;
            }
        }
        String history = parsed.page != null && !parsed.page.isEmpty() ? parsed.page : url;
        String encoding = parsed.encoding == null ? "utf-8" : parsed.encoding;
        String text = new String(parsed.body, StandardCharsets.UTF_8);
        session.show(history, text, parsed.mime, encoding, history);
    }

    static void message(int id, String text) {
        Bridge bridge = bridge(id);
        if (bridge != null) {
            bridge.message(text == null ? "" : text);
        }
    }

    public static void result(int id, String token, String value) {
        Bridge bridge = bridge(id);
        if (bridge != null) {
            bridge.evalResult(token == null ? "" : token, value == null ? "" : value);
        }
    }

    private static Bridge bridge(int id) {
        synchronized (lock) {
            return bridges.get(id);
        }
    }

    private static void drop(int id) {
        Bridge bridge;
        synchronized (lock) {
            bridge = bridges.remove(id);
            dark.remove(id);
        }
        if (bridge != null) {
            bridge.closed();
        }
    }

    private static boolean isOrigin(String url) {
        return url.equals(ORIGIN) || url.startsWith(ORIGIN + "/");
    }

    private static String headerText(Map<String, String> headers) {
        if (headers == null || headers.isEmpty()) {
            return "";
        }
        StringBuilder out = new StringBuilder();
        for (Map.Entry<String, String> entry : headers.entrySet()) {
            if (entry.getKey() == null || entry.getValue() == null) {
                continue;
            }
            out.append(entry.getKey()).append(": ").append(entry.getValue()).append('\n');
        }
        return out.toString();
    }

    private static void applyCookies(String url, List<String> cookies) {
        if (cookies == null || cookies.isEmpty()) {
            return;
        }
        CookieManager manager = CookieManager.getInstance();
        for (String cookie : cookies) {
            manager.setCookie(url, cookie);
        }
    }

    private static WebResourceResponse notFound() {
        return new WebResourceResponse(
                "text/plain",
                "utf-8",
                404,
                "Not Found",
                new LinkedHashMap<>(),
                new ByteArrayInputStream(new byte[0]));
    }

    private static void main(Runnable job) {
        if (Looper.myLooper() == Looper.getMainLooper()) {
            job.run();
            return;
        }
        new Handler(Looper.getMainLooper()).post(job);
    }

    static final class Parsed {
        int status = 500;
        String mime = "application/octet-stream";
        String encoding;
        String page = "";
        Map<String, String> headers = new LinkedHashMap<>();
        List<String> cookies = new ArrayList<>();
        byte[] body = new byte[0];

        WebResourceResponse toResponse() {
            String reason = reason(status);
            String charset = encoding;
            if (charset == null && (mime.startsWith("text/") || mime.contains("json") || mime.contains("javascript") || mime.contains("xml"))) {
                charset = "utf-8";
            }
            return new WebResourceResponse(mime, charset, status, reason, headers, new ByteArrayInputStream(body));
        }
    }

    static Parsed parse(byte[] packed) {
        Parsed parsed = new Parsed();
        if (packed == null) {
            return parsed;
        }
        int split = -1;
        for (int i = 0; i + 1 < packed.length; i++) {
            if (packed[i] == '\n' && packed[i + 1] == '\n') {
                split = i;
                break;
            }
        }
        if (split < 0) {
            parsed.body = packed;
            return parsed;
        }
        String head = new String(packed, 0, split, StandardCharsets.UTF_8);
        parsed.body = new byte[packed.length - split - 2];
        System.arraycopy(packed, split + 2, parsed.body, 0, parsed.body.length);
        String[] lines = head.split("\n", -1);
        if (lines.length == 0) {
            return parsed;
        }
        try {
            parsed.status = Integer.parseInt(lines[0].trim());
        } catch (NumberFormatException ignored) {
            parsed.status = 500;
        }
        if (parsed.status < 100) {
            parsed.status = 500;
        }
        for (int i = 1; i < lines.length; i++) {
            String line = lines[i];
            int colon = line.indexOf(':');
            if (colon <= 0) {
                continue;
            }
            String name = line.substring(0, colon).trim();
            String value = line.substring(colon + 1).trim();
            if (name.equalsIgnoreCase("Content-Type")) {
                parsed.mime = mimeOf(value);
                parsed.encoding = charsetOf(value);
                continue;
            }
            if (name.equalsIgnoreCase("Set-Cookie")) {
                parsed.cookies.add(value);
                continue;
            }
            if (name.equalsIgnoreCase(PAGE_HEADER)) {
                parsed.page = value;
                continue;
            }
            parsed.headers.put(name, value);
        }
        return parsed;
    }

    private static String mimeOf(String value) {
        int semi = value.indexOf(';');
        String mime = semi >= 0 ? value.substring(0, semi) : value;
        mime = mime.trim().toLowerCase(Locale.ROOT);
        if (mime.isEmpty()) {
            return "application/octet-stream";
        }
        return mime;
    }

    private static String charsetOf(String value) {
        String lower = value.toLowerCase(Locale.ROOT);
        int at = lower.indexOf("charset=");
        if (at < 0) {
            return null;
        }
        String charset = value.substring(at + "charset=".length()).trim();
        int semi = charset.indexOf(';');
        if (semi >= 0) {
            charset = charset.substring(0, semi).trim();
        }
        if (charset.startsWith("\"") && charset.endsWith("\"") && charset.length() >= 2) {
            charset = charset.substring(1, charset.length() - 1);
        }
        return charset.isEmpty() ? null : charset;
    }

    private static String reason(int status) {
        switch (status) {
            case 200:
                return "OK";
            case 201:
                return "Created";
            case 204:
                return "No Content";
            case 301:
                return "Moved Permanently";
            case 302:
                return "Found";
            case 303:
                return "See Other";
            case 304:
                return "Not Modified";
            case 307:
                return "Temporary Redirect";
            case 308:
                return "Permanent Redirect";
            case 400:
                return "Bad Request";
            case 401:
                return "Unauthorized";
            case 403:
                return "Forbidden";
            case 404:
                return "Not Found";
            case 405:
                return "Method Not Allowed";
            case 500:
                return "Internal Server Error";
            case 502:
                return "Bad Gateway";
            default:
                return "Status";
        }
    }
}
