import Foundation

enum OpenDrop {
    struct Dirs {
        let data: URL
        let cache: URL
        let config: URL
    }

    static func cacheDir() -> URL {
        let base = FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask).first
            ?? FileManager.default.temporaryDirectory
        try? FileManager.default.createDirectory(at: base, withIntermediateDirectories: true)
        return base
    }

    static func dirs() -> Dirs {
        let support = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first
            ?? FileManager.default.temporaryDirectory
        let data = support.appendingPathComponent("data", isDirectory: true)
        let config = support.appendingPathComponent("config", isDirectory: true)
        let cache = cacheDir()
        try? FileManager.default.createDirectory(at: data, withIntermediateDirectories: true)
        try? FileManager.default.createDirectory(at: config, withIntermediateDirectories: true)
        return Dirs(data: data, cache: cache, config: config)
    }

    @discardableResult
    static func applyProcessEnv() -> Dirs {
        let d = dirs()
        setenv("ELETROCROMO_DATA_DIR", d.data.path, 1)
        setenv("ELETROCROMO_CACHE_DIR", d.cache.path, 1)
        setenv("ELETROCROMO_CONFIG_DIR", d.config.path, 1)
        setenv("LEWKIT_APP_ID", "{{.PackageID}}", 1)
        setenv("LEWKIT_DATA_DIR", d.data.path, 1)
        setenv("LEWKIT_CACHE_DIR", d.cache.path, 1)
        setenv("LEWKIT_CONFIG_DIR", d.config.path, 1)
        return d
    }

    static func deliver(_ urls: [URL]) {
        if urls.contains(where: { $0.host == "share-ready" }) {
            drainIncoming()
        }
        guard !urls.isEmpty else { return }
        var files: [String] = []
        for url in urls {
            if url.host == "share-ready" { continue }
            let scheme = url.scheme?.lowercased() ?? ""
            if scheme != "http", scheme != "https", scheme != "file" {
                append(jsonLine(kind: "url", url: url.absoluteString, paths: nil))
                continue
            }
            if let path = materialize(url) {
                files.append(path)
            }
        }
        if !files.isEmpty {
            append(jsonLine(kind: "files", url: nil, paths: files))
        }
    }

    static func drainIncoming() {
        let groupID = "group." + (Bundle.main.bundleIdentifier ?? "")
        guard let root = FileManager.default.containerURL(forSecurityApplicationGroupIdentifier: groupID) else {
            return
        }
        let src = root.appendingPathComponent("open.jsonl")
        guard let data = try? Data(contentsOf: src), !data.isEmpty else { return }
        let text = String(data: data, encoding: .utf8) ?? ""
        for line in text.split(whereSeparator: \.isNewline) {
            guard !line.isEmpty, let rewritten = adoptGroupLine(String(line)) else { continue }
            append(rewritten)
        }
        try? FileManager.default.removeItem(at: src)
    }

    // Copies group-container files into Cache/inbox and rewrites paths.
    private static func adoptGroupLine(_ line: String) -> String? {
        guard let raw = line.data(using: .utf8),
              var obj = try? JSONSerialization.jsonObject(with: raw) as? [String: Any]
        else { return nil }
        if let paths = obj["paths"] as? [String] {
            var copied: [String] = []
            for p in paths {
                if let dest = materialize(URL(fileURLWithPath: p)) {
                    copied.append(dest)
                    try? FileManager.default.removeItem(atPath: p)
                }
            }
            if copied.isEmpty { return nil }
            obj["paths"] = copied
            let out = try? JSONSerialization.data(withJSONObject: obj)
            return String(data: out ?? Data(), encoding: .utf8)
        }
        return line
    }

    private static func materialize(_ url: URL) -> String? {
        let inbox = cacheDir().appendingPathComponent("inbox", isDirectory: true)
        try? FileManager.default.createDirectory(at: inbox, withIntermediateDirectories: true)
        let dest = inbox.appendingPathComponent(UUID().uuidString + "-" + url.lastPathComponent)
        let scoped = url.startAccessingSecurityScopedResource()
        defer {
            if scoped { url.stopAccessingSecurityScopedResource() }
        }
        do {
            if FileManager.default.fileExists(atPath: dest.path) {
                try FileManager.default.removeItem(at: dest)
            }
            try FileManager.default.copyItem(at: url, to: dest)
            return dest.path
        } catch {
            return url.isFileURL ? url.path : nil
        }
    }

    private static func append(_ line: String) {
        let file = cacheDir().appendingPathComponent("open.jsonl")
        let data = (line + "\n").data(using: .utf8) ?? Data()
        if FileManager.default.fileExists(atPath: file.path) {
            if let h = try? FileHandle(forWritingTo: file) {
                defer { try? h.close() }
                _ = try? h.seekToEnd()
                try? h.write(contentsOf: data)
                return
            }
        }
        try? data.write(to: file, options: .atomic)
    }

    private static func jsonLine(kind: String, url: String?, paths: [String]?) -> String {
        var obj: [String: Any] = ["kind": kind]
        if let url { obj["url"] = url }
        if let paths { obj["paths"] = paths }
        let raw = try? JSONSerialization.data(withJSONObject: obj, options: [])
        return String(data: raw ?? Data(), encoding: .utf8) ?? ""
    }
}
