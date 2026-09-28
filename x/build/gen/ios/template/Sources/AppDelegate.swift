import UIKit

@main
final class AppDelegate: UIResponder, UIApplicationDelegate {
    var window: UIWindow?
    private var root: RootViewController?
    private let server = ServerProcess()

    func application(
        _ application: UIApplication,
        didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?
    ) -> Bool {
        let root = RootViewController()
        self.root = root
        root.onRetry = { [weak self] in
            self?.restartServer()
        }
        let window = UIWindow(frame: UIScreen.main.bounds)
        window.rootViewController = root
        window.makeKeyAndVisible()
        self.window = window
        if let url = launchOptions?[.url] as? URL {
            OpenDrop.deliver([url])
        }
        OpenDrop.drainIncoming()
        ShareWatch.start()
        startServer()
        return true
    }

    func application(
        _ app: UIApplication,
        open url: URL,
        options: [UIApplication.OpenURLOptionsKey: Any] = [:]
    ) -> Bool {
        OpenDrop.deliver([url])
        return true
    }

    func applicationDidBecomeActive(_ application: UIApplication) {
        OpenDrop.drainIncoming()
    }

    func applicationWillTerminate(_ application: UIApplication) {
        server.stop()
    }

    private func startServer() {
        root?.quietSplash()
        server.start(
            onStatus: { [weak self] message in
                self?.root?.noteStatus(message)
            },
            onReady: { [weak self] url in
                self?.root?.load(url)
            },
            onFailed: { [weak self] message in
                self?.root?.showSplash(status: "Could not start the app", detail: message, error: true)
            }
        )
    }

    private func restartServer() {
        server.stop()
        startServer()
    }
}
