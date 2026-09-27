import AppKit
import UniformTypeIdentifiers

final class AppDelegate: NSObject, NSApplicationDelegate {
  var window: NSWindow!
  let heading = NSTextField(labelWithString: "Connect this Mac")
  let detail = NSTextField(
    wrappingLabelWithString: "Install the local collector and connect it to your workspace.")
  let person = NSTextField(
    wrappingLabelWithString: "Download a connection file from People → Connect device.")
  let address = NSTextField(string: "")
  let policy = NSTextField(
    wrappingLabelWithString: "Choose a connection file to review the collection policy.")
  let consent = NSButton(
    checkboxWithTitle: "I agree to this collection policy on this Mac", target: nil, action: nil)
  let choose = NSButton(title: "Choose connection file…", target: nil, action: nil)
  let install = NSButton(title: "Agree & connect", target: nil, action: nil)
  let status = NSTextField(wrappingLabelWithString: "")
  let appVersion = NSTextField(labelWithString: "")
  let serviceDetails = NSTextField(wrappingLabelWithString: "Checking background services…")
  let syncDetails = NSTextField(wrappingLabelWithString: "")
  let syncToggle = NSButton(title: "Checking…", target: nil, action: nil)
  let refresh = NSButton(title: "Refresh status", target: nil, action: nil)
  var statusTimer: Timer?
  var refreshing = false
  var nextSyncAction = ""
  var configured = false
  let spinner = NSProgressIndicator()
  let controls = NSStackView()
  var pendingPath: String?
  var version = 0
  var running = false
  var workspace = ""
  var process: Process?
  var root: URL {
    FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent(
      "Library/Application Support/Shared Account AI Usage Monitor")
  }
  let viewerControls = NSStackView()

