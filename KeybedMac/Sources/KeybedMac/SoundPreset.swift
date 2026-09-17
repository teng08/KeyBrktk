import AppKit

struct SoundPreset {
    let id: String
    let name: String
    let detail: String
    let symbol: String
    let color: NSColor

    static let all: [SoundPreset] = [
        .init(id: "alpaca", name: "Alpaca Linear", detail: "The original · crisp & clean", symbol: "keyboard", color: .init(srgbRed: 1, green: 0.60, blue: 0.35, alpha: 1)),
        .init(id: "tactile", name: "Tactile Brown", detail: "A warm, satisfying bump", symbol: "square.stack.3d.up", color: .init(srgbRed: 0.82, green: 0.66, blue: 0.49, alpha: 1)),
        .init(id: "blue", name: "Clicky Blue", detail: "Bright mechanical clicks", symbol: "bolt.fill", color: .init(srgbRed: 0.43, green: 0.72, blue: 1, alpha: 1)),
        .init(id: "thock", name: "Deep Thock", detail: "Low, rounded & punchy", symbol: "waveform", color: .init(srgbRed: 0.68, green: 0.57, blue: 1, alpha: 1)),
        .init(id: "marble", name: "Creamy Marble", detail: "Soft taps with a glassy body", symbol: "circle.hexagongrid.fill", color: .init(srgbRed: 0.89, green: 0.82, blue: 0.65, alpha: 1)),
        .init(id: "typewriter", name: "Typewriter", detail: "Metallic, old-school clacks", symbol: "textformat.abc", color: .init(srgbRed: 0.77, green: 0.82, blue: 0.85, alpha: 1)),
        .init(id: "bubble", name: "Bubble Pop", detail: "Tiny, playful water drops", symbol: "drop.fill", color: .init(srgbRed: 0.36, green: 0.86, blue: 0.77, alpha: 1)),
        .init(id: "pixel", name: "Pixel Tap", detail: "A little arcade in every key", symbol: "gamecontroller.fill", color: .init(srgbRed: 1, green: 0.48, blue: 0.65, alpha: 1)),
        .init(id: "birdy", name: "Birdy Chirp", detail: "Quick, cheerful little chirps", symbol: "music.note", color: .init(srgbRed: 0.75, green: 0.90, blue: 0.41, alpha: 1)),
        .init(id: "skibiddy", name: "Skibiddy Toilet", detail: "Robot vocals · skibiddy, toilet, dop dop", symbol: "speaker.wave.2.fill", color: .init(srgbRed: 0.46, green: 0.95, blue: 0.65, alpha: 1))
    ]
    static func index(for id: String?) -> Int { all.firstIndex { $0.id == id } ?? 0 }
}

// Only a count and a timestamp are retained; typed characters are never stored.
final class TypingActivity {
    static let resetInterval: TimeInterval = 3
    private let lock = NSLock()
    private let clock: () -> TimeInterval
    private var count = 0
    private var lastHit: TimeInterval = 0
    private var windowStart: TimeInterval?

    init(clock: @escaping () -> TimeInterval = { ProcessInfo.processInfo.systemUptime }) { self.clock = clock }

    // A typing window expires even if keys continue arriving or the HUD is hidden.
    private func expire(at now: TimeInterval) {
        if let windowStart, now - windowStart >= Self.resetInterval {
            count = 0
            self.windowStart = nil
        }
    }
    func record() {
        lock.lock()
        let now = clock()
        expire(at: now)
        if windowStart == nil { windowStart = now }
        count += 1
        lastHit = now
        lock.unlock()
    }
    func reset() {
        lock.lock()
        count = 0
        lastHit = 0
        windowStart = nil
        lock.unlock()
    }
    var snapshot: (count: Int, lastHit: TimeInterval) {
        lock.lock()
        defer { lock.unlock() }
        expire(at: clock())
        return (count, lastHit)
    }
    static func selfTest() throws {
        let activity = TypingActivity(clock: { 1 })
        DispatchQueue.concurrentPerform(iterations: 4) { _ in
            for _ in 0..<1_000 { activity.record() }
        }
        guard activity.snapshot.count == 4_000, activity.snapshot.lastHit > 0 else {
            throw KeybedError(message: "Concurrent keystrokes were lost by the typing counter.")
        }
        activity.reset()
        guard activity.snapshot.count == 0, activity.snapshot.lastHit == 0 else {
            throw KeybedError(message: "Reset must clear the typing count and pulse.")
        }
        var now: TimeInterval = 100
        let timed = TypingActivity(clock: { now })
        timed.record()
        now = 102.999
        timed.record()
        guard timed.snapshot.count == 2 else { throw KeybedError(message: "The counter reset before three seconds.") }
        now = 103
        guard timed.snapshot.count == 0, timed.snapshot.lastHit == 102.999 else {
            throw KeybedError(message: "The three-second window must reset during continuous typing.")
        }
        timed.record()
        guard timed.snapshot.count == 1 else { throw KeybedError(message: "A new typing window must start at one.") }
        now = 200
        timed.record() // No timer snapshots during this gap, as when the overlay is hidden.
        guard timed.snapshot.count == 1 else { throw KeybedError(message: "An expired hidden counter must restart at one.") }
        now = 203
        guard timed.snapshot.count == 0 else { throw KeybedError(message: "Idle typing windows must also reset.") }
        print("PASS: concurrent typing count, three-second boundary, continuous typing, hidden counter and manual reset.")
    }
}
