import AppKit

/// Presents dialogs the helper writes as request.json under the ask directory.
/// Kinds: 1 alert, 2 confirm, 3 prompt, 4 choose. The reply is a status line.
enum AskWatch {
    private static let lock = NSLock()
    private static var started = false

    static func start(directory: String) {
        lock.lock()
        let already = started
        started = true
        lock.unlock()
        if already || directory.isEmpty {
            return
        }
        try? FileManager.default.createDirectory(atPath: directory, withIntermediateDirectories: true)
        DispatchQueue.global(qos: .userInitiated).async {
            poll(directory)
        }
    }

    private static func poll(_ directory: String) {
        let root = URL(fileURLWithPath: directory, isDirectory: true)
        let req = root.appendingPathComponent("request.json")
        let reply = root.appendingPathComponent("reply.txt")
        while true {
            if let data = try? Data(contentsOf: req),
               let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
                let kind = number(obj["kind"])
                let title = obj["title"] as? String ?? ""
                let body = obj["body"] as? String ?? ""
                let style = obj["style"] as? String ?? ""
                let answer = DispatchQueue.main.sync {
                    present(kind: kind, title: title, body: body, style: style)
                }
                try? answer.write(to: reply, atomically: true, encoding: .utf8)
                try? FileManager.default.removeItem(at: req)
                while FileManager.default.fileExists(atPath: reply.path) {
                    Thread.sleep(forTimeInterval: 0.02)
                }
            }
            Thread.sleep(forTimeInterval: 0.02)
        }
    }

    private static func number(_ value: Any?) -> Int {
        switch value {
        case let n as Int:
            return n
        case let n as NSNumber:
            return n.intValue
        default:
            return 0
        }
    }

    private static func present(kind: Int, title: String, body: String, style: String) -> String {
        switch kind {
        case 2:
            return confirm(title: title, message: body)
        case 3:
            return prompt(title: title, text: body)
        case 4:
            return choose(title: title, items: body)
        default:
            return alert(title: title, message: body, style: style)
        }
    }

    private static func alert(title: String, message: String, style: String) -> String {
        let alert = NSAlert()
        alert.messageText = title.isEmpty ? "Error" : title
        alert.informativeText = message
        alert.alertStyle = alertStyle(style)
        alert.addButton(withTitle: "OK")
        alert.runModal()
        return "ok\n"
    }

    private static func alertStyle(_ name: String) -> NSAlert.Style {
        switch name {
        case "warning", "warn":
            return .warning
        case "informational", "info":
            return .informational
        default:
            return .critical
        }
    }

    private static func confirm(title: String, message: String) -> String {
        let alert = NSAlert()
        alert.messageText = title.isEmpty ? "Confirm" : title
        alert.informativeText = message
        alert.addButton(withTitle: "Yes")
        alert.addButton(withTitle: "No")
        if alert.runModal() == .alertFirstButtonReturn {
            return "yes\n"
        }
        return "no\n"
    }

    private static func prompt(title: String, text: String) -> String {
        let alert = NSAlert()
        alert.messageText = title.isEmpty ? "Input" : title
        let field = NSTextField(frame: NSRect(x: 0, y: 0, width: 240, height: 24))
        field.stringValue = text
        alert.accessoryView = field
        alert.addButton(withTitle: "OK")
        alert.addButton(withTitle: "Cancel")
        if alert.runModal() == .alertFirstButtonReturn {
            return "ok\n" + field.stringValue
        }
        return "canceled\n"
    }

    private static func choose(title: String, items: String) -> String {
        let alert = NSAlert()
        alert.messageText = title.isEmpty ? "Choose" : title
        var labels: [String] = []
        for line in items.split(separator: "\n", omittingEmptySubsequences: true) {
            labels.append(String(line))
            alert.addButton(withTitle: String(line))
        }
        alert.addButton(withTitle: "Cancel")
        let response = alert.runModal()
        let index = response.rawValue - NSApplication.ModalResponse.alertFirstButtonReturn.rawValue
        if index >= 0 && index < labels.count {
            return "ok\n" + labels[index]
        }
        return "canceled\n"
    }
}
