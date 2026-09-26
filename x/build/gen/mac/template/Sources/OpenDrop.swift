import Foundation

enum OpenDrop {
    static func cacheDir() -> URL {
        let base = FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask).first
            ?? FileManager.default.temporaryDirectory
        let id = Bundle.main.bundleIdentifier ?? "eletrocromo"
        let dir = base.appendingPathComponent(id, isDirectory: true)
        try? FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        return dir
    }

    static func applyEnv(_ env: inout [String: String]) {
        let support = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first
            ?? FileManager.default.temporaryDirectory
        let id = Bundle.main.bundleIdentifier ?? "eletrocromo"
        let root = support.appendingPathComponent(id, isDirectory: true)
        let data = root.appendingPathComponent("data", isDirectory: true)
        let config = root.appendingPathComponent("config", isDirectory: true)
        let cache = cacheDir()
        try? FileManager.default.createDirectory(at: data, withIntermediateDirectories: true)
        try? FileManager.default.createDirectory(at: config, withIntermediateDirectories: true)
        env["ELETROCROMO_DATA_DIR"] = data.path
        env["ELETROCROMO_CACHE_DIR"] = cache.path
        env["ELETROCROMO_CONFIG_DIR"] = config.path
        env["LEWKIT_APP_ID"] = "{{.PackageID}}"
        env["LEWKIT_DATA_DIR"] = data.path
        env["LEWKIT_CACHE_DIR"] = cache.path
        env["LEWKIT_CONFIG_DIR"] = config.path
    }

    static func deliver(_ urls: [URL]) {
        guard !urls.isEmpty else { return }
        var urlLines: [String] = []
        var files: [String] = []
        for url in urls {
            let scheme = url.scheme?.lowercased() ?? ""
            if scheme != "http", scheme != "https", scheme != "file" {
                urlLines.append(jsonLine(kind: "url", url: url.absoluteString, paths: nil))
                continue
            }
            if let path = materialize(url) {
                files.append(path)
            }
        }
        for line in urlLines {
            append(line)
        }
        if !files.isEmpty {
            append(jsonLine(kind: "files", url: nil, paths: files))
        }
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
