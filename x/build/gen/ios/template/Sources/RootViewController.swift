import UIKit
import WebKit

final class RootViewController: UIViewController, WKNavigationDelegate, WKUIDelegate {
    var onRetry: (() -> Void)?

    private let webView: WKWebView
    private let refresh = UIRefreshControl()
    private let splash = UIView()
    private let logo = UIImageView(image: UIImage(named: "SplashLogo"))
    private let statusLabel = UILabel()
    private let detailLabel = UILabel()
    private let spinner = UIActivityIndicatorView(style: .medium)
    private let retryButton = UIButton(type: .system)
    private let progressStack = UIStackView()
    private var appURL: URL?
    private var lastStatus = "Starting local server…"
    private var stuckWork: DispatchWorkItem?

    private static let stuckAfter: TimeInterval = 3

    init() {
        let config = WKWebViewConfiguration()
        webView = WKWebView(frame: .zero, configuration: config)
        super.init(nibName: nil, bundle: nil)
        webView.navigationDelegate = self
        webView.uiDelegate = self
        if #available(iOS 16.4, *) {
            webView.isInspectable = true
        }
    }

    @available(*, unavailable)
    required init?(coder: NSCoder) {
        fatalError("init(coder:) has not been implemented")
    }

    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = .systemBackground

        webView.translatesAutoresizingMaskIntoConstraints = false
        webView.scrollView.alwaysBounceVertical = true
        webView.scrollView.contentInsetAdjustmentBehavior = .automatic
        refresh.addTarget(self, action: #selector(pullToRefresh), for: .valueChanged)
        refresh.accessibilityLabel = "Reload"
        webView.scrollView.refreshControl = refresh

        splash.translatesAutoresizingMaskIntoConstraints = false
        splash.backgroundColor = .systemBackground

        logo.translatesAutoresizingMaskIntoConstraints = false
        logo.contentMode = .scaleAspectFit
        logo.accessibilityLabel = Bundle.main.object(forInfoDictionaryKey: "CFBundleDisplayName") as? String
            ?? Bundle.main.object(forInfoDictionaryKey: "CFBundleName") as? String
            ?? "App"

        statusLabel.translatesAutoresizingMaskIntoConstraints = false
        statusLabel.textAlignment = .center
        statusLabel.font = .preferredFont(forTextStyle: .body)
        statusLabel.textColor = .secondaryLabel
        statusLabel.adjustsFontForContentSizeCategory = true
        statusLabel.numberOfLines = 0
        statusLabel.text = "Starting local server…"

        detailLabel.translatesAutoresizingMaskIntoConstraints = false
        detailLabel.textAlignment = .center
        detailLabel.font = .preferredFont(forTextStyle: .footnote)
        detailLabel.textColor = .tertiaryLabel
        detailLabel.numberOfLines = 6
        detailLabel.adjustsFontForContentSizeCategory = true
        detailLabel.isHidden = true

        spinner.translatesAutoresizingMaskIntoConstraints = false

        retryButton.translatesAutoresizingMaskIntoConstraints = false
        retryButton.setTitle("Try again", for: .normal)
        retryButton.addTarget(self, action: #selector(retryTapped), for: .touchUpInside)
        retryButton.isHidden = true
        retryButton.accessibilityLabel = "Try again"

        progressStack.axis = .vertical
        progressStack.alignment = .center
        progressStack.spacing = 16
        progressStack.translatesAutoresizingMaskIntoConstraints = false
        progressStack.isHidden = true
        for v in [spinner, statusLabel, detailLabel, retryButton] {
            progressStack.addArrangedSubview(v)
        }

        view.addSubview(webView)
        view.addSubview(splash)
        splash.addSubview(logo)
        splash.addSubview(progressStack)

        NSLayoutConstraint.activate([
            webView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            webView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            webView.topAnchor.constraint(equalTo: view.topAnchor),
            webView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            splash.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            splash.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            splash.topAnchor.constraint(equalTo: view.topAnchor),
            splash.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            logo.centerXAnchor.constraint(equalTo: splash.centerXAnchor),
            logo.centerYAnchor.constraint(equalTo: splash.centerYAnchor),
            logo.widthAnchor.constraint(equalToConstant: 120), // keep in sync with LaunchScreen.storyboard
            logo.heightAnchor.constraint(equalToConstant: 120),
            progressStack.topAnchor.constraint(equalTo: logo.bottomAnchor, constant: 24),
            progressStack.leadingAnchor.constraint(equalTo: splash.leadingAnchor, constant: 32),
            progressStack.trailingAnchor.constraint(equalTo: splash.trailingAnchor, constant: -32),
        ])

        quietSplash()
    }

    /// Logo only. Status + spinner stay hidden until we are stuck or fail.
    func quietSplash() {
        endRefresh()
        splash.isHidden = false
        webView.isHidden = true
        progressStack.isHidden = true
        spinner.stopAnimating()
        retryButton.isHidden = true
        detailLabel.isHidden = true
        scheduleStuckReveal()
    }

    func noteStatus(_ message: String) {
        lastStatus = message
        if !progressStack.isHidden {
            statusLabel.text = message
        }
    }

    func showSplash(status: String, detail: String?, error: Bool) {
        lastStatus = status
        if error {
            showFailed(status: status, detail: detail)
            return
        }
        noteStatus(status)
        quietSplash()
    }

    func load(_ url: URL) {
        appURL = url
        noteStatus("Loading app…")
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

    @objc private func pullToRefresh() {
        reload()
    }

    private func endRefresh() {
        if refresh.isRefreshing {
            refresh.endRefreshing()
        }
    }

    private func scheduleStuckReveal() {
        stuckWork?.cancel()
        let work = DispatchWorkItem { [weak self] in
            self?.revealIfStuck()
        }
        stuckWork = work
        DispatchQueue.main.asyncAfter(deadline: .now() + Self.stuckAfter, execute: work)
    }

    private func revealIfStuck() {
        guard !splash.isHidden else { return }
        progressStack.isHidden = false
        statusLabel.isHidden = false
        statusLabel.text = lastStatus
        retryButton.isHidden = true
        spinner.isHidden = false
        spinner.startAnimating()
    }

    private func showFailed(status: String, detail: String?) {
        stuckWork?.cancel()
        endRefresh()
        splash.isHidden = false
        webView.isHidden = true
        progressStack.isHidden = false
        statusLabel.isHidden = false
        statusLabel.text = status
        if let detail, !detail.isEmpty {
            detailLabel.text = detail
            detailLabel.isHidden = false
        } else {
            detailLabel.text = ""
            detailLabel.isHidden = true
        }
        spinner.stopAnimating()
        spinner.isHidden = true
        retryButton.isHidden = false
    }

    private func hideSplash() {
        stuckWork?.cancel()
        splash.isHidden = true
        webView.isHidden = false
        endRefresh()
    }

    func webView(_ webView: WKWebView, didFinish navigation: WKNavigation!) {
        hideSplash()
    }

    func webView(_ webView: WKWebView, didFail navigation: WKNavigation!, withError error: Error) {
        showFailed(status: "Could not load the page", detail: error.localizedDescription)
    }

    func webView(_ webView: WKWebView, didFailProvisionalNavigation navigation: WKNavigation!, withError error: Error) {
        showFailed(status: "Could not load the page", detail: error.localizedDescription)
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
        UIApplication.shared.open(url)
    }
}
