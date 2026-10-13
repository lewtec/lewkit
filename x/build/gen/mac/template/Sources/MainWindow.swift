import AppKit
import WebKit

final class MainWindow: NSObject, NSWindowDelegate, WKNavigationDelegate, WKUIDelegate {
    var onRetry: (() -> Void)?

    private let window: NSWindow
    private let webView: WKWebView
    private var siblings: [SiblingWindow] = []
    private let splash: NSView
    private let statusLabel: NSTextField
    private let detailLabel: NSTextField
    private let spinner: NSProgressIndicator
    private let retryButton: NSButton
    private var appURL: URL?
    private var keyMonitor: Any?

    override init() {
        let title = Bundle.main.object(forInfoDictionaryKey: "CFBundleDisplayName") as? String
            ?? Bundle.main.object(forInfoDictionaryKey: "CFBundleName") as? String
            ?? "eletrocromo"

        window = NSWindow(
            contentRect: NSRect(x: 0, y: 0, width: 960, height: 640),
            styleMask: [.titled, .closable, .miniaturizable, .resizable],
            backing: .buffered,
            defer: false
        )
        window.title = title
        window.minSize = NSSize(width: 640, height: 400)
        window.center()

        let config = WKWebViewConfiguration()
        config.preferences.isElementFullscreenEnabled = false
        config.preferences.javaScriptCanOpenWindowsAutomatically = true
        webView = WKWebView(frame: .zero, configuration: config)
        if #available(macOS 13.3, *) {
            webView.isInspectable = true
        }
        webView.translatesAutoresizingMaskIntoConstraints = false

        splash = NSView(frame: .zero)
        splash.translatesAutoresizingMaskIntoConstraints = false
        splash.wantsLayer = true
        splash.layer?.backgroundColor = NSColor.windowBackgroundColor.cgColor

        statusLabel = NSTextField(labelWithString: "Starting…")
        statusLabel.alignment = .center
        statusLabel.font = .systemFont(ofSize: 15, weight: .medium)
        statusLabel.translatesAutoresizingMaskIntoConstraints = false

        detailLabel = NSTextField(labelWithString: "")
        detailLabel.alignment = .center
        detailLabel.font = .systemFont(ofSize: 12)
        detailLabel.textColor = .secondaryLabelColor
        detailLabel.lineBreakMode = .byWordWrapping
        detailLabel.maximumNumberOfLines = 6
        detailLabel.translatesAutoresizingMaskIntoConstraints = false
        detailLabel.isHidden = true

        spinner = NSProgressIndicator()
        spinner.style = .spinning
        spinner.controlSize = .regular
        spinner.translatesAutoresizingMaskIntoConstraints = false

        retryButton = NSButton(title: "Retry", target: nil, action: nil)
        retryButton.bezelStyle = .rounded
        retryButton.translatesAutoresizingMaskIntoConstraints = false
        retryButton.isHidden = true

        super.init()

        window.delegate = self
        webView.navigationDelegate = self
        webView.uiDelegate = self
        retryButton.target = self
        retryButton.action = #selector(retryTapped)

        guard let content = window.contentView else { return }
        content.addSubview(webView)
        content.addSubview(splash)
        splash.addSubview(statusLabel)
        splash.addSubview(detailLabel)
        splash.addSubview(spinner)
        splash.addSubview(retryButton)

        NSLayoutConstraint.activate([
            webView.leadingAnchor.constraint(equalTo: content.leadingAnchor),
            webView.trailingAnchor.constraint(equalTo: content.trailingAnchor),
            webView.topAnchor.constraint(equalTo: content.topAnchor),
            webView.bottomAnchor.constraint(equalTo: content.bottomAnchor),
            splash.leadingAnchor.constraint(equalTo: content.leadingAnchor),
            splash.trailingAnchor.constraint(equalTo: content.trailingAnchor),
            splash.topAnchor.constraint(equalTo: content.topAnchor),
            splash.bottomAnchor.constraint(equalTo: content.bottomAnchor),
            statusLabel.centerXAnchor.constraint(equalTo: splash.centerXAnchor),
            statusLabel.centerYAnchor.constraint(equalTo: splash.centerYAnchor, constant: -12),
            statusLabel.leadingAnchor.constraint(greaterThanOrEqualTo: splash.leadingAnchor, constant: 24),
            statusLabel.trailingAnchor.constraint(lessThanOrEqualTo: splash.trailingAnchor, constant: -24),
            detailLabel.topAnchor.constraint(equalTo: statusLabel.bottomAnchor, constant: 8),
            detailLabel.leadingAnchor.constraint(equalTo: splash.leadingAnchor, constant: 32),
            detailLabel.trailingAnchor.constraint(equalTo: splash.trailingAnchor, constant: -32),
            spinner.bottomAnchor.constraint(equalTo: statusLabel.topAnchor, constant: -16),
            spinner.centerXAnchor.constraint(equalTo: splash.centerXAnchor),
            retryButton.topAnchor.constraint(equalTo: detailLabel.bottomAnchor, constant: 16),
            retryButton.centerXAnchor.constraint(equalTo: splash.centerXAnchor),
        ])

