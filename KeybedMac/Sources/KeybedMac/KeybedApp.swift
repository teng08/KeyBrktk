import AppKit
import ApplicationServices

final class KeybedApp: NSObject, NSApplicationDelegate, NSWindowDelegate {
    static let closeStudioNotification = Notification.Name("com.keybed.keyboard-sound.close-studio")
    private let typing = TypingActivity()
    private var hud: FloatingHUD!
    private var cards: [SoundCard] = []
    private var libraryScroll: NSScrollView!
    private var soundItems: [NSMenuItem] = []
    private var overlayItem: NSMenuItem!
    private var overlayButton: NSButton!
    private var countLabel: NSTextField!
    private var selectedSoundLabel: NSTextField!
    private var countItem: NSMenuItem!
    private var presetIndex = 0
    private var overlayEnabled = true
    private var isSmokeTest: Bool { ProcessInfo.processInfo.arguments.contains("--smoke-test") }
    private var smokeWithoutAudio: Bool { isSmokeTest && ProcessInfo.processInfo.arguments.contains("--no-audio") }
    private var audio: KeyboardAudio?
    private var keyboard: KeyboardMonitor?
    private var localMonitor: Any?
    private var permissionTimer: Timer?
    private var countTimer: Timer?
    private var displayedCount = -1
    private var statusItem: NSStatusItem!
    private var window: NSWindow!
    private var statusLabel: NSTextField!
    private var detailLabel: NSTextField!
    private var volumeLabel: NSTextField!
    private var volumeSlider: NSSlider!
    private var intensityControl: NSSegmentedControl!
    private var muteButton: NSButton!
    private var releaseButton: NSButton!
    private var startupButton: NSButton!
    private var listeningItem: NSMenuItem!
    private var muteItem: NSMenuItem!
    private var intensityItems: [NSMenuItem] = []
    private var localPressed = Set<UInt16>()
    private let defaults = UserDefaults.standard
    private var intensity = 1
    private var volume = 0.8
    private var muted = false
    private var releases = true
    private var activity: NSObjectProtocol?
    private var workspaceObservers: [NSObjectProtocol] = []
    private var fallbackMonitor: Any?
    private var fallbackPressed = Set<UInt16>()
    private var fallbackEventCount = 0
    private var globalAccessRequested = false
    private var closeObserver: NSObjectProtocol?
    private var diagnosticsURL: URL? {
        let arguments = ProcessInfo.processInfo.arguments
        guard let index = arguments.firstIndex(of: "--diagnostic-output"), index + 1 < arguments.count else { return nil }
        return URL(fileURLWithPath: arguments[index + 1])
    }

    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.accessory)
        defaults.register(defaults: ["intensity": 1, "volume": 0.8, "releases": true, "soundPreset": "alpaca", "cursorOverlay": true])
        intensity = max(0, min(2, defaults.integer(forKey: "intensity")))
        volume = max(0, min(1, defaults.double(forKey: "volume")))
        releases = defaults.bool(forKey: "releases")
        presetIndex = SoundPreset.index(for: defaults.string(forKey: "soundPreset"))
        overlayEnabled = defaults.bool(forKey: "cursorOverlay")
        if isSmokeTest { overlayEnabled = true }
        buildMenu()
        buildWindow()
        closeObserver = DistributedNotificationCenter.default().addObserver(
            forName: Self.closeStudioNotification, object: Bundle.main.bundlePath, queue: .main) { [weak self] _ in
                self?.window.performClose(nil)
            }
        hud = FloatingHUD(typing: typing)
        hud.onCountChange = { [weak self] count in self?.updateCount(count) }
        do {
            guard let resources = Bundle.main.resourceURL else { throw KeybedError(message: "The app's sound recordings are missing. Rebuild Keybed.") }
            audio = try KeyboardAudio(resourceURL: resources.appendingPathComponent("Sounds"))
            applySettings()
            if !smokeWithoutAudio { try audio?.start() }
            keyboard = KeyboardMonitor(audio: audio!, typing: typing)
            keyboard?.setForeground(NSApp.isActive)
            if !isSmokeTest { keyboard?.start() }
            refreshGlobalFallback()
            activity = ProcessInfo.processInfo.beginActivity(options: .userInitiatedAllowingIdleSystemSleep,
                reason: "Play keyboard sounds while typing in other apps")
            localMonitor = NSEvent.addLocalMonitorForEvents(matching: [.keyDown, .keyUp]) { [weak self] event in
                self?.processLocalEvent(event) ?? event
            }
            countTimer = Timer(timeInterval: 0.1, repeats: true) { [weak self] _ in
                guard let self else { return }
                self.updateCount(self.typing.snapshot.count)
            }
            if let countTimer { RunLoop.main.add(countTimer, forMode: .common) }
            permissionTimer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] _ in
                guard let self else { return }
                self.keepBackgroundServicesRunning()
                self.updateStatus()
            }
            if let permissionTimer { RunLoop.main.add(permissionTimer, forMode: .common) }
            let center = NSWorkspace.shared.notificationCenter
            for name in [NSWorkspace.didWakeNotification, NSWorkspace.sessionDidBecomeActiveNotification] {
                workspaceObservers.append(center.addObserver(forName: name, object: nil, queue: .main) { [weak self] _ in
                    guard let self else { return }
                    self.keepBackgroundServicesRunning()
                    self.updateStatus()
                })
            }
        } catch {
            if ProcessInfo.processInfo.arguments.contains("--smoke-test") {
                fputs("Keybed startup failed: \(error.localizedDescription)\n", stderr)
                exit(1)
            }
            let alert = NSAlert()
            alert.messageText = "Keybed couldn't start its sound engine"
            alert.informativeText = error.localizedDescription
            alert.runModal()
        }
        updateStatus()
        if !ProcessInfo.processInfo.arguments.contains("--background") || !KeyboardMonitor.hasGlobalPermission {
            if isSmokeTest { window.orderFront(nil) } else { showWindow() }
        }
        if !KeyboardMonitor.hasGlobalPermission && !ProcessInfo.processInfo.arguments.contains("--smoke-test") {
            requestGlobalAccess()
        }
        if ProcessInfo.processInfo.arguments.contains("--smoke-test") {
            DispatchQueue.main.asyncAfter(deadline: .now() + 2) {
                guard self.audio != nil, (self.smokeWithoutAudio || self.audio?.isRunning == true),
                      let content = self.window.contentView,
                      self.cards.count == SoundPreset.all.count,
                      let library = self.libraryScroll.documentView,
                      self.cards.allSatisfy({ library.bounds.contains($0.frame) }),
                      content.subviews.allSatisfy({ view in
                          let rect = view.frame
                          return rect.minY >= 8 && rect.minX >= 8 && rect.maxX <= content.bounds.width - 8 && rect.maxY <= content.bounds.height - 8
                      }),
                      self.hud.panel.ignoresMouseEvents,
                      !self.hud.panel.canBecomeKey, !self.hud.panel.canBecomeMain else {
                    fputs("Keybed: startup, visible window layout or cursor panel check failed.\n", stderr)
                    exit(1)
                }
                let previousCount = self.typing.snapshot.count
                for (index, card) in self.cards.enumerated() {
                    guard !self.cards.enumerated().contains(where: { $0.offset != index && $0.element.frame.intersects(card.frame) }) else {
                        fputs("Keybed: sound cards overlap.\n", stderr)
                        exit(1)
                    }
                    _ = library.scrollToVisible(card.frame)
                    self.libraryScroll.reflectScrolledClipView(self.libraryScroll.contentView)
                    self.selectSound(card)
                    guard self.libraryScroll.contentView.documentVisibleRect.contains(card.frame),
                          self.presetIndex == index, self.hud.view.preset.id == card.preset.id,
                          self.soundItems[index].state == .on,
                          self.typing.snapshot.count == previousCount else {
                        fputs("Keybed: scrolling/selection check failed for \(card.preset.name): card=\(card.frame), visible=\(self.libraryScroll.contentView.documentVisibleRect), insets=\(self.libraryScroll.contentInsets), selected=\(self.presetIndex), HUD=\(self.hud.view.preset.id), menu=\(self.soundItems[index].state.rawValue), count=\(self.typing.snapshot.count), expectedCount=\(previousCount).\n", stderr)
                        exit(1)
                    }
                }
                let smokePreset = SoundPreset.index(for: "turquoise")
                self.selectSound(self.cards[smokePreset])
                self.libraryScroll.contentView.scroll(to: .zero)
                self.libraryScroll.reflectScrolledClipView(self.libraryScroll.contentView)
                let screen = NSRect(x: -1_920, y: -180, width: 1_920, height: 1_080)
                for cursor in [NSPoint(x: -1_920, y: -180), NSPoint(x: 0, y: 900)] {
                    let origin = FloatingHUD.origin(cursor: cursor, visibleFrame: screen, size: self.hud.panel.frame.size)
                    guard screen.contains(NSRect(origin: origin, size: self.hud.panel.frame.size)) else {
                        fputs("Keybed: cursor panel screen bounds check failed.\n", stderr)
                        exit(1)
                    }
                }
                // Exercise the AppKit monitor handler with native event objects.
                // Keep fixtures independent of whichever app the user is typing in.
                for (type, repeatKey) in [(NSEvent.EventType.keyDown, false), (.keyDown, true), (.keyUp, false)] {
                    guard let event = NSEvent.keyEvent(with: type, location: .zero, modifierFlags: [], timestamp: 0,
                        windowNumber: self.window.windowNumber, context: nil, characters: "a", charactersIgnoringModifiers: "a",
                        isARepeat: repeatKey, keyCode: 0) else { exit(1) }
                    _ = self.processLocalEvent(event)
                }
                DispatchQueue.main.asyncAfter(deadline: .now() + 0.25) {
                    guard self.typing.snapshot.count == previousCount + 2 else {
                        fputs("Keybed: key event count failed: actual=\(self.typing.snapshot.count), expected=\(previousCount + 2).\n", stderr)
                        exit(1)
                    }
                    self.captureSnapshot(self.window.contentView!, argument: "--snapshot-output")
                    self.captureSnapshot(self.hud.view, argument: "--hud-snapshot-output")
                    DispatchQueue.main.asyncAfter(deadline: .now() + 3.5) {
                        guard self.typing.snapshot.count == 0, self.countLabel.stringValue == "0",
                              self.hud.view.count == 0, self.hud.panel.frame.height <= 37,
                              self.hud.view.preset.id == "turquoise" else {
                            fputs("Keybed: timed reset/collapse failed: count=\(self.typing.snapshot.count), label=\(self.countLabel.stringValue), HUD=\(self.hud.view.count), height=\(self.hud.panel.frame.height), collapse=\(self.hud.view.collapse), sound=\(self.hud.view.preset.id).\n", stderr)
                            self.captureSnapshot(self.hud.view, argument: "--collapsed-hud-snapshot-output")
                            exit(1)
                        }
                        self.captureSnapshot(self.hud.view, argument: "--collapsed-hud-snapshot-output")
                        self.overlayEnabled = false
                        self.applySettings()
                        guard !self.hud.panel.isVisible else { exit(1) }
                        self.typing.reset()
                        self.updateCount(0)
                        guard self.typing.snapshot.count == 0, self.countLabel.stringValue == "0" else { exit(1) }
                        self.window.performClose(nil)
                        DispatchQueue.main.asyncAfter(deadline: .now() + 1.2) {
                            guard !self.window.isVisible, (self.smokeWithoutAudio || self.audio?.isRunning == true),
                                  self.permissionTimer?.isValid == true, self.countTimer?.isValid == true,
                                  self.localMonitor != nil, self.activity != nil,
                                  !self.applicationShouldTerminateAfterLastWindowClosed(NSApp) else {
                                fputs("Keybed: closing the studio stopped background services.\n", stderr)
                                exit(1)
                            }
                            self.testSound(nil)
                            self.window.orderFront(nil)
                            guard self.window.isVisible else { exit(1) }
                            print("PASS: \(SoundPreset.all.count) selectable sounds, scrolling, counting, idle collapse and overlay controls; close/reopen keeps keyboard handler, timers and menu bar running.")
                            print(self.smokeWithoutAudio ? "NOTE: smoke test intentionally disables audio output; PCM is verified separately." : "PASS: audio engine stays running after closing and reopening the studio.")
                            NSApp.terminate(nil)
                        }
                    }
                }
            }
        }
    }

    func applicationWillTerminate(_ notification: Notification) {
        hud?.configure(enabled: false, preset: SoundPreset.all[presetIndex], muted: muted)
        permissionTimer?.invalidate()
        countTimer?.invalidate()
        keyboard?.stop()
        if let localMonitor { NSEvent.removeMonitor(localMonitor) }
        if let fallbackMonitor { NSEvent.removeMonitor(fallbackMonitor) }
        if let activity { ProcessInfo.processInfo.endActivity(activity) }
        workspaceObservers.forEach { NSWorkspace.shared.notificationCenter.removeObserver($0) }
        if let closeObserver { DistributedNotificationCenter.default().removeObserver(closeObserver) }
    }

    func applicationShouldTerminateAfterLastWindowClosed(_ sender: NSApplication) -> Bool { false }

    // The red close button hides only the studio. Services belong to the app,
    // and are stopped solely by applicationWillTerminate after an explicit Quit.
    func windowShouldClose(_ sender: NSWindow) -> Bool {
        guard sender === window else { return true }
        sender.orderOut(nil)
        keepBackgroundServicesRunning()
        updateStatus()
        if !isSmokeTest { NSApp.deactivate() }
        return false
    }

    private func keepBackgroundServicesRunning() {
        if !smokeWithoutAudio && audio?.isRunning != true { try? audio?.start() }
        if !isSmokeTest && keyboard?.isListening != true { keyboard?.start() }
        refreshGlobalFallback()
    }

    func applicationDidBecomeActive(_ notification: Notification) { keyboard?.setForeground(true) }
    func applicationDidResignActive(_ notification: Notification) {
        keyboard?.setForeground(false)
        localPressed.removeAll()
        keepBackgroundServicesRunning()
    }

    private var globalListening: Bool {
        keyboard?.isListening == true || (fallbackMonitor != nil && AXIsProcessTrusted())
    }

    private func processLocalEvent(_ event: NSEvent) -> NSEvent {
        guard keyboard?.isListening != true else { return event }
        if isSmokeTest && event.timestamp != 0 { return event }
        if event.type == .keyDown {
            localPressed.insert(event.keyCode)
            audio?.play(keyCode: event.keyCode)
            typing.record()
        } else if event.type == .keyUp && localPressed.remove(event.keyCode) != nil {
            audio?.play(keyCode: event.keyCode, release: true)
        }
        return event
    }

    private func refreshGlobalFallback() {
        guard !isSmokeTest else { return }
        if AXIsProcessTrusted() && fallbackMonitor == nil {
            fallbackMonitor = NSEvent.addGlobalMonitorForEvents(matching: [.keyDown, .keyUp]) { [weak self] event in
                guard let self, self.keyboard?.isListening != true else { return }
                self.fallbackEventCount += 1
                if event.type == .keyDown {
                    self.fallbackPressed.insert(event.keyCode)
                    self.audio?.play(keyCode: event.keyCode)
                    self.typing.record()
                } else if self.fallbackPressed.remove(event.keyCode) != nil {
                    self.audio?.play(keyCode: event.keyCode, release: true)
                }
            }
        } else if !AXIsProcessTrusted(), let fallbackMonitor {
            NSEvent.removeMonitor(fallbackMonitor)
            self.fallbackMonitor = nil
            fallbackPressed.removeAll()
        }
    }

    private func requestGlobalAccess() {
        guard !globalAccessRequested else { return }
        globalAccessRequested = true
        _ = CGRequestListenEventAccess()
        keyboard?.start()
        updateStatus()
    }

    private func writeDiagnostics() {
        guard let diagnosticsURL else { return }
        let counts = keyboard?.counts ?? (total: 0, background: 0)
        let snapshot: [String: Any] = [
            "bundlePath": Bundle.main.bundlePath,
            "inputMonitoring": CGPreflightListenEventAccess(),
            "accessibility": AXIsProcessTrusted(),
            "eventTapEnabled": keyboard?.tapEnabled ?? false,
            "globalListening": globalListening,
            "globalEvents": counts.total + fallbackEventCount,
            "backgroundEvents": counts.background + fallbackEventCount,
            "audioRunning": audio?.isRunning ?? false,
            "active": NSApp.isActive,
            "muted": muted,
            "volume": volume,
            "launchAtLogin": LoginStartup.isEnabled,
            "selectedSound": SoundPreset.all[presetIndex].id,
            "cursorOverlay": overlayEnabled,
            "typedKeys": typing.snapshot.count,
            "studioVisible": window?.isVisible ?? false,
            "permissionTimerRunning": permissionTimer?.isValid ?? false,
            "countTimerRunning": countTimer?.isValid ?? false,
            "processID": ProcessInfo.processInfo.processIdentifier
        ]
        if let data = try? JSONSerialization.data(withJSONObject: snapshot, options: [.prettyPrinted, .sortedKeys]) {
            try? data.write(to: diagnosticsURL, options: .atomic)
        }
    }

    func applicationShouldHandleReopen(_ sender: NSApplication, hasVisibleWindows flag: Bool) -> Bool {
        showWindow()
        return true
    }

    private func label(_ text: String, frame: NSRect, size: CGFloat = 12, weight: NSFont.Weight = .regular,
                       color: NSColor = NSColor(white: 0.66, alpha: 1)) -> NSTextField {
        let label = NSTextField(labelWithString: text)
        label.frame = frame
        label.font = .systemFont(ofSize: size, weight: weight)
        label.textColor = color
        window.contentView?.addSubview(label)
        return label
    }

    private func buildWindow() {
        let studioWindow = StudioWindow(contentRect: NSRect(x: 0, y: 0, width: 760, height: 714),
            styleMask: [.titled, .closable, .miniaturizable], backing: .buffered, defer: false)
        studioWindow.allowsInput = !isSmokeTest
        window = studioWindow
        window.delegate = self
        window.ignoresMouseEvents = isSmokeTest
        window.title = "Keybed · Sound Studio"
        window.appearance = NSAppearance(named: .darkAqua)
        window.backgroundColor = NSColor(srgbRed: 0.065, green: 0.075, blue: 0.095, alpha: 1)
        window.center()
        window.isReleasedWhenClosed = false
        let root = StudioView(frame: NSRect(x: 0, y: 0, width: 760, height: 714))
        window.contentView = root
        let accent = SoundPreset.all[0].color
        _ = label("K E Y B E D   /   S O U N D   S T U D I O", frame: NSRect(x: 28, y: 22, width: 430, height: 16),
                  size: 10, weight: .bold, color: accent)
        _ = label("Find your typing rhythm.", frame: NSRect(x: 28, y: 45, width: 510, height: 36),
                  size: 28, weight: .bold, color: .white)
        _ = label("\(SoundPreset.all.count) sounds. Instant feedback. A little joy in every key.", frame: NSRect(x: 28, y: 85, width: 520, height: 19))
        _ = label("3-SECOND COUNT", frame: NSRect(x: 581, y: 27, width: 148, height: 16), size: 9, weight: .bold)
        countLabel = label("0", frame: NSRect(x: 578, y: 46, width: 152, height: 35), size: 28, weight: .semibold, color: .white)
        countLabel.font = .monospacedDigitSystemFont(ofSize: 28, weight: .semibold)
        _ = label("SOUND LIBRARY · SCROLL", frame: NSRect(x: 28, y: 116, width: 140, height: 16), size: 10, weight: .bold)
        selectedSoundLabel = label("", frame: NSRect(x: 175, y: 113, width: 260, height: 20), size: 11, weight: .medium, color: accent)
        let tryField = NSTextField(frame: NSRect(x: 462, y: 108, width: 270, height: 25))
        tryField.placeholderString = "Type here to try your sound…"
        tryField.font = .systemFont(ofSize: 11)
        root.addSubview(tryField)
        // Keep the new switches visible without growing the studio off laptop screens.
        let featuredIndex = SoundPreset.index(for: "skibiddy")
        let newIndices = SoundPreset.all.indices.filter { ["holypanda", "cream", "turquoise"].contains(SoundPreset.all[$0].id) }
        let gridOrder = newIndices + SoundPreset.all.indices.filter { $0 != featuredIndex && !newIndices.contains($0) }
        let gridPositions = Dictionary(uniqueKeysWithValues: gridOrder.enumerated().map { ($0.element, $0.offset) })
        let rows = (gridOrder.count + 2) / 3
        let library = SoundLibraryView(frame: NSRect(x: 0, y: 0, width: 704, height: 70 + rows * 70 - 8),
                                       featuredIndex: featuredIndex, gridOrder: gridOrder)
        libraryScroll = NSScrollView(frame: NSRect(x: 28, y: 142, width: 704, height: 276))
        libraryScroll.hasVerticalScroller = true
        libraryScroll.autohidesScrollers = true
        libraryScroll.scrollerStyle = .overlay
        libraryScroll.drawsBackground = false
        libraryScroll.documentView = library
        root.addSubview(libraryScroll)
        libraryScroll.tile()
        library.setFrameSize(NSSize(width: libraryScroll.contentSize.width, height: library.frame.height))
        for (index, preset) in SoundPreset.all.enumerated() {
            let position = gridPositions[index] ?? 0
            let featured = index == featuredIndex
            let cardFrame = featured
                ? NSRect(x: 0, y: 0, width: 704, height: 62)
                : NSRect(x: (position % 3) * 238, y: 70 + (position / 3) * 70, width: 228, height: 62)
            let card = SoundCard(preset: preset, index: index, frame: cardFrame,
                target: self, action: #selector(selectSound(_:)))
            cards.append(card)
            library.addSubview(card)
        }
        library.layoutCards()
        _ = label("INTENSITY", frame: NSRect(x: 28, y: 443, width: 100, height: 18), size: 10, weight: .bold)
        intensityControl = NSSegmentedControl(labels: KeyboardAudio.intensityNames, trackingMode: .selectOne,
            target: self, action: #selector(changeIntensity(_:)))
        intensityControl.frame = NSRect(x: 127, y: 436, width: 294, height: 28)
        root.addSubview(intensityControl)
        volumeLabel = label("", frame: NSRect(x: 449, y: 443, width: 102, height: 18), size: 11, weight: .medium)
        volumeSlider = NSSlider(value: volume, minValue: 0, maxValue: 1, target: self, action: #selector(changeVolume(_:)))
        volumeSlider.frame = NSRect(x: 548, y: 436, width: 183, height: 28)
        volumeSlider.isContinuous = true
        root.addSubview(volumeSlider)
        _ = label("Count resets every 3 seconds. The floating counter collapses when you pause.", frame: NSRect(x: 28, y: 475, width: 590, height: 18), size: 11)
        releaseButton = NSButton(checkboxWithTitle: "Play key release sounds", target: self, action: #selector(changeReleases(_:)))
        releaseButton.frame = NSRect(x: 28, y: 507, width: 310, height: 24)
        releaseButton.state = releases ? .on : .off
        root.addSubview(releaseButton)
        startupButton = NSButton(checkboxWithTitle: "Start automatically when I log in", target: self, action: #selector(changeStartup(_:)))
        startupButton.frame = NSRect(x: 380, y: 507, width: 350, height: 24)
        startupButton.state = LoginStartup.isEnabled ? .on : .off
        root.addSubview(startupButton)
        overlayButton = NSButton(checkboxWithTitle: "Floating counter follows my mouse", target: self, action: #selector(changeOverlay(_:)))
        overlayButton.frame = NSRect(x: 28, y: 541, width: 350, height: 24)
        root.addSubview(overlayButton)
        let reset = NSButton(title: "Reset count", target: self, action: #selector(resetCount(_:)))
        reset.bezelStyle = .rounded
        reset.frame = NSRect(x: 618, y: 539, width: 116, height: 28)
        root.addSubview(reset)
        statusLabel = label("Starting…", frame: NSRect(x: 28, y: 584, width: 704, height: 20), size: 12, weight: .semibold, color: accent)
        detailLabel = NSTextField(wrappingLabelWithString: "")
        detailLabel.frame = NSRect(x: 28, y: 608, width: 704, height: 31)
        detailLabel.font = .systemFont(ofSize: 11)
        detailLabel.textColor = NSColor(white: 0.66, alpha: 1)
        root.addSubview(detailLabel)
        let test = NSButton(title: "▶  Preview sound", target: self, action: #selector(testSound(_:)))
        test.bezelStyle = .rounded
        test.frame = NSRect(x: 24, y: 651, width: 150, height: 30)
        root.addSubview(test)
        muteButton = NSButton(title: "Mute", target: self, action: #selector(toggleMute(_:)))
        muteButton.bezelStyle = .rounded
        muteButton.frame = NSRect(x: 179, y: 651, width: 89, height: 30)
        root.addSubview(muteButton)
        let permission = NSButton(title: "Enable keyboard access…", target: self, action: #selector(openPermissions(_:)))
        permission.bezelStyle = .rounded
        permission.frame = NSRect(x: 530, y: 651, width: 207, height: 30)
        root.addSubview(permission)
        _ = label("Close this window to keep the sounds and counter running in your menu bar.",
                  frame: NSRect(x: 28, y: 690, width: 704, height: 16), size: 10)
    }

    private func buildMenu() {
        let mainMenu = NSMenu()
        let appItem = NSMenuItem()
        let appMenu = NSMenu()
        appMenu.addItem(NSMenuItem(title: "Show Keybed", action: #selector(showWindowAction(_:)), keyEquivalent: ""))
        appMenu.addItem(NSMenuItem(title: "Keep running in menu bar", action: #selector(hideStudio(_:)), keyEquivalent: "w"))
        appMenu.addItem(NSMenuItem(title: "Quit Keybed", action: #selector(quit(_:)), keyEquivalent: "q"))
        appItem.submenu = appMenu
        mainMenu.addItem(appItem)
        NSApp.mainMenu = mainMenu
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        statusItem.button?.image = NSImage(systemSymbolName: "keyboard", accessibilityDescription: "Keybed")
        statusItem.button?.isEnabled = !isSmokeTest
        let menu = NSMenu()
        menu.addItem(NSMenuItem(title: "Show Keybed", action: #selector(showWindowAction(_:)), keyEquivalent: ""))
        menu.addItem(NSMenuItem(title: "Keep running in menu bar", action: #selector(hideStudio(_:)), keyEquivalent: ""))
        listeningItem = NSMenuItem(title: "Starting…", action: nil, keyEquivalent: "")
        menu.addItem(listeningItem)
        countItem = NSMenuItem(title: "0 keys · 3-second window", action: nil, keyEquivalent: "")
        menu.addItem(countItem)
        menu.addItem(.separator())
        let library = NSMenuItem(title: "Sound", action: nil, keyEquivalent: "")
        let soundMenu = NSMenu()
        for (index, preset) in SoundPreset.all.enumerated() {
            let item = NSMenuItem(title: preset.name, action: #selector(selectSound(_:)), keyEquivalent: "")
            item.tag = index
            item.target = self
            soundItems.append(item)
            soundMenu.addItem(item)
        }
        library.submenu = soundMenu
        menu.addItem(library)
        for (index, name) in KeyboardAudio.intensityNames.enumerated() {
            let item = NSMenuItem(title: name, action: #selector(selectIntensity(_:)), keyEquivalent: "")
            item.tag = index
            item.state = index == intensity ? .on : .off
            intensityItems.append(item)
            menu.addItem(item)
        }
        menu.addItem(.separator())
        overlayItem = NSMenuItem(title: "Floating cursor counter", action: #selector(toggleOverlay(_:)), keyEquivalent: "")
        menu.addItem(overlayItem)
        menu.addItem(NSMenuItem(title: "Reset typing count", action: #selector(resetCount(_:)), keyEquivalent: ""))
        menu.addItem(NSMenuItem(title: "Test sound", action: #selector(testSound(_:)), keyEquivalent: ""))
        muteItem = NSMenuItem(title: "Mute", action: #selector(toggleMute(_:)), keyEquivalent: "")
        menu.addItem(muteItem)
        menu.addItem(NSMenuItem(title: "Enable keyboard access…", action: #selector(openPermissions(_:)), keyEquivalent: ""))
        menu.addItem(NSMenuItem(title: "Quit Keybed", action: #selector(quit(_:)), keyEquivalent: "q"))
        statusItem.menu = menu
        for item in menu.items + appMenu.items { item.target = self }
    }

    private func applySettings() {
        audio?.configure(preset: presetIndex, intensity: intensity, volume: volume, muted: muted, releases: releases)
        if !isSmokeTest {
            defaults.set(intensity, forKey: "intensity")
            defaults.set(volume, forKey: "volume")
            defaults.set(releases, forKey: "releases")
            defaults.set(SoundPreset.all[presetIndex].id, forKey: "soundPreset")
            defaults.set(overlayEnabled, forKey: "cursorOverlay")
        }
        cards.forEach { $0.selected = $0.tag == presetIndex }
        soundItems.forEach { $0.state = $0.tag == presetIndex ? .on : .off }
        selectedSoundLabel?.stringValue = "Now playing · " + SoundPreset.all[presetIndex].name
        selectedSoundLabel?.textColor = SoundPreset.all[presetIndex].color
        overlayButton?.state = overlayEnabled ? .on : .off
        overlayItem?.state = overlayEnabled ? .on : .off
        hud?.configure(enabled: overlayEnabled, preset: SoundPreset.all[presetIndex], muted: muted)
        intensityItems.forEach { $0.state = $0.tag == intensity ? .on : .off }
        intensityControl?.selectedSegment = intensity
        volumeLabel?.stringValue = "Volume  \(Int(volume * 100))%"
        muteButton?.title = muted ? "Unmute" : "Mute"
        muteItem?.title = muted ? "Unmute" : "Mute"
        updateStatus()
    }

    private func updateStatus() {
        guard statusLabel != nil else { return }
        let text: String
        if smokeWithoutAudio {
            text = "Smoke test — audio output disabled"
            detailLabel.stringValue = "Native controls and keyboard handlers are tested without requiring an audio device."
        } else if audio == nil || audio?.lastError != nil || audio?.isRunning != true {
            text = "Sound engine needs attention"
            detailLabel.stringValue = audio?.lastError ?? "Quit and reopen Keybed to retry."
        } else if globalListening {
            text = muted ? "Muted" : "Ready — listening in all apps"
            detailLabel.stringValue = "\(SoundPreset.all[presetIndex].name) is ready. Your counter follows the mouse in other apps, too."
        } else {
            text = "Keyboard access needed"
            let settingsPath = ProcessInfo.processInfo.operatingSystemVersion.majorVersion >= 13
                ? "System Settings → Privacy & Security"
                : "System Preferences → Security & Privacy → Privacy"
            detailLabel.stringValue = "Sounds work only inside this window until Keybed is enabled in \(settingsPath) → Input Monitoring. Use + to add /Applications/Keybed.app."
        }
        statusLabel.stringValue = text
        listeningItem.title = text
        statusItem.button?.appearsDisabled = muted
        statusItem.button?.contentTintColor = globalListening ? nil : SoundPreset.all[0].color
        statusItem.button?.toolTip = "Keybed · " + text
        writeDiagnostics()
    }

    private func showWindow() {
        NSApp.activate(ignoringOtherApps: true)
        window.makeKeyAndOrderFront(nil)
    }

    private func captureSnapshot(_ view: NSView, argument: String) {
        let arguments = ProcessInfo.processInfo.arguments
        guard let index = arguments.firstIndex(of: argument), index + 1 < arguments.count else { return }
        view.layoutSubtreeIfNeeded()
        guard let bitmap = view.bitmapImageRepForCachingDisplay(in: view.bounds) else { exit(1) }
        view.cacheDisplay(in: view.bounds, to: bitmap)
        guard let data = bitmap.representation(using: .png, properties: [:]) else { exit(1) }
        do { try data.write(to: URL(fileURLWithPath: arguments[index + 1])) } catch { exit(1) }
    }

    private func updateCount(_ count: Int) {
        guard count != displayedCount else { return }
        displayedCount = count
        let number = NumberFormatter.localizedString(from: NSNumber(value: count), number: .decimal)
        countLabel?.stringValue = number
        countItem?.title = number + " keys · 3-second window"
    }
    @objc private func selectSound(_ sender: AnyObject) {
        guard let tag = (sender as? NSControl)?.tag ?? (sender as? NSMenuItem)?.tag,
              SoundPreset.all.indices.contains(tag) else { return }
        presetIndex = tag
        applySettings()
        testSound(nil)
    }
    @objc private func changeOverlay(_ sender: NSButton) { overlayEnabled = sender.state == .on; applySettings() }
    @objc private func toggleOverlay(_ sender: Any?) { overlayEnabled.toggle(); applySettings() }
    @objc private func resetCount(_ sender: Any?) { typing.reset(); updateCount(0) }
    @objc private func showWindowAction(_ sender: Any?) { showWindow() }
    @objc private func hideStudio(_ sender: Any?) { window.performClose(nil) }
    @objc private func changeIntensity(_ sender: NSSegmentedControl) { intensity = sender.selectedSegment; applySettings() }
    @objc private func selectIntensity(_ sender: NSMenuItem) { intensity = sender.tag; applySettings() }
    @objc private func changeVolume(_ sender: NSSlider) { volume = sender.doubleValue; applySettings() }
    @objc private func changeReleases(_ sender: NSButton) { releases = sender.state == .on; applySettings() }
    @objc private func changeStartup(_ sender: NSButton) {
        do { try LoginStartup.setEnabled(sender.state == .on) }
        catch {
            sender.state = LoginStartup.isEnabled ? .on : .off
            let alert = NSAlert()
            alert.messageText = "Could not change login startup"
            alert.informativeText = error.localizedDescription
            alert.runModal()
        }
    }
    @objc private func toggleMute(_ sender: Any?) { muted.toggle(); applySettings() }
    @objc private func testSound(_ sender: Any?) {
        if !smokeWithoutAudio && audio?.isRunning != true { try? audio?.start() }
        audio?.play(keyCode: 0)
        hud?.preview()
    }
    @objc private func openPermissions(_ sender: Any?) {
        requestGlobalAccess()
        // Reveal the exact copy to add instead of sending users to a stale entry.
        NSWorkspace.shared.activateFileViewerSelecting([Bundle.main.bundleURL])
        NSWorkspace.shared.open(URL(string: "x-apple.systempreferences:com.apple.preference.security?Privacy_ListenEvent")!)
        keyboard?.start()
    }
    @objc private func quit(_ sender: Any?) { NSApp.terminate(nil) }
}
