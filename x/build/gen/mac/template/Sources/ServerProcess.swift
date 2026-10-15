import Foundation

/// Spawns the packaged Go helper (ELETROCROMO_NO_UI) and waits for READY.
final class ServerProcess {
    static let helperName = "eletrocromo-server"
    static let readyPrefix = "ELETROCROMO_READY "

    private var process: Process?
    private let lock = NSLock()

    func start(
        onStatus: @escaping (String) -> Void,
        onReady: @escaping (URL) -> Void,
        onSurface: @escaping () -> Void = {},
        onExit: @escaping () -> Void = {},
        onFailed: @escaping (String) -> Void
    ) {
        let status = { (msg: String) in DispatchQueue.main.async { onStatus(msg) } }
        let ready = { (url: URL) in DispatchQueue.main.async { onReady(url) } }
        let surface = { DispatchQueue.main.async { onSurface() } }
        let exited = { DispatchQueue.main.async { onExit() } }
        let failed = { (msg: String) in DispatchQueue.main.async { onFailed(msg) } }

        DispatchQueue.global(qos: .userInitiated).async { [weak self] in
            guard let self else { return }
            self.run(status: status, ready: ready, surface: surface, exited: exited, failed: failed)
        }
    }

    func stop() {
        lock.lock()
        let proc = process
        process = nil
        lock.unlock()
        proc?.terminate()
        proc?.waitUntilExit()
    }

    private func run(
        status: @escaping (String) -> Void,
        ready: @escaping (URL) -> Void,
        surface: @escaping () -> Void,
        exited: @escaping () -> Void,
        failed: @escaping (String) -> Void
    ) {
        guard let helper = Self.helperURL() else {
            failed("missing \(Self.helperName) in the app bundle")
            return
        }

        let readyFile = FileManager.default.temporaryDirectory
            .appendingPathComponent("eletrocromo-ready-\(UUID().uuidString).url")
        try? FileManager.default.removeItem(at: readyFile)

        let proc = Process()
        proc.executableURL = helper
        proc.currentDirectoryURL = helper.deletingLastPathComponent()
        var env = ProcessInfo.processInfo.environment
        env["ELETROCROMO_NO_UI"] = "1"
        env["ELETROCROMO_NO_ENSURE"] = "1"
        env["ELETROCROMO_READY_FILE"] = readyFile.path
        env["NO_PROXY"] = "127.0.0.1,localhost,::1"
        env["no_proxy"] = "127.0.0.1,localhost,::1"
        OpenDrop.applyEnv(&env)
        let askDir = OpenDrop.cacheDir().appendingPathComponent("ask", isDirectory: true)
        try? FileManager.default.createDirectory(at: askDir, withIntermediateDirectories: true)
        env["ELETROCROMO_ASK_DIR"] = askDir.path
        AskWatch.start(directory: askDir.path)
        proc.environment = env

        let pipe = Pipe()
        proc.standardOutput = pipe
        proc.standardError = pipe

        lock.lock()
        process = proc
        lock.unlock()

        status("Binding loopback…")
        do {
            try proc.run()
        } catch {
            failed(error.localizedDescription)
            return
        }

        var stdoutURL: String?
        var stdoutSurface = false
        let reader = DispatchQueue(label: "br.tec.lew.eletrocromo.stdout")
        reader.async {
            let handle = pipe.fileHandleForReading
            while true {
                let data = handle.availableData
                if data.isEmpty { break }
                guard let chunk = String(data: data, encoding: .utf8) else { continue }
                for line in chunk.split(whereSeparator: \.isNewline) {
                    let text = String(line)
                    if Self.isSurfaceLine(text) {
                        stdoutSurface = true
                    } else if stdoutURL == nil, let parsed = Self.extractReadyURL(text) {
                        stdoutURL = parsed
                    }
                }
            }
        }

        var deadline = Date().addingTimeInterval(30)
        while Date() < deadline {
            if FileManager.default.fileExists(atPath: askDir.appendingPathComponent("request.json").path) {
                deadline = Date().addingTimeInterval(30)
            }
            if stdoutSurface || Self.readSurfaceFile(readyFile) {
                surface()
                proc.waitUntilExit()
                if self.stillCurrent(proc) {
                    if proc.terminationStatus == 0 {
                        exited()
                    } else {
                        failed("server process exited (exit=\(proc.terminationStatus))")
                    }
                }
                return
            }
            if let raw = stdoutURL ?? Self.readReadyFile(readyFile) {
                if let url = Self.forceLoopback(raw) {
                    ready(url)
                    proc.waitUntilExit()
                    if self.stillCurrent(proc) {
                        failed("server process exited")
                    }
                    return
                }
            }
            if !proc.isRunning {
                failed("no ELETROCROMO_READY (exit=\(proc.terminationStatus))")
                return
            }
            Thread.sleep(forTimeInterval: 0.05)
        }
        failed("timed out waiting for ELETROCROMO_READY")
        proc.terminate()
    }

    private func stillCurrent(_ proc: Process) -> Bool {
        lock.lock()
        defer { lock.unlock() }
        return process === proc
    }

    private static func helperURL() -> URL? {
        let fm = FileManager.default
        var candidates: [URL] = [
            Bundle.main.bundleURL.appendingPathComponent("Contents/MacOS/\(helperName)"),
        ]
        if let exe = Bundle.main.executableURL {
            candidates.append(exe.deletingLastPathComponent().appendingPathComponent(helperName))
        }
        for url in candidates {
            if fm.fileExists(atPath: url.path) {
                return url
            }
        }
        return nil
    }

    static func isSurfaceLine(_ line: String) -> Bool {
        let trimmed = line.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed == readyPrefix.trimmingCharacters(in: .whitespaces)
    }

    private static func readSurfaceFile(_ url: URL) -> Bool {
        guard let text = try? String(contentsOf: url, encoding: .utf8), !text.isEmpty else {
            return false
        }
        return text.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
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