        installTitlebarReload()

        keyMonitor = NSEvent.addLocalMonitorForEvents(matching: .keyDown) { [weak self] event in
            if event.modifierFlags.contains(.command),
               event.charactersIgnoringModifiers == "r"
            {
                self?.reloadFocused()
                return nil
            }
            return event
        }
    }

    /// Reload sits in the existing title bar (trailing). Not a toolbar strip.
    private func installTitlebarReload() {
        let button = NSButton(frame: NSRect(x: 0, y: 0, width: 28, height: 22))
        button.bezelStyle = .inline
        button.isBordered = false
        button.image = NSImage(systemSymbolName: "arrow.clockwise", accessibilityDescription: "Reload")
        button.imagePosition = .imageOnly
        button.imageScaling = .scaleProportionallyDown
        button.toolTip = "Reload"
        button.setAccessibilityLabel("Reload")
        button.target = self
        button.action = #selector(reloadTapped)

        let host = NSView(frame: NSRect(x: 0, y: 0, width: 32, height: 22))
        button.frame = host.bounds
        button.autoresizingMask = [.width, .height]
        host.addSubview(button)

        let accessory = NSTitlebarAccessoryViewController()
        accessory.layoutAttribute = .right
        accessory.view = host
        window.addTitlebarAccessoryViewController(accessory)
    }

    deinit {
        if let keyMonitor {
            NSEvent.removeMonitor(keyMonitor)
        }
    }

    func show() {
        window.makeKeyAndOrderFront(nil)
    }

    func showSplash(status: String, detail: String?, error: Bool) {
        splash.isHidden = false
        webView.isHidden = true
        statusLabel.stringValue = status
        if let detail, !detail.isEmpty {
            detailLabel.stringValue = detail
            detailLabel.isHidden = false
        } else {
            detailLabel.stringValue = ""
            detailLabel.isHidden = true
        }
        if error {
            spinner.stopAnimation(nil)
            spinner.isHidden = true
            retryButton.isHidden = false
        } else {
            spinner.isHidden = false
            spinner.startAnimation(nil)
            retryButton.isHidden = true
        }
    }

    func load(_ url: URL) {
        appURL = url
        showSplash(status: "Loading…", detail: nil, error: false)
        webView.load(URLRequest(url: url))
    }

    func reload() {
        if let url = appURL {
            webView.load(URLRequest(url: url))
        } else {
            webView.reload()
        }
    }

    @objc private func retryTapped() {
        onRetry?()
    }

    @objc private func reloadTapped() {
        reload()
    }

    private func reloadFocused() {
        if let sibling = siblings.first(where: { $0.window.isKeyWindow }) {
            sibling.reload()
            return
        }
        reload()
    }

    func windowWillClose(_ notification: Notification) {
        if !siblings.isEmpty {
            return
        }
        if let keyMonitor {
            NSEvent.removeMonitor(keyMonitor)
            self.keyMonitor = nil
        }
    }

    func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        splash.isHidden = true
        webView.isHidden = false
    }

    func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
        showSplash(status: "Load failed", detail: error.localizedDescription, error: true)
    }

    func webView(_ webView: WKWebView, didFailProvisionalNavigation navigation: WKNavigation!, withError error: Error) {
        showSplash(status: "Load failed", detail: error.localizedDescription, error: true)
    }

    func webView(
        _ webView: WKWebView,
        decidePolicyFor navigationAction: WKNavigationAction,
        decisionHandler: @escaping (WKNavigationActionPolicy) -> Void
    ) {
        if navigationAction.targetFrame == nil {
            if let url = navigationAction.request.url, Self.leavesApp(url) {
                Self.openExternal(url)
                decisionHandler(.cancel)
                return
            }
            decisionHandler(.allow)
            return
        }
        guard let url = navigationAction.request.url else {
            decisionHandler(.cancel)
            return
        }
        if Self.isLoopback(url) {
            decisionHandler(.allow)
            return
        }
        Self.openExternal(url)
        decisionHandler(.cancel)
    }

    func webView(
        _ webView: WKWebView,
        createWebViewWith configuration: WKWebViewConfiguration,
        for navigationAction: WKNavigationAction,
        windowFeatures: WKWindowFeatures
    ) -> WKWebView? {
        if let url = navigationAction.request.url, Self.leavesApp(url) {
            Self.openExternal(url)
            return nil
        }
        let sibling = SiblingWindow(
            configuration: configuration,
            features: windowFeatures,
            title: window.title,
            cascade: siblings.count,
            uiDelegate: self
        )
        sibling.onClose = { [weak self, weak sibling] in
            guard let self, let sibling else { return }
            self.siblings.removeAll { $0 === sibling }
        }
        siblings.append(sibling)
        return sibling.webView
    }

    func webViewDidClose(_ webView: WKWebView) {
        guard let sibling = siblings.first(where: { $0.webView == webView }) else { return }
        sibling.close()
    }

    private static func isLoopback(_ url: URL) -> Bool {
        let host = url.host?.lowercased() ?? ""
        return host == "127.0.0.1" || host == "localhost" || host == "::1"
    }

    // New windows only. A same-frame link still uses the loopback check above.
    // Pages stay in this process. mailto and other schemes leave.
    fileprivate static func leavesApp(_ url: URL) -> Bool {
        if isLoopback(url) { return false }
        switch url.scheme?.lowercased() ?? "" {
        case "", "about", "blob", "data", "http", "https":
            return false
        default:
            return true
        }
    }

    // decidePolicyFor and createWebViewWith can both see one new-window request.
    private static var externalOpened: Set<String> = []

    fileprivate static func openExternal(_ url: URL) {
        if isLoopback(url) { return }
        let scheme = url.scheme?.lowercased() ?? ""
        if scheme == "about" || scheme == "blob" { return }
        let key = url.absoluteString
        if externalOpened.contains(key) {
            return
        }
        externalOpened.insert(key)
        DispatchQueue.main.async {
            externalOpened.remove(key)
        }
        NSWorkspace.shared.open(url)
    }
}

