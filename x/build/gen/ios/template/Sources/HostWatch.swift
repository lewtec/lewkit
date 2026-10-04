import UIKit
import UniformTypeIdentifiers
import UserNotifications

/// Serves battery.json, daynight.txt, and request.json under the ios directory.
/// The poll thread waits. The main queue presents UIKit and is not blocked.
enum HostWatch {
    private static let lock = NSLock()
    private static var root: URL?
    private static var started = false
    private static var picker: UIDocumentPickerViewController?
    private static var pickerBox: PickerBox?
    private static var opener: UIDocumentInteractionController?
    private static var held: [URL] = []
    private static var batteryTokens: [NSObjectProtocol] = []
    private static var batteryProbe = 0
    private static var batteryKnown = false

    static func prepare(style: UIUserInterfaceStyle) {
        let dir = OpenDrop.cacheDir().appendingPathComponent("ios", isDirectory: true)
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        lock.lock()
        root = dir
        lock.unlock()
        publishDaynight(style)
        watchBattery()
    }

    static func start(directory: String) {
        guard !directory.isEmpty else { return }
        let dir = URL(fileURLWithPath: directory, isDirectory: true)
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        lock.lock()
        let already = started
        started = true
        root = dir
        lock.unlock()
        if already { return }
        try? FileManager.default.removeItem(at: dir.appendingPathComponent("request.json"))
        try? FileManager.default.removeItem(at: dir.appendingPathComponent("reply.txt"))
        DispatchQueue.global(qos: .userInitiated).async {
            poll(dir)
        }
    }

    static func publishDaynight(_ style: UIUserInterfaceStyle) {
        guard let dir = directory() else { return }
        let text = style == .dark ? "dark\n" : "light\n"
        try? text.write(to: dir.appendingPathComponent("daynight.txt"), atomically: true, encoding: .utf8)
    }

    /// Enables monitoring and samples until UIDevice leaves the unknown state.
    /// The first read at launch is unknown, and no notification follows if the
    /// charge does not change.
    static func watchBattery() {
        if batteryTokens.isEmpty {
            let center = NotificationCenter.default
            let queue = OperationQueue.main
            batteryTokens.append(center.addObserver(forName: UIDevice.batteryLevelDidChangeNotification, object: nil, queue: queue) { _ in
                _ = publishBattery()
            })
            batteryTokens.append(center.addObserver(forName: UIDevice.batteryStateDidChangeNotification, object: nil, queue: queue) { _ in
                _ = publishBattery()
            })
        }
        batteryProbe += 1
        let probe = batteryProbe
        UIDevice.current.isBatteryMonitoringEnabled = true
        probeBattery(probe, left: 8)
    }

