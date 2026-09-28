import AppKit

enum ShareWatch {
    private static var offset: UInt64 = 0
    private static var timer: Timer?

    static func start() {
        guard timer == nil else { return }
        timer = Timer.scheduledTimer(withTimeInterval: 0.2, repeats: true) { _ in
            poll()
        }
    }

    private static func poll() {
        let file = OpenDrop.cacheDir().appendingPathComponent("share.jsonl")
        guard let data = try? Data(contentsOf: file), UInt64(data.count) > offset else { return }
        let slice = data.dropFirst(Int(offset))
        offset = UInt64(data.count)
        let text = String(data: slice, encoding: .utf8) ?? ""
        for line in text.split(separator: "\n") {
            guard let items = decode(String(line)),
                  let window = NSApp.keyWindow ?? NSApp.windows.first
            else { continue }
            let picker = NSSharingServicePicker(items: items)
            let view = window.contentView ?? window.contentViewController?.view
            if let view {
                picker.show(relativeTo: view.bounds, of: view, preferredEdge: .minY)
            }
        }
    }

    private static func decode(_ line: String) -> [Any]? {
        guard let raw = line.data(using: .utf8),
              let obj = try? JSONSerialization.jsonObject(with: raw) as? [String: Any]
        else { return nil }
        var items: [Any] = []
        if let text = obj["text"] as? String, !text.isEmpty { items.append(text) }
        if let url = obj["url"] as? String, let u = URL(string: url) { items.append(u) }
        if let paths = obj["paths"] as? [String] {
            for p in paths {
                items.append(URL(fileURLWithPath: p))
            }
        }
        return items.isEmpty ? nil : items
    }
}
