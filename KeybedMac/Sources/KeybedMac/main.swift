import AppKit
import AVFoundation
import ApplicationServices
import AudioMixer

struct KeybedError: LocalizedError {
    let message: String
    var errorDescription: String? { message }
}

enum LoginStartup {
    static var agentURL: URL {
        FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent("Library/LaunchAgents/com.keybed.keyboard-sound.login.plist")
    }
    static var isEnabled: Bool { FileManager.default.fileExists(atPath: agentURL.path) }

    static func setEnabled(_ enabled: Bool) throws {
        if !enabled {
            if isEnabled { try FileManager.default.removeItem(at: agentURL) }
            return
        }
        let appURL = Bundle.main.bundleURL
        guard appURL.pathExtension == "app" else {
            throw KeybedError(message: "Open Keybed.app to configure startup.")
        }
        let plist: [String: Any] = [
            "Label": "com.keybed.keyboard-sound.login",
            "ProgramArguments": ["/usr/bin/open", "-g", appURL.path, "--args", "--background"],
            "RunAtLoad": true,
            "LimitLoadToSessionType": "Aqua"
        ]
        let data = try PropertyListSerialization.data(fromPropertyList: plist, format: .xml, options: 0)
        try FileManager.default.createDirectory(at: agentURL.deletingLastPathComponent(), withIntermediateDirectories: true)
        try data.write(to: agentURL, options: .atomic)
    }
}

// A dedicated event run loop keeps UI work off the keyboard-to-audio path.
final class KeyboardMonitor {
    private let audio: KeyboardAudio
    private let typing: TypingActivity
    private let lock = NSLock()
    private var tap: CFMachPort?
    private var runLoop: CFRunLoop?
    private var installing = false
    private var stopping = false
    private var pressed = Set<UInt16>()
    private var foreground = false
    private var eventCount = 0
    private var backgroundCount = 0

    init(audio: KeyboardAudio, typing: TypingActivity) { self.audio = audio; self.typing = typing }

    static var hasGlobalPermission: Bool { CGPreflightListenEventAccess() || AXIsProcessTrusted() }

    var tapEnabled: Bool {
        lock.lock()
        defer { lock.unlock() }
        return tap.map { CGEvent.tapIsEnabled(tap: $0) } ?? false
    }

    // A process-local tap can open without permission to observe other apps.
    var isListening: Bool { Self.hasGlobalPermission && tapEnabled }

    var counts: (total: Int, background: Int) {
        lock.lock()
        defer { lock.unlock() }
        return (eventCount, backgroundCount)
    }

    func setForeground(_ value: Bool) {
        lock.lock()
        foreground = value
        lock.unlock()
    }

    private func recordEvent() {
        lock.lock()
        eventCount += 1
        if !foreground { backgroundCount += 1 }
        lock.unlock()
    }

    func start() {
        guard Self.hasGlobalPermission else { return }
        lock.lock()
        guard !installing, !stopping else { lock.unlock(); return }
        if let tap {
            CGEvent.tapEnable(tap: tap, enable: true)
            lock.unlock()
            return
        }
        installing = true
        lock.unlock()
        let thread = Thread { [self] in install() }
        thread.name = "Keybed keyboard events"
        thread.qualityOfService = .userInteractive
        thread.start()
    }

