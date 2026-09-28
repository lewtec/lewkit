import AppKit
import WebKit

final class MainWindow: NSObject, NSWindowDelegate, WKNavigationDelegate, WKUIDelegate {
    var onRetry: (() -> Void)?

    private let window: NSWindow
    private let webView: WKWebView
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
                self?.reload()
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

    func windowWillClose(_ notification: Notification) {
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
        if let url = navigationAction.request.url {
            Self.openExternal(url)
        }
        return nil
    }

    private static func isLoopback(_ url: URL) -> Bool {
        let host = url.host?.lowercased() ?? ""
        return host == "127.0.0.1" || host == "localhost" || host == "::1"
    }

    private static func openExternal(_ url: URL) {
        if isLoopback(url) { return }
        let scheme = url.scheme?.lowercased() ?? ""
        if scheme == "about" || scheme == "blob" { return }
        NSWorkspace.shared.open(url)
    }
}