    private static func probeBattery(_ probe: Int, left: Int) {
        guard probe == batteryProbe else { return }
        if publishBattery() || left <= 0 {
            return
        }
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.35) {
            UIDevice.current.isBatteryMonitoringEnabled = true
            probeBattery(probe, left: left - 1)
        }
    }

    /// Writes battery.json. An unknown sample does not replace a known one.
    /// Returns whether both state and level were known.
    @discardableResult
    static func publishBattery() -> Bool {
        guard let dir = directory() else { return false }
        let device = UIDevice.current
        device.isBatteryMonitoringEnabled = true
        let stateKnown = device.batteryState != .unknown
        let rawLevel = device.batteryLevel
        let levelKnown = rawLevel >= 0
        if !stateKnown && !levelKnown && batteryKnown {
            return false
        }
        if stateKnown || levelKnown {
            batteryKnown = true
        }
        let status: String
        switch device.batteryState {
        case .charging:
            status = "Charging"
        case .full:
            status = "Full"
        case .unplugged:
            status = "Discharging"
        default:
            status = "Unknown"
        }
        var percent = -1
        if levelKnown {
            percent = Int((rawLevel * 100).rounded())
            if percent > 100 { percent = 100 }
            if percent < 0 { percent = 0 }
        }
        guard let data = try? JSONSerialization.data(withJSONObject: ["status": status, "level": percent]) else { return false }
        try? data.write(to: dir.appendingPathComponent("battery.json"), options: .atomic)
        return stateKnown && levelKnown
    }

    private static func directory() -> URL? {
        lock.lock()
        defer { lock.unlock() }
        return root
    }

    private static func poll(_ root: URL) {
        let req = root.appendingPathComponent("request.json")
        let reply = root.appendingPathComponent("reply.txt")
        while true {
            if FileManager.default.fileExists(atPath: req.path) {
                let answer: String
                if let data = try? Data(contentsOf: req),
                   let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
                   let op = obj["op"] as? String, !op.isEmpty {
                    answer = perform(obj)
                } else {
                    answer = "error\nbad request"
                }
                try? answer.write(to: reply, atomically: true, encoding: .utf8)
                try? FileManager.default.removeItem(at: req)
                // Go deletes reply.txt after it reads it. A canceled call
                // already left, so do not wait on that file forever.
                let deadline = Date().addingTimeInterval(2)
                while FileManager.default.fileExists(atPath: reply.path), Date() < deadline {
                    Thread.sleep(forTimeInterval: 0.02)
                }
                try? FileManager.default.removeItem(at: reply)
            }
            Thread.sleep(forTimeInterval: 0.02)
        }
    }

    /// Shows the call on the main queue and waits on this background thread.
    private static func perform(_ obj: [String: Any]) -> String {
        let box = Answer()
        DispatchQueue.main.async {
            run(obj, box)
        }
        let answer = box.wait(seconds: 120)
        DispatchQueue.main.async {
            picker?.dismiss(animated: false)
            picker = nil
            pickerBox = nil
        }
        return answer
    }

    private static func run(_ obj: [String: Any], _ box: Answer) {
        switch obj["op"] as? String {
        case "clipboard":
            clipboard(obj, box)
        case "notify":
            notify(obj, box)
        case "open":
            open(obj, box)
        case "pick":
            pick(obj, box)
        default:
            box.finish("error\nbad request")
        }
    }

    private static func clipboard(_ obj: [String: Any], _ box: Answer) {
        if let png = obj["png"] as? String, !png.isEmpty {
            guard let data = Data(base64Encoded: png), let image = UIImage(data: data) else {
                box.finish("error\nbad image")
                return
            }
            UIPasteboard.general.image = image
            box.finish("ok\n")
            return
        }
        if let text = obj["text"] as? String {
            UIPasteboard.general.string = text
            box.finish("ok\n")
            return
        }
        box.finish("error\nempty clipboard")
    }

    private static func notify(_ obj: [String: Any], _ box: Answer) {
        let center = UNUserNotificationCenter.current()
        center.requestAuthorization(options: [.alert, .sound, .badge]) { granted, error in
            if let error {
                box.finish("error\n" + error.localizedDescription)
                return
            }
            if !granted {
                box.finish("error\nnotifications denied")
                return
            }
            let content = UNMutableNotificationContent()
            var title = obj["title"] as? String ?? ""
            if title.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                title = "Notification"
            }
            content.title = title
            content.body = obj["message"] as? String ?? ""
            switch obj["urgency"] as? String {
            case "low":
                content.interruptionLevel = .passive
            case "critical":
                content.interruptionLevel = .timeSensitive
                content.sound = .default
            default:
                content.interruptionLevel = .active
                content.sound = .default
            }
            let ident: String
            if let id = obj["id"] as? NSNumber, id.uint32Value != 0 {
                ident = "lewkit-\(id.uint32Value)"
            } else {
                ident = UUID().uuidString
            }
            center.add(UNNotificationRequest(identifier: ident, content: content, trigger: nil)) { err in
                if let err {
                    box.finish("error\n" + err.localizedDescription)
                    return
                }
                box.finish("ok\n")
            }
        }
    }

    private static func open(_ obj: [String: Any], _ box: Answer) {
        guard let target = obj["target"] as? String, let url = urlFrom(target) else {
            box.finish("error\nbad target")
            return
        }
        UIApplication.shared.open(url, options: [:]) { ok in
            if ok {
                box.finish("ok\n")
                return
            }
            DispatchQueue.main.async {
                presentFile(url, box)
            }
        }
    }

    private static func presentFile(_ url: URL, _ box: Answer) {
        guard url.isFileURL, FileManager.default.fileExists(atPath: url.path), let presenter = topViewController() else {
            box.finish("error\ncould not open")
            return
        }
        let doc = UIDocumentInteractionController(url: url)
        opener = doc
        let bounds = presenter.view.bounds
        let rect = CGRect(x: bounds.midX, y: bounds.midY, width: 1, height: 1)
        if doc.presentOpenInMenu(from: rect, in: presenter.view, animated: true) {
            box.finish("ok\n")
            return
        }
        box.finish("error\ncould not open")
    }

    private static func urlFrom(_ target: String) -> URL? {
        let text = target.trimmingCharacters(in: .whitespacesAndNewlines)
        if text.isEmpty { return nil }
        if text.hasPrefix("/") {
            return URL(fileURLWithPath: text)
        }
        guard let url = URL(string: text), let scheme = url.scheme, !scheme.isEmpty else {
            return nil
        }
        return url
    }

    private static func pick(_ obj: [String: Any], _ box: Answer) {
        guard let presenter = topViewController() else {
            box.finish("no activity\n")
            return
        }
        let save = obj["save"] as? Bool ?? false
        let folder = obj["folder"] as? Bool ?? false
        let multiple = obj["multiple"] as? Bool ?? false
        let pickerController: UIDocumentPickerViewController
        var savePath: String?
        var created = false
        if save {
            let file = inboxURL().appendingPathComponent(safeName(obj["name"] as? String ?? ""))
            if !FileManager.default.fileExists(atPath: file.path) {
                FileManager.default.createFile(atPath: file.path, contents: Data())
                created = true
            }
            savePath = file.path
            pickerController = UIDocumentPickerViewController(forExporting: [file], asCopy: true)
        } else {
            pickerController = UIDocumentPickerViewController(forOpeningContentTypes: types(obj, folder: folder), asCopy: !folder)
            pickerController.allowsMultipleSelection = multiple && !folder
        }
        if let dir = obj["directory"] as? String, !dir.isEmpty {
            pickerController.directoryURL = URL(fileURLWithPath: dir, isDirectory: true)
        }
        let delegate = PickerBox(box: box, savePath: savePath, removeOnCancel: created)
        pickerController.delegate = delegate
        pickerBox = delegate
        picker = pickerController
        if let pop = pickerController.popoverPresentationController {
            pop.sourceView = presenter.view
            let bounds = presenter.view.bounds
            pop.sourceRect = CGRect(x: bounds.midX, y: bounds.midY, width: 1, height: 1)
            pop.permittedArrowDirections = []
        }
        presenter.present(pickerController, animated: true)
    }

    private static func types(_ obj: [String: Any], folder: Bool) -> [UTType] {
        if folder {
            return [.folder]
        }
        let exts = obj["exts"] as? [String] ?? []
        let found = exts.compactMap { UTType(filenameExtension: $0) }
        if found.isEmpty {
            return [.content]
        }
        return found
    }

    private static func safeName(_ name: String) -> String {
        let base = (name as NSString).lastPathComponent
        if base.isEmpty || base == "." || base == ".." {
            return "untitled"
        }
        return base
    }

    private static func inboxURL() -> URL {
        let base = directory() ?? OpenDrop.cacheDir().appendingPathComponent("ios", isDirectory: true)
        let inbox = base.appendingPathComponent("inbox", isDirectory: true)
        try? FileManager.default.createDirectory(at: inbox, withIntermediateDirectories: true)
        return inbox
    }

    fileprivate static func keep(_ url: URL) -> String? {
        let inbox = inboxURL()
        let base = url.lastPathComponent.isEmpty ? "file" : url.lastPathComponent
        var dest = inbox.appendingPathComponent(base)
        if FileManager.default.fileExists(atPath: dest.path) {
            dest = inbox.appendingPathComponent(UUID().uuidString + "-" + base)
        }
        let scoped = url.startAccessingSecurityScopedResource()
        do {
            try FileManager.default.copyItem(at: url, to: dest)
            if scoped {
                url.stopAccessingSecurityScopedResource()
            }
            return dest.path
        } catch {
            if scoped {
                held.append(url)
                return url.path
            }
            return nil
        }
    }

    fileprivate static func jsonArray(_ paths: [String]) -> String {
        guard let data = try? JSONSerialization.data(withJSONObject: paths),
              let text = String(data: data, encoding: .utf8) else {
            return "[]"
        }
        return text
    }

    private static func topViewController() -> UIViewController? {
        let scenes = UIApplication.shared.connectedScenes.compactMap { $0 as? UIWindowScene }
        let window = scenes.flatMap(\.windows).first { $0.isKeyWindow } ?? scenes.flatMap(\.windows).first
        var controller = window?.rootViewController
        while let presented = controller?.presentedViewController {
            controller = presented
        }
        return controller
    }
}

private final class PickerBox: NSObject, UIDocumentPickerDelegate {
    private let box: Answer
    private let savePath: String?
    private let removeOnCancel: Bool

    init(box: Answer, savePath: String?, removeOnCancel: Bool) {
        self.box = box
        self.savePath = savePath
        self.removeOnCancel = removeOnCancel
    }

    func documentPicker(_ controller: UIDocumentPickerViewController, didPickDocumentsAt urls: [URL]) {
        if let savePath {
            box.finish("ok\n" + HostWatch.jsonArray([savePath]))
            return
        }
        var paths: [String] = []
        for url in urls {
            if let path = HostWatch.keep(url) {
                paths.append(path)
            }
        }
        if paths.isEmpty {
            box.finish("canceled\n")
            return
        }
        box.finish("ok\n" + HostWatch.jsonArray(paths))
    }

    func documentPickerWasCancelled(_ controller: UIDocumentPickerViewController) {
        if removeOnCancel, let savePath {
            try? FileManager.default.removeItem(atPath: savePath)
        }
        box.finish("canceled\n")
    }
}
