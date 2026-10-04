package lewkit;

import android.Manifest;
import android.app.Activity;
import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.content.Context;
import android.content.pm.PackageManager;
import android.os.Build;

/**
 * Posts one local notification.
 * Go calls post. A missing POST_NOTIFICATIONS grant asks the foreground
 * activity, then finish polls until the user answers.
 */
public final class Notify {
    public static final int POSTED = 0;
    public static final int WAITING = 1;
    public static final int DENIED = 2;
    private static final int CODE = 41;

    private static volatile int permit;
    private static boolean waiting;
    private static int pendingID;
    private static String pendingTitle = "";
    private static String pendingMessage = "";
    private static String pendingUrgency = "";
    private static boolean pendingProgress;
    private static int pendingPercent;
    private static int next = 1;

    private Notify() {}

    public static int post(int id, String title, String message, String urgency, boolean progress, int percent) {
        Context app = Host.app;
        if (app == null) {
            return -1;
        }
        ensure(app, urgency);
        if (canPost(app)) {
            return show(app, id, title, message, urgency, progress, percent) ? POSTED : DENIED;
        }
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) {
            return DENIED;
        }
        Activity activity = Host.foreground;
        if (activity == null) {
            return -1;
        }
        pendingID = id;
        pendingTitle = title;
        pendingMessage = message;
        pendingUrgency = urgency;
        pendingProgress = progress;
        pendingPercent = percent;
        waiting = true;
        permit = WAITING;
        activity.requestPermissions(new String[]{Manifest.permission.POST_NOTIFICATIONS}, CODE);
        return WAITING;
    }

    public static int finish() {
        if (permit == DENIED) {
            return DENIED;
        }
        if (permit == POSTED && !waiting) {
            return POSTED;
        }
        Context app = Host.app;
        if (app != null && canPost(app)) {
            if (waiting) {
                waiting = false;
                if (!show(app, pendingID, pendingTitle, pendingMessage, pendingUrgency, pendingProgress, pendingPercent)) {
                    permit = DENIED;
                    return DENIED;
                }
            }
            permit = POSTED;
            return POSTED;
        }
        if (!waiting) {
            return DENIED;
        }
        return WAITING;
    }

    public static boolean onResult(int requestCode, int[] results) {
        if (requestCode != CODE) {
            return false;
        }
        boolean granted = results != null && results.length > 0 && results[0] == PackageManager.PERMISSION_GRANTED;
        if (!granted) {
            permit = DENIED;
            waiting = false;
            return true;
        }
        Context app = Host.app;
        if (waiting && app != null && show(app, pendingID, pendingTitle, pendingMessage, pendingUrgency, pendingProgress, pendingPercent)) {
            waiting = false;
            permit = POSTED;
            return true;
        }
        waiting = false;
        permit = DENIED;
        return true;
    }

    private static boolean canPost(Context app) {
        NotificationManager nm = manager(app);
        if (nm == null || !nm.areNotificationsEnabled()) {
            return false;
        }
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            return app.checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED;
        }
        return true;
    }

    private static void ensure(Context app, String urgency) {
        NotificationManager nm = manager(app);
        if (nm == null) {
            return;
        }
        int importance = importance(urgency);
        String id = channelID(importance);
        if (nm.getNotificationChannel(id) != null) {
            return;
        }
        nm.createNotificationChannel(new NotificationChannel(id, channelName(app, urgency), importance));
    }

    private static boolean show(Context app, int id, String title, String message, String urgency, boolean progress, int percent) {
        NotificationManager nm = manager(app);
        if (nm == null) {
            return false;
        }
        ensure(app, urgency);
        String shown = title == null ? "" : title.trim();
        if (shown.isEmpty()) {
            shown = "Notification";
        }
        if (message == null) {
            message = "";
        }
        int importance = importance(urgency);
        Notification.Builder builder = new Notification.Builder(app, channelID(importance))
                .setSmallIcon(android.R.drawable.ic_dialog_info)
                .setContentTitle(shown)
                .setContentText(message)
                .setAutoCancel(true)
                .setOnlyAlertOnce(true);
        if (progress) {
            if (percent < 0) {
                percent = 0;
            } else if (percent > 100) {
                percent = 100;
            }
            builder.setProgress(100, percent, false);
        }
        int note = id;
        if (note == 0) {
            note = next++;
            if (next == Integer.MAX_VALUE) {
                next = 1;
            }
        }
        try {
            nm.notify(note, builder.build());
        } catch (SecurityException e) {
            return false;
        }
        return true;
    }

    private static int importance(String urgency) {
        if ("low".equals(urgency)) {
            return NotificationManager.IMPORTANCE_LOW;
        }
        if ("critical".equals(urgency)) {
            return NotificationManager.IMPORTANCE_HIGH;
        }
        return NotificationManager.IMPORTANCE_DEFAULT;
    }

    private static String channelID(int importance) {
        return "lewkit." + importance;
    }

    private static String channelName(Context app, String urgency) {
        CharSequence label = app.getApplicationInfo().loadLabel(app.getPackageManager());
        String name = label == null ? "" : label.toString().trim();
        if (name.isEmpty()) {
            name = "Notifications";
        }
        if ("low".equals(urgency)) {
            return name + " quiet";
        }
        if ("critical".equals(urgency)) {
            return name + " alerts";
        }
        return name;
    }

    private static NotificationManager manager(Context app) {
        Object service = app.getSystemService(Context.NOTIFICATION_SERVICE);
        if (service instanceof NotificationManager) {
            return (NotificationManager) service;
        }
        return null;
    }
}