    private func install() {
        let mask = CGEventMask((1 << CGEventType.keyDown.rawValue) | (1 << CGEventType.keyUp.rawValue))
        let newTap = CGEvent.tapCreate(tap: .cgSessionEventTap, place: .headInsertEventTap, options: .listenOnly,
            eventsOfInterest: mask, callback: { _, type, event, info in
                guard let info else { return Unmanaged.passUnretained(event) }
                let monitor = Unmanaged<KeyboardMonitor>.fromOpaque(info).takeUnretainedValue()
                if type == .tapDisabledByTimeout || type == .tapDisabledByUserInput {
                    monitor.reenable()
                } else if type == .keyDown || type == .keyUp {
                    monitor.recordEvent()
                    let code = UInt16(event.getIntegerValueField(.keyboardEventKeycode))
                    if type == .keyDown {
                        // macOS sends more key-down events while a key is held.
                        // Sound and count only the physical down/up cycle once.
                        if monitor.pressed.insert(code).inserted {
                            monitor.audio.play(keyCode: code)
                            monitor.typing.record()
                        }
                    } else if monitor.pressed.remove(code) != nil {
                        monitor.audio.play(keyCode: code, release: true)
                    }
                }
                return Unmanaged.passUnretained(event)
            }, userInfo: Unmanaged.passUnretained(self).toOpaque())
        lock.lock()
        installing = false
        guard let newTap, !stopping else {
            lock.unlock()
            if let newTap { CFMachPortInvalidate(newTap) }
            return
        }
        tap = newTap
        let loop = CFRunLoopGetCurrent()!
        runLoop = loop
        lock.unlock()
        let source = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, newTap, 0)!
        CFRunLoopAddSource(loop, source, .commonModes)
        CGEvent.tapEnable(tap: newTap, enable: true)
        // Check stopping even if stop() arrived before entering the run loop.
        while true {
            lock.lock()
            let shouldStop = stopping
            lock.unlock()
            if shouldStop { break }
            CFRunLoopRunInMode(.defaultMode, 0.5, false)
        }
        CFRunLoopRemoveSource(loop, source, .commonModes)
        CFMachPortInvalidate(newTap)
        lock.lock()
        tap = nil
        runLoop = nil
        lock.unlock()
    }

    private func reenable() {
        lock.lock()
        // A disabled tap can miss key-up, so do not leave a key stuck as held.
        pressed.removeAll()
        if let tap, !stopping { CGEvent.tapEnable(tap: tap, enable: true) }
        lock.unlock()
    }

    func stop() {
        lock.lock()
        stopping = true
        if let runLoop { CFRunLoopStop(runLoop) }
        lock.unlock()
    }
}

// Draw the app icon from vector geometry during packaging.
private func makeIcon(in directory: URL) throws {
    try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
    let entries: [(String, Int)] = [("16x16", 16), ("16x16@2x", 32), ("32x32", 32), ("32x32@2x", 64),
        ("128x128", 128), ("128x128@2x", 256), ("256x256", 256), ("256x256@2x", 512),
        ("512x512", 512), ("512x512@2x", 1024)]
    for (name, size) in entries {
        guard let bitmap = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: size, pixelsHigh: size,
            bitsPerSample: 8, samplesPerPixel: 4, hasAlpha: true, isPlanar: false,
            colorSpaceName: .deviceRGB, bytesPerRow: size * 4, bitsPerPixel: 32),
              let context = CGContext(data: bitmap.bitmapData, width: size, height: size, bitsPerComponent: 8,
                bytesPerRow: size * 4, space: CGColorSpaceCreateDeviceRGB(),
                bitmapInfo: CGImageAlphaInfo.premultipliedLast.rawValue) else {
            throw KeybedError(message: "Could not draw the app icon.")
        }
        context.scaleBy(x: CGFloat(size) / 1024, y: CGFloat(size) / 1024)
        func fill(_ rect: CGRect, radius: CGFloat, color: CGColor) {
            context.setFillColor(color)
            context.addPath(CGPath(roundedRect: rect, cornerWidth: radius, cornerHeight: radius, transform: nil))
            context.fillPath()
        }
        let dark = CGColor(red: 0.12, green: 0.11, blue: 0.16, alpha: 1)
        let light = CGColor(red: 0.91, green: 0.89, blue: 0.84, alpha: 1)
        let accent = CGColor(red: 1, green: 0.40, blue: 0.20, alpha: 1)
        fill(CGRect(x: 32, y: 32, width: 960, height: 960), radius: 215, color: dark)
        fill(CGRect(x: 140, y: 294, width: 744, height: 436), radius: 64, color: light)
        for row in 0..<3 {
            for column in 0..<10 {
                fill(CGRect(x: 182 + column * 67, y: 454 + row * 78, width: 54, height: 54),
                    radius: 12, color: row == 2 && column == 9 ? accent : dark)
            }
        }
        fill(CGRect(x: 182, y: 350, width: 100, height: 62), radius: 14, color: dark)
        fill(CGRect(x: 307, y: 350, width: 410, height: 62), radius: 14, color: accent)
        fill(CGRect(x: 742, y: 350, width: 100, height: 62), radius: 14, color: dark)
        guard let png = bitmap.representation(using: .png, properties: [:]) else {
            throw KeybedError(message: "Could not save the app icon.")
        }
        try png.write(to: directory.appendingPathComponent("icon_\(name).png"))
    }
}