  func applicationDidFinishLaunching(_ notification: Notification) {
    NSApp.setActivationPolicy(.regular)
    window = NSWindow(
      contentRect: NSRect(x: 0, y: 0, width: 610, height: 650),
      styleMask: [.titled, .closable, .miniaturizable], backing: .buffered, defer: false)
    window.title = "Shared Account AI Usage Monitor"
    window.isReleasedWhenClosed = false
    let stack = NSStackView()
    stack.orientation = .vertical
    stack.alignment = .leading
    stack.spacing = 18
    stack.translatesAutoresizingMaskIntoConstraints = false
    let content = window.contentView!
    content.addSubview(stack)
    NSLayoutConstraint.activate([
      stack.leadingAnchor.constraint(equalTo: content.leadingAnchor, constant: 32),
      stack.trailingAnchor.constraint(equalTo: content.trailingAnchor, constant: -32),
      stack.topAnchor.constraint(equalTo: content.topAnchor, constant: 30),
      stack.bottomAnchor.constraint(lessThanOrEqualTo: content.bottomAnchor, constant: -24),
    ])
    heading.font = .systemFont(ofSize: 28, weight: .semibold)
    detail.textColor = .secondaryLabelColor
    detail.font = .systemFont(ofSize: 14)
    let release = Bundle.main.object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "Unknown"
    let build = Bundle.main.object(forInfoDictionaryKey: "CFBundleVersion") as? String ?? "Unknown"
    appVersion.stringValue = "Version \(release) (build \(build))"
    appVersion.font = .systemFont(ofSize: 12)
    appVersion.textColor = .secondaryLabelColor
    serviceDetails.font = .systemFont(ofSize: 13)
    syncDetails.font = .systemFont(ofSize: 13)
    serviceDetails.isHidden = true
    syncDetails.isHidden = true
    syncToggle.target = self
    syncToggle.action = #selector(toggleSync)
    syncToggle.isEnabled = false
    refresh.target = self
    refresh.action = #selector(refreshStatus)
    address.placeholderString = "https://your-workspace.example.com"
    address.font = .systemFont(ofSize: 14)
    policy.font = .systemFont(ofSize: 14)
    policy.maximumNumberOfLines = 8
    choose.target = self
    choose.action = #selector(selectFile)
    consent.target = self
    consent.action = #selector(updateButton)
    install.target = self
    install.action = #selector(connect)
    install.bezelStyle = .rounded
    install.keyEquivalent = "\r"
    controls.orientation = .horizontal
    controls.spacing = 12
    controls.addArrangedSubview(install)
    spinner.style = .spinning
    spinner.controlSize = .small
    spinner.isDisplayedWhenStopped = false
    controls.addArrangedSubview(spinner)
    viewerControls.orientation = .horizontal
    viewerControls.spacing = 12
    viewerControls.isHidden = true
    for (title, action) in [
      ("Open AgentsView", #selector(openAgentsView)),
      ("Copy local access key", #selector(copyLocalKey)),
    ] {
      viewerControls.addArrangedSubview(NSButton(title: title, target: self, action: action))
    }
    for v in [
      heading, appVersion, detail, choose, person, address, policy, consent, serviceDetails, syncDetails, controls, viewerControls, status,
    ] {
      stack.addArrangedSubview(v)
    }
    for v in [detail, person, address, policy, serviceDetails, syncDetails, status] {
      v.widthAnchor.constraint(equalTo: stack.widthAnchor).isActive = true
    }
    status.font = .systemFont(ofSize: 13)
    status.maximumNumberOfLines = 6
    let credit = NSTextField(
      wrappingLabelWithString:
        "Uses AgentsView 0.44.0, installed separately from its official release. Sync runs in the background under your Mac account."
    )
    credit.font = .systemFont(ofSize: 11)
    credit.textColor = .secondaryLabelColor
    stack.addArrangedSubview(credit)
    credit.widthAnchor.constraint(equalTo: stack.widthAnchor).isActive = true
    let menu = NSMenu()
    let item = NSMenuItem()
    menu.addItem(item)
    let appMenu = NSMenu()
    appMenu.addItem(
      withTitle: "Quit AI Usage Monitor", action: #selector(NSApplication.terminate(_:)),
      keyEquivalent: "q")
    item.submenu = appMenu
    NSApp.mainMenu = menu
    window.center()
    window.makeKeyAndOrderFront(nil)
    NSApp.activate(ignoringOtherApps: true)
    updateButton()
    if let path = pendingPath { load(path) } else { existing() }
  }
  func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool {
    return true
  }
  func application(_ sender: NSApplication, openFiles filenames: [String]) {
    if let path = filenames.first {
      if window == nil { pendingPath = path } else if !running { load(path) }
    }
    sender.reply(toOpenOrPrint: .success)
  }
  @objc func selectFile() {
    let panel = NSOpenPanel()
    panel.allowedContentTypes = [UTType(filenameExtension: "aiusage") ?? .json, .json]
    panel.allowsMultipleSelection = false
    panel.canChooseDirectories = false
    panel.beginSheetModal(for: window) { response in
      if response == .OK, let path = panel.url?.path { self.load(path) }
    }
  }
  func load(_ path: String) {
    if FileManager.default.fileExists(atPath: root.appendingPathComponent("team-agent.json").path) {
      existing()
      status.stringValue = "This Mac is already configured. A new connection file is not needed."
      return
    }
    do {
      let data = try Data(contentsOf: URL(fileURLWithPath: path))
      guard data.count < 65536,
        let object = try JSONSerialization.jsonObject(with: data) as? [String: Any],
        let p = object["policy"] as? [String: Any], let v = p["version"] as? Int, v > 0,
        object["invitation"] is String
      else { throw NSError(domain: "Invalid", code: 1) }
      pendingPath = path
      version = v
      address.stringValue = object["server"] as? String ?? ""
      address.isEditable = address.stringValue.isEmpty
      person.stringValue =
        "Connect as: \(object["person"] as? String ?? "person assigned to this invitation")"
      let content = p["content"] as? String ?? "unknown"
      let redaction = p["redaction"] as? String ?? "unknown"
      let visibility = p["visibility"] as? String ?? "unknown"
      let retention = p["retention_days"] as? Int ?? 0
      policy.stringValue =
        "Collection: \(content == "full" ? "Full available prompts, replies and tool content" : content)\nRedaction: \(redaction == "none" ? "None — content is sent without masking" : redaction)\nVisible to: \(visibility == "team" ? "Everyone in your workspace" : "You and workspace managers")\nRetention: \(retention == 0 ? "Until deleted" : "\(retention) days") · Policy \(v)"
      consent.state = .off
      status.stringValue = "Review the workspace address and policy before connecting."
      updateButton()
    } catch {
      pendingPath = nil
      version = 0
      status.stringValue =
        "This file could not be read. Download a new connection file from your workspace."
      updateButton()
    }
  }
  @objc func updateButton() {
    install.isEnabled = !running && pendingPath != nil && version > 0 && consent.state == .on
  }
  @objc func connect() {
    guard let path = pendingPath, let components = URLComponents(string: address.stringValue),
      components.scheme == "https", components.host != nil, components.user == nil,
      components.password == nil, components.path.isEmpty, components.query == nil,
      components.fragment == nil
    else {
      status.stringValue = "Enter the HTTPS workspace address without a path."
      return
    }
    workspace = address.stringValue
    // Keep the app accessible after the downloaded archive is removed.
    let apps = FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent(
      "Applications")
    let destination = apps.appendingPathComponent("AI Usage Monitor.app")
    do {
      try FileManager.default.createDirectory(at: apps, withIntermediateDirectories: true)
      if !FileManager.default.fileExists(atPath: destination.path) {
        try FileManager.default.copyItem(at: Bundle.main.bundleURL, to: destination)
      }
    } catch {
      status.stringValue =
        "Could not save the app in ~/Applications. Check folder permissions and retry."
      return
    }
    execute([
      "install", "--invitation-file", path, "--server", workspace, "--ack-version", String(version),
      "--resources", Bundle.main.resourcePath!,
    ]) { ok in
      if ok {
        self.status.stringValue =
          "Connected. AgentsView is indexing local history; sessions appear as sync progresses."
        self.existing()
      }
    }
  }
  func execute(_ args: [String], completion: @escaping (Bool) -> Void) {
    if running { return }
    running = true
    choose.isEnabled = false
    consent.isEnabled = false
    install.isEnabled = false
    syncToggle.isEnabled = false
    refresh.isEnabled = false
    spinner.startAnimation(nil)
    let task = Process()
    task.executableURL = Bundle.main.url(forResource: "team-setup", withExtension: nil)
    task.arguments = args
    let pipe = Pipe()
    task.standardOutput = pipe
    task.standardError = pipe
    pipe.fileHandleForReading.readabilityHandler = { handle in
      let data = handle.availableData
      if data.isEmpty { return }
      if let text = String(data: data, encoding: .utf8) {
        DispatchQueue.main.async {
          self.status.stringValue = text.trimmingCharacters(in: .whitespacesAndNewlines)
        }
      }
    }
    task.terminationHandler = { task in
      DispatchQueue.main.async {
        pipe.fileHandleForReading.readabilityHandler = nil
        self.running = false
        self.choose.isEnabled = true
        self.consent.isEnabled = true
        self.spinner.stopAnimation(nil)
        self.updateButton()
        completion(task.terminationStatus == 0)
        self.refresh.isEnabled = true
        if self.configured { self.refreshStatus() }
      }
    }
    process = task
    do { try task.run() } catch {
      running = false
      spinner.stopAnimation(nil)
      status.stringValue = "Setup could not start. Download the complete app again."
      choose.isEnabled = true
      consent.isEnabled = true
      updateButton()
      refresh.isEnabled = true
      if configured { refreshStatus() }
    }
  }
  func existing() {
    guard let data = try? Data(contentsOf: root.appendingPathComponent("team-agent.json")),
      let cfg = (try? JSONSerialization.jsonObject(with: data)) as? [String: Any],
      let server = cfg["server"] as? String
    else { return }
    workspace = server
    configured = true
    heading.stringValue = "Checking sync status…"
    detail.stringValue =
      "You can close or quit this app. When enabled, background services continue while you’re logged in and your Mac is awake. Use Pause sync to stop them."
    for v in [choose, person, address, policy, consent, install] { v.isHidden = true }
    if controls.arrangedSubviews.count == 2 {
      controls.addArrangedSubview(NSButton(title: "Open workspace", target: self, action: #selector(openWorkspace)))
      controls.addArrangedSubview(syncToggle)
      controls.addArrangedSubview(refresh)
    }
    serviceDetails.isHidden = false
    syncDetails.isHidden = false
    viewerControls.isHidden = false
    window.setContentSize(NSSize(width: 610, height: 650))
    refreshStatus()
    if statusTimer == nil {
      statusTimer = Timer.scheduledTimer(withTimeInterval: 10, repeats: true) { [weak self] _ in
        self?.refreshStatus()
      }
    }
  }
  @objc func refreshStatus() {
    guard configured, !running, !refreshing else { return }
    refreshing = true
    refresh.isEnabled = false
    DispatchQueue.global(qos: .utility).async {
      let task = Process()
      task.executableURL = Bundle.main.url(forResource: "team-setup", withExtension: nil)
      task.arguments = ["status", "--json"]
      let output = Pipe()
      task.standardOutput = output
      task.standardError = FileHandle.nullDevice
      var snapshot: [String: Any]?
      do {
        try task.run()
        let data = output.fileHandleForReading.readDataToEndOfFile()
        task.waitUntilExit()
        if task.terminationStatus == 0 {
          snapshot = (try? JSONSerialization.jsonObject(with: data)) as? [String: Any]
        }
      } catch {}
      let result = snapshot
      DispatchQueue.main.async {
        self.refreshing = false
        self.refresh.isEnabled = !self.running
        guard !self.running else { return }
        guard let result = result else {
          self.heading.stringValue = "Sync status unavailable"
          self.serviceDetails.stringValue = "Could not inspect background services. Refresh to retry."
          self.syncDetails.stringValue = "Successful sync times are unavailable."
          self.syncToggle.isEnabled = false
          return
        }
        self.showStatus(result)
      }
    }
  }
  func showStatus(_ snapshot: [String: Any]) {
    let state = snapshot["state"] as? String ?? "unknown"
    let titles = ["running": "Background services are running", "paused": "Sync is paused",
                  "off": "Background sync is off", "needs_attention": "Sync needs attention",
                  "not_configured": "This Mac is not connected", "unknown": "Sync status unavailable"]
    heading.stringValue = titles[state] ?? "Sync status unavailable"
    heading.font = .systemFont(ofSize: 25, weight: .semibold)
    let labels = ["agentsview": "Local history", "analytics": "Usage analytics", "quota": "Account quota", "companion": "Session uploads"]
    let states = ["running": "Running", "loaded": "Loaded · waiting", "paused": "Paused", "off": "Stopped",
                  "not_installed": "Not installed", "error": "Exited with an error", "unknown": "Unavailable"]
    let services = snapshot["services"] as? [[String: Any]] ?? []
    serviceDetails.stringValue = services.map { service in
      let name = service["name"] as? String ?? ""
      let current = service["state"] as? String ?? "unknown"
      return "\(labels[name] ?? name): \(states[current] ?? "Unavailable")"
    }.joined(separator: "\n")
    syncDetails.stringValue = [syncSummary("Analytics", snapshot["analytics"]), syncSummary("Quota", snapshot["quota"])].joined(separator: "\n\n")
    if state == "running", [snapshot["analytics"], snapshot["quota"]].contains(where: { ($0 as? [String: Any])?["state"] as? String == "error" }) {
      heading.stringValue = "Sync needs attention"
    }
    nextSyncAction = snapshot["action"] as? String ?? ""
    syncToggle.title = nextSyncAction == "pause" ? "Pause sync" : "Resume sync"
    syncToggle.isEnabled = !running && ["pause", "resume"].contains(nextSyncAction)
  }
  func syncSummary(_ name: String, _ value: Any?) -> String {
    guard let record = value as? [String: Any] else { return "\(name): sync evidence unavailable" }
    let state = record["state"] as? String ?? "unknown"
    if state == "unknown" { return "\(name): sync evidence unavailable" }
    var text = "\(name): no successful sync recorded"
    if let last = record["last_success_at"] as? String { text = "\(name): last successful sync \(localTime(last))" }
    if name == "Analytics", state == "syncing", let uploaded = record["last_upload_at"] as? String {
      text += "\nLatest source acknowledged \(localTime(uploaded))."
    }
    if state == "error", let attempted = record["last_error_at"] as? String {
      text += "\nLast attempt failed \(localTime(attempted)). Check the connection or private logs."
    } else if state == "syncing", let attempted = record["last_attempt_at"] as? String {
      text += "\nAttempt started \(localTime(attempted)); completion not yet recorded."
    }
    return text
  }
  func localTime(_ timestamp: String) -> String {
    let parser = ISO8601DateFormatter()
    parser.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
    var date = parser.date(from: timestamp)
    if date == nil { parser.formatOptions = [.withInternetDateTime]; date = parser.date(from: timestamp) }
    guard let date = date else { return "at an unavailable time" }
    let formatter = DateFormatter()
    formatter.dateStyle = .medium
    formatter.timeStyle = .short
    return formatter.string(from: date)
  }
  @objc func toggleSync() {
    if nextSyncAction == "pause" { pauseSync() } else if nextSyncAction == "resume" { resumeSync() }
  }

  func localSettings() -> [String: Any]? {
    guard let data = try? Data(contentsOf: root.appendingPathComponent("team-agent.json")),
      let values = try? JSONSerialization.jsonObject(with: data) as? [String: Any]
    else { return nil }
    return values
  }
  @objc func openAgentsView() {
    guard let origin = localSettings()?["upstream"] as? String,
      let url = URL(string: origin), ["http", "https"].contains(url.scheme ?? ""),
      ["127.0.0.1", "localhost", "[::1]", "::1"].contains(url.host ?? ""),
      url.user == nil, url.password == nil, url.query == nil, url.fragment == nil
    else {
      status.stringValue = "Local AgentsView address is unavailable."
      return
    }
    NSWorkspace.shared.open(url)
    status.stringValue =
      "AgentsView: \(origin). If it asks for authentication, copy the local access key and paste it only into this local viewer."
  }
  @objc func copyLocalKey() {
    guard let path = localSettings()?["upstream_token_file"] as? String,
      let data = try? Data(contentsOf: URL(fileURLWithPath: path)), data.count < 4096,
      let key = String(data: data, encoding: .utf8)?.trimmingCharacters(
        in: .whitespacesAndNewlines), !key.isEmpty
    else {
      status.stringValue = "Local access key is unavailable."
      return
    }
    NSPasteboard.general.clearContents()
    NSPasteboard.general.setString(key, forType: .string)
    status.stringValue = "Local access key copied. Paste it only into AgentsView on this computer."
  }
  @objc func openWorkspace() {
    if let u = URL(string: workspace), u.scheme == "https" { NSWorkspace.shared.open(u) }
  }
  @objc func pauseSync() {
    execute(["pause"]) { ok in
      if ok {
        self.status.stringValue =
          "Local collection and upload paused. Existing central history is unchanged."
      }
    }
  }
  @objc func resumeSync() {
    execute(["resume"]) { ok in
      if ok {
        self.status.stringValue =
          "Background services started. Check the workspace for the latest sync."
      }
    }
  }
}
let app = NSApplication.shared
let delegate = AppDelegate()
app.delegate = delegate
app.run()
