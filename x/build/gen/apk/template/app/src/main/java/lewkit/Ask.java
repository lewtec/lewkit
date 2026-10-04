package lewkit;

import android.app.Activity;
import android.app.AlertDialog;
import android.os.Handler;
import android.os.Looper;
import android.text.InputType;
import android.widget.EditText;

import java.util.ArrayList;
import java.util.function.Consumer;

/**
 * Shows one alert, confirm, prompt, or list on the foreground activity.
 * Go sets the listener, then calls alert, confirm, prompt, or choose.
 * The listener receives one askwire message: a status line and a payload.
 */
public final class Ask {
    private static volatile Consumer<String> listener;
    private static boolean pending;

    private Ask() {}

    public static void setListener(Consumer<String> next) {
        listener = next;
    }

    public static void alert(String title, String message) {
        post(() -> {
            Activity activity = ready();
            if (activity == null) {
                return;
            }
            new AlertDialog.Builder(activity)
                    .setTitle(shown(title, "Error"))
                    .setMessage(shown(message, ""))
                    .setPositiveButton(android.R.string.ok, (d, w) -> deliver("ok\n"))
                    .setOnCancelListener(d -> deliver("canceled\n"))
                    .show();
        });
    }

    public static void confirm(String title, String message) {
        post(() -> {
            Activity activity = ready();
            if (activity == null) {
                return;
            }
            new AlertDialog.Builder(activity)
                    .setTitle(shown(title, "Confirm"))
                    .setMessage(shown(message, title))
                    .setPositiveButton(android.R.string.yes, (d, w) -> deliver("yes\n"))
                    .setNegativeButton(android.R.string.no, (d, w) -> deliver("no\n"))
                    .setOnCancelListener(d -> deliver("canceled\n"))
                    .show();
        });
    }

    public static void prompt(String title, String message, String text) {
        post(() -> {
            Activity activity = ready();
            if (activity == null) {
                return;
            }
            EditText input = new EditText(activity);
            input.setInputType(InputType.TYPE_CLASS_TEXT);
            input.setText(text == null ? "" : text);
            String body = message == null ? "" : message;
            AlertDialog.Builder builder = new AlertDialog.Builder(activity)
                    .setTitle(shown(title, "Input"))
                    .setView(input)
                    .setPositiveButton(android.R.string.ok, (d, w) -> deliver("ok\n" + input.getText()))
                    .setNegativeButton(android.R.string.cancel, (d, w) -> deliver("canceled\n"))
                    .setOnCancelListener(d -> deliver("canceled\n"));
            if (!body.isEmpty()) {
                builder.setMessage(body);
            }
            builder.show();
        });
    }

    public static void choose(String title, String items) {
        post(() -> {
            Activity activity = ready();
            if (activity == null) {
                return;
            }
            String[] lines = rows(items);
            new AlertDialog.Builder(activity)
                    .setTitle(shown(title, "Choose"))
                    .setItems(lines, (d, which) -> deliver("ok\n" + lines[which]))
                    .setNegativeButton(android.R.string.cancel, (d, w) -> deliver("canceled\n"))
                    .setOnCancelListener(d -> deliver("canceled\n"))
                    .show();
        });
    }

    public static void cancel() {
        post(() -> deliver("canceled\n"));
    }

    private static void post(Runnable run) {
        if (Looper.myLooper() == Looper.getMainLooper()) {
            run.run();
            return;
        }
        new Handler(Looper.getMainLooper()).post(run);
    }

    private static Activity ready() {
        if (pending) {
            accept("busy\n");
            return null;
        }
        Activity activity = activity();
        if (activity == null) {
            accept("no activity\n");
            return null;
        }
        pending = true;
        return activity;
    }

    private static Activity activity() {
        Activity activity = Host.foreground;
        if (activity == null || activity.isFinishing()) {
            return null;
        }
        return activity;
    }

    private static void deliver(String value) {
        if (!pending) {
            return;
        }
        pending = false;
        accept(value);
    }

    private static void accept(String value) {
        Consumer<String> cb = listener;
        if (cb != null) {
            cb.accept(value == null ? "canceled\n" : value);
        }
    }

    private static String shown(String value, String fallback) {
        if (value == null || value.isEmpty()) {
            return fallback;
        }
        return value;
    }

    private static String[] rows(String items) {
        if (items == null || items.isEmpty()) {
            return new String[0];
        }
        String[] parts = items.split("\n", -1);
        ArrayList<String> rows = new ArrayList<>();
        for (String part : parts) {
            if (!part.isEmpty()) {
                rows.add(part);
            }
        }
        return rows.toArray(new String[0]);
    }
}