if CommandLine.arguments.contains("--no-audio") && !CommandLine.arguments.contains("--smoke-test") {
    fputs("Keybed: --no-audio is only supported with --smoke-test.\n", stderr)
    exit(1)
}

if CommandLine.arguments.contains("--close-studio") {
    DistributedNotificationCenter.default().postNotificationName(KeybedApp.closeStudioNotification,
        object: Bundle.main.bundlePath, userInfo: nil, deliverImmediately: true)
} else if CommandLine.arguments.contains("--stop-running") {
    let running = NSRunningApplication.runningApplications(withBundleIdentifier: "com.keybed.keyboard-sound")
        .filter { $0.processIdentifier != ProcessInfo.processInfo.processIdentifier }
    running.forEach { _ = $0.terminate() }
    let deadline = Date().addingTimeInterval(3)
    while running.contains(where: { !$0.isTerminated }) && Date() < deadline {
        RunLoop.current.run(until: Date().addingTimeInterval(0.05))
    }
    running.filter { !$0.isTerminated }.forEach { _ = $0.forceTerminate() }
} else if CommandLine.arguments.contains("--make-icon") {
    do { try makeIcon(in: URL(fileURLWithPath: CommandLine.arguments.last!)) }
    catch { fputs("Keybed: \(error.localizedDescription)\n", stderr); exit(1) }
} else if CommandLine.arguments.contains("--enable-login") || CommandLine.arguments.contains("--disable-login") {
    do { try LoginStartup.setEnabled(CommandLine.arguments.contains("--enable-login")) }
    catch { fputs("Keybed: \(error.localizedDescription)\n", stderr); exit(1) }
} else if let index = CommandLine.arguments.firstIndex(of: "--export-windows-sounds") {
    do {
        guard index + 1 < CommandLine.arguments.count, let resources = Bundle.main.resourceURL else {
            throw KeybedError(message: "Usage: Keybed --export-windows-sounds /path/to/Keybed.soundbank")
        }
        try KeyboardAudio.exportWindowsSoundBank(resourceURL: resources.appendingPathComponent("Sounds"),
            output: URL(fileURLWithPath: CommandLine.arguments[index + 1]))
    } catch { fputs("Keybed: \(error.localizedDescription)\n", stderr); exit(1) }
} else if CommandLine.arguments.contains("--self-test") {
    do {
        guard let resources = Bundle.main.resourceURL else { throw KeybedError(message: "Missing app resources.") }
        let audio = try KeyboardAudio(resourceURL: resources.appendingPathComponent("Sounds"))
        try audio.selfTest()
        try TypingActivity.selfTest()
    } catch {
        fputs("Keybed: \(error.localizedDescription)\n", stderr)
        exit(1)
    }
} else {
    if !CommandLine.arguments.contains("--smoke-test"),
       let existing = NSRunningApplication.runningApplications(withBundleIdentifier: "com.keybed.keyboard-sound")
        .first(where: { $0.processIdentifier != ProcessInfo.processInfo.processIdentifier }) {
        existing.activate(options: [.activateIgnoringOtherApps, .activateAllWindows])
        exit(0)
    }
    let app = NSApplication.shared
    let delegate = KeybedApp()
    app.delegate = delegate
    app.run()
}
