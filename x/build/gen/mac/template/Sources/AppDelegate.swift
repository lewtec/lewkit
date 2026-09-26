import AppKit

/// NSApplication.delegate is weak; keep the instance alive for the process.
private var retainedDelegate: AppDelegate?

@main
enum AppMain {
    static func main() {
        let delegate = AppDelegate()
        retainedDelegate = delegate
        let app = NSApplication.shared
        app.setActivationPolicy(.regular)
        app.delegate = delegate
        _ = NSApplicationMain(CommandLine.argc, CommandLine.unsafeArgv)
    }
}

final class AppDelegate: NSObject, NSApplicationDelegate {
    private var window: MainWindow?
    private let server = ServerProcess()

    func applicationDidFinishLaunching(_ notification: Notification) {
        let win = MainWindow()
        window = win
        win.onRetry = { [weak self] in
            self?.restartServer()
        }
        win.show()
        ShareWatch.start()
        startServer()
        NSApp.activate(ignoringOtherApps: true)
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
        true
    }

    func applicationWillTerminate(_ notification: Notification) {
        server.stop()
    }

    func application(_ application: NSApplication, open urls: [URL]) {
        OpenDrop.deliver(urls)
    }

    private func startServer() {
        window?.showSplash(status: "Starting server…", detail: nil, error: false)
        server.start(
            onStatus: { [weak self] message in
                self?.window?.showSplash(status: message, detail: nil, error: false)
            },
            onReady: { [weak self] url in
                self?.window?.load(url)
            },
            onFailed: { [weak self] message in
                self?.window?.showSplash(status: "Server failed", detail: message, error: true)
            }
        )
    }

    private func restartServer() {
        server.stop()
        startServer()
    }
}