// Another window of this app. WebKit loads the request in the view we return.
final class SiblingWindow: NSObject, NSWindowDelegate, WKNavigationDelegate {
    let window: NSWindow
    let webView: WKWebView
    var onClose: (() -> Void)?
    private let fallbackTitle: String
    private var closing = false

    init(configuration: WKWebViewConfiguration, features: WKWindowFeatures, title: String, cascade: Int, uiDelegate: WKUIDelegate) {
        fallbackTitle = title
        let size = NSSize(
            width: Self.points(features.width, fallback: 960),
            height: Self.points(features.height, fallback: 640)
        )
        window = NSWindow(
            contentRect: NSRect(origin: .zero, size: size),
            styleMask: [.titled, .closable, .miniaturizable, .resizable],
            backing: .buffered,
            defer: false
        )
        window.title = title
        window.minSize = NSSize(width: 320, height: 240)
        // This object owns the window. AppKit must not release it on close.
        window.isReleasedWhenClosed = false
        webView = WKWebView(frame: .zero, configuration: configuration)
        if #available(macOS 13.3, *) {
            webView.isInspectable = true
        }
        super.init()
        webView.uiDelegate = uiDelegate
        webView.navigationDelegate = self
        window.delegate = self
        window.contentView = webView
        window.center()
        if cascade > 0 {
            let step = CGFloat(cascade % 8) * 28
            window.setFrameOrigin(NSPoint(x: window.frame.origin.x + step, y: window.frame.origin.y - step))
        }
        window.makeKeyAndOrderFront(nil)
    }

    func reload() {
        webView.reload()
    }

    func close() {
        if closing { return }
        closing = true
        window.close()
    }

    func webView(
        _ webView: WKWebView,
        decidePolicyFor navigationAction: WKNavigationAction,
        decisionHandler: @escaping (WKNavigationActionPolicy) -> Void
    ) {
        if navigationAction.targetFrame == nil {
            if let url = navigationAction.request.url, MainWindow.leavesApp(url) {
                MainWindow.openExternal(url)
                decisionHandler(.cancel)
                return
            }
            decisionHandler(.allow)
            return
        }
        guard let url = navigationAction.request.url else {
            decisionHandler(.cancel)
            return
        }
        if MainWindow.leavesApp(url) {
            MainWindow.openExternal(url)
            decisionHandler(.cancel)
            return
        }
        decisionHandler(.allow)
    }

    func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        if let title = webView.title, !title.isEmpty {
            window.title = title
            return
        }
        window.title = fallbackTitle
    }

    func windowWillClose(_ notification: Notification) {
        closing = true
        let notify = onClose
        onClose = nil
        DispatchQueue.main.async {
            notify?()
        }
    }

    private static func points(_ number: NSNumber?, fallback: CGFloat) -> CGFloat {
        guard let number else { return fallback }
        let value = CGFloat(number.doubleValue)
        if value < 1 { return fallback }
        return value
    }
}
