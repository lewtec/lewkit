import UIKit

/// Presents dialogs Go writes as request.json under the ask directory.
/// Kinds: 1 alert, 2 confirm, 3 prompt, 4 choose. The reply is a status line.
enum AskWatch {
    private static let lock = NSLock()
    private static var started = false
    private static var current: UIAlertController?

    static func start(directory: String) {
        lock.lock()
        let already = started
        started = true
        lock.unlock()
        if already || directory.isEmpty {
            return
        }
        try? FileManager.default.createDirectory(atPath: directory, withIntermediateDirectories: true)
        // A swipe-away leaves request.json. Presenting it on the next launch
        // covers the window before the page exists.
        let root = URL(fileURLWithPath: directory, isDirectory: true)
        try? FileManager.default.removeItem(at: root.appendingPathComponent("request.json"))
        try? FileManager.default.removeItem(at: root.appendingPathComponent("reply.txt"))
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
                let answer = ask(kind: kind, title: title, body: body)
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

    /// Shows the dialog on the main queue and waits on this background thread.
    /// A nested main run loop turns the web view black and never delivers the tap.
    private static func ask(kind: Int, title: String, body: String) -> String {
        let box = Answer()
        DispatchQueue.main.async {
            show(box, kind: kind, title: title, body: body)
        }
        let answer = box.wait(seconds: 120)
        DispatchQueue.main.async {
            current?.dismiss(animated: false)
            current = nil
        }
        return answer
    }

    private static func show(_ box: Answer, kind: Int, title: String, body: String) {
        guard let presenter = topViewController() else {
            box.finish("no activity\n")
            return
        }
        let alert = UIAlertController(
            title: title.isEmpty ? fallbackTitle(kind) : title,
            message: (kind == 3 || kind == 4) ? nil : body,
            preferredStyle: kind == 4 ? .actionSheet : .alert
        )
        switch kind {
        case 2:
            alert.addAction(UIAlertAction(title: "Yes", style: .default) { _ in box.finish("yes\n") })
            alert.addAction(UIAlertAction(title: "No", style: .cancel) { _ in box.finish("no\n") })
        case 3:
            alert.addTextField { field in
                field.text = body
            }
            alert.addAction(UIAlertAction(title: "OK", style: .default) { _ in
                let text = alert.textFields?.first?.text ?? ""
                box.finish("ok\n" + text)
            })
            alert.addAction(UIAlertAction(title: "Cancel", style: .cancel) { _ in box.finish("canceled\n") })
        case 4:
            for line in body.split(separator: "\n", omittingEmptySubsequences: true) {
                let label = String(line)
                alert.addAction(UIAlertAction(title: label, style: .default) { _ in
                    box.finish("ok\n" + label)
                })
            }
            alert.addAction(UIAlertAction(title: "Cancel", style: .cancel) { _ in box.finish("canceled\n") })
            if let pop = alert.popoverPresentationController {
                pop.sourceView = presenter.view
                let bounds = presenter.view.bounds
                pop.sourceRect = CGRect(x: bounds.midX, y: bounds.midY, width: 1, height: 1)
                pop.permittedArrowDirections = []
            }
        default:
            alert.addAction(UIAlertAction(title: "OK", style: .default) { _ in box.finish("ok\n") })
        }
        current = alert
        presenter.present(alert, animated: true)
    }

    private static func fallbackTitle(_ kind: Int) -> String {
        switch kind {
        case 2:
            return "Confirm"
        case 3:
            return "Input"
        case 4:
            return "Choose"
        default:
            return "Error"
        }
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

/// One dialog result. finish is safe from the button and from the timeout.
final class Answer {
    private let lock = NSLock()
    private let ready = DispatchSemaphore(value: 0)
    private var value = "canceled\n"
    private var done = false

    func finish(_ next: String) {
        lock.lock()
        if done {
            lock.unlock()
            return
        }
        done = true
        value = next
        lock.unlock()
        ready.signal()
    }

    func wait(seconds: Double) -> String {
        _ = ready.wait(timeout: .now() + seconds)
        lock.lock()
        defer { lock.unlock() }
        return value
    }
}
