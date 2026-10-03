import Foundation

/// Starts the in-process Go archive (ELETROCROMO_NO_UI) and waits for READY.
final class ServerProcess {
    static let readyPrefix = "ELETROCROMO_READY "

    private let lock = NSLock()
    private var started = false

    func start(
        onStatus: @escaping (String) -> Void,
        onReady: @escaping (URL) -> Void,
        onFailed: @escaping (String) -> Void
    ) {
        let status = { (msg: String) in DispatchQueue.main.async { onStatus(msg) } }
        let ready = { (url: URL) in DispatchQueue.main.async { onReady(url) } }
        let failed = { (msg: String) in DispatchQueue.main.async { onFailed(msg) } }

        DispatchQueue.global(qos: .userInitiated).async { [weak self] in
            guard let self else { return }
            self.run(status: status, ready: ready, failed: failed)
        }
    }

    func stop() {
        // In-process Go lives with the app process. A second start is a no-op.
    }

    private func run(
        status: @escaping (String) -> Void,
        ready: @escaping (URL) -> Void,
        failed: @escaping (String) -> Void
    ) {
        lock.lock()
        let already = started
        if !already {
            started = true
        }
        lock.unlock()
        if already {
            failed("server already running in-process")
            return
        }

        let readyFile = FileManager.default.temporaryDirectory
            .appendingPathComponent("eletrocromo-ready-\(UUID().uuidString).url")
        try? FileManager.default.removeItem(at: readyFile)

        status("Waiting for server…")
        let dirs = OpenDrop.applyProcessEnv()
        DispatchQueue.global(qos: .userInitiated).async {
            readyFile.path.withCString { ready in
                dirs.data.path.withCString { data in
                    dirs.cache.path.withCString { cache in
                        dirs.config.path.withCString { config in
                            EletrocromoStart(
                                UnsafeMutablePointer(mutating: ready),
                                UnsafeMutablePointer(mutating: data),
                                UnsafeMutablePointer(mutating: cache),
                                UnsafeMutablePointer(mutating: config)
                            )
                        }
                    }
                }
            }
        }

        let deadline = Date().addingTimeInterval(30)
        while Date() < deadline {
            if let raw = Self.readReadyFile(readyFile), let url = Self.forceLoopback(raw) {
                ready(url)
                return
            }
            Thread.sleep(forTimeInterval: 0.05)
        }
        failed("timed out waiting for ELETROCROMO_READY")
    }

    static func extractReadyURL(_ line: String) -> String? {
        let trimmed = line.trimmingCharacters(in: .whitespacesAndNewlines)
        if let range = trimmed.range(of: readyPrefix) {
            let rest = trimmed[range.upperBound...].trimmingCharacters(in: .whitespaces)
            if rest.hasPrefix("http://") || rest.hasPrefix("https://") {
                return String(rest)
            }
        }
        if trimmed.hasPrefix("http://") || trimmed.hasPrefix("https://") {
            return trimmed
        }
        return nil
    }

    private static func readReadyFile(_ url: URL) -> String? {
        guard let text = try? String(contentsOf: url, encoding: .utf8) else {
            return nil
        }
        let first = text.split(whereSeparator: \.isNewline).first.map(String.init) ?? text
        return extractReadyURL(readyPrefix + first) ?? extractReadyURL(first)
    }

    static func forceLoopback(_ raw: String) -> URL? {
        guard var comps = URLComponents(string: raw) else { return nil }
        let host = comps.host ?? ""
        if host.isEmpty || host == "localhost" || host == "::1" {
            comps.host = "127.0.0.1"
        }
        return comps.url
    }

    static func redactURL(_ raw: String) -> String {
        guard var comps = URLComponents(string: raw) else { return raw }
        comps.query = nil
        comps.fragment = nil
        return comps.string ?? raw
    }
}
