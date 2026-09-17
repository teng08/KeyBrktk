import AppKit

final class CursorPanel: NSPanel {
    override var canBecomeKey: Bool { false }
    override var canBecomeMain: Bool { false }
}

final class HUDView: NSView {
    override var isFlipped: Bool { true }
    var count = 0
    var preset = SoundPreset.all[0]
    var muted = false
    var pulse: CGFloat = 0
    var phase: CGFloat = 0
    var collapse: CGFloat = 0

    override func draw(_ dirtyRect: NSRect) {
        let box = bounds.insetBy(dx: 2, dy: 2)
        NSColor(srgbRed: 0.075, green: 0.085, blue: 0.105, alpha: 0.96).setFill()
        NSBezierPath(roundedRect: box, xRadius: 16, yRadius: 16).fill()
        preset.color.withAlphaComponent(0.3 + pulse * 0.5).setStroke()
        let border = NSBezierPath(roundedRect: box.insetBy(dx: 0.5, dy: 0.5), xRadius: 16, yRadius: 16)
        border.lineWidth = 1
        border.stroke()
        preset.color.withAlphaComponent(0.10 + pulse * 0.18).setFill()
        let centerY = 32 - collapse * 14
        let diameter = 34 - collapse * 12 + pulse * 4
        NSBezierPath(ovalIn: NSRect(x: 29 - diameter / 2, y: centerY - diameter / 2, width: diameter, height: diameter)).fill()
        for index in 0..<5 {
            let wave = abs(sin(phase * 12 + CGFloat(index) * 1.4))
            let height: CGFloat = 5 + pulse * (9 + wave * 11)
            preset.color.withAlphaComponent(muted ? 0.4 : 0.95).setFill()
            NSBezierPath(roundedRect: NSRect(x: 19 + CGFloat(index) * 4, y: centerY - height / 2, width: 2.5, height: height), xRadius: 1.2, yRadius: 1.2).fill()
        }
        let countText = NumberFormatter.localizedString(from: NSNumber(value: count), number: .decimal)
        (countText + " keys" as NSString).draw(at: NSPoint(x: 57, y: 12), withAttributes: [
            .font: NSFont.monospacedDigitSystemFont(ofSize: 16, weight: .semibold), .foregroundColor: NSColor.white.withAlphaComponent(1 - collapse)])
        ((muted ? "Muted · " : "") + preset.name as NSString).draw(at: NSPoint(x: 57, y: 36 - collapse * 26), withAttributes: [
            .font: NSFont.systemFont(ofSize: 11, weight: .medium), .foregroundColor: preset.color])
    }
}

final class FloatingHUD {
    let panel: CursorPanel
    let view = HUDView(frame: NSRect(x: 0, y: 0, width: 218, height: 66))
    private let typing: TypingActivity
    private var timer: Timer?
    private var previewTime: TimeInterval = 0
    private var enabled = false
    var onCountChange: ((Int) -> Void)?

    init(typing: TypingActivity) {
        self.typing = typing
        panel = CursorPanel(contentRect: view.bounds, styleMask: [.borderless, .nonactivatingPanel],
                            backing: .buffered, defer: false)
        panel.contentView = view
        panel.isOpaque = false
        panel.backgroundColor = .clear
        panel.hasShadow = true
        panel.level = .floating
        panel.ignoresMouseEvents = true
        panel.hidesOnDeactivate = false
        panel.isReleasedWhenClosed = false
        panel.collectionBehavior = [.canJoinAllSpaces, .fullScreenAuxiliary, .ignoresCycle, .stationary]
        panel.isExcludedFromWindowsMenu = true
    }

    deinit { timer?.invalidate() }

    func configure(enabled: Bool, preset: SoundPreset, muted: Bool) {
        view.preset = preset
        view.muted = muted
        view.needsDisplay = true
        panel.alphaValue = muted ? 0.72 : 1
        guard self.enabled != enabled else { return }
        self.enabled = enabled
        if enabled {
            tick(immediate: true)
            panel.orderFrontRegardless()
            let timer = Timer(timeInterval: 1.0 / 30, repeats: true) { [weak self] _ in self?.tick() }
            self.timer = timer
            RunLoop.main.add(timer, forMode: .common)
        } else {
            timer?.invalidate()
            timer = nil
            panel.orderOut(nil)
        }
    }

    func preview() { previewTime = ProcessInfo.processInfo.systemUptime; view.needsDisplay = true }

    static func origin(cursor: NSPoint, visibleFrame: NSRect, size: NSSize) -> NSPoint {
        var x = cursor.x + 18
        var y = cursor.y - size.height - 18
        if x + size.width > visibleFrame.maxX - 8 { x = cursor.x - size.width - 18 }
        if y < visibleFrame.minY + 8 { y = cursor.y + 18 }
        x = max(visibleFrame.minX + 8, min(x, visibleFrame.maxX - size.width - 8))
        y = max(visibleFrame.minY + 8, min(y, visibleFrame.maxY - size.height - 8))
        return NSPoint(x: x, y: y)
    }

    private func tick(immediate: Bool = false) {
        let snapshot = typing.snapshot
        let now = ProcessInfo.processInfo.systemUptime
        let age = now - max(snapshot.lastHit, previewTime)
        let reduceMotion = NSWorkspace.shared.accessibilityDisplayShouldReduceMotion
        let pulse = reduceMotion ? CGFloat(0) : CGFloat(age < 0.8 ? exp(-age * 8) : 0)
        let targetCollapse: CGFloat = age >= TypingActivity.resetInterval ? 1 : 0
        let difference = targetCollapse - view.collapse
        let collapse = immediate || reduceMotion || abs(difference) < 0.002 ? targetCollapse : view.collapse + difference * 0.28
        if view.collapse != collapse {
            view.collapse = collapse
            let height = (66 - collapse * 30).rounded()
            let frame = panel.frame
            panel.setFrame(NSRect(x: frame.minX, y: frame.maxY - height, width: frame.width, height: height), display: false)
            view.needsDisplay = true
        }
        if view.count != snapshot.count {
            view.count = snapshot.count
            onCountChange?(snapshot.count)
            view.needsDisplay = true
        }
        if view.pulse != pulse {
            view.pulse = pulse
            view.phase = CGFloat(now)
            view.needsDisplay = true
        }
        let cursor = NSEvent.mouseLocation
        guard let screen = NSScreen.screens.first(where: { NSMouseInRect(cursor, $0.frame, false) }) ?? NSScreen.main else { return }
        let target = Self.origin(cursor: cursor, visibleFrame: screen.visibleFrame, size: panel.frame.size)
        let old = panel.frame.origin
        let distance = hypot(target.x - old.x, target.y - old.y)
        if distance < 0.4 { return }
        let amount: CGFloat = immediate || reduceMotion || distance > 500 ? 1 : 0.48
        let next = NSPoint(x: old.x + (target.x - old.x) * amount, y: old.y + (target.y - old.y) * amount)
        panel.setFrameOrigin(next)
    }
}

final class StudioWindow: NSWindow {
    var allowsInput = true
    override var canBecomeKey: Bool { allowsInput }
    override var canBecomeMain: Bool { allowsInput }
}

final class StudioView: NSView {
    override var isFlipped: Bool { true }
    override func draw(_ dirtyRect: NSRect) {
        NSColor(srgbRed: 0.065, green: 0.075, blue: 0.095, alpha: 1).setFill()
        bounds.fill()
    }
}

// Legacy (always-visible) scrollbars occupy space; overlay scrollbars do not.
// Follow the clip view's width so neither style creates hidden horizontal content.
final class SoundLibraryView: NSView {
    override var isFlipped: Bool { true }
    private let featuredIndex: Int
    private let gridPositions: [Int: Int]

    init(frame: NSRect, featuredIndex: Int, gridOrder: [Int]) {
        self.featuredIndex = featuredIndex
        gridPositions = Dictionary(uniqueKeysWithValues: gridOrder.enumerated().map { ($0.element, $0.offset) })
        super.init(frame: frame)
        autoresizingMask = [.width]
    }

    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }

    override func setFrameSize(_ newSize: NSSize) {
        super.setFrameSize(newSize)
        layoutCards()
    }

    func layoutCards() {
        let pitch = (bounds.width + 10) / 3
        for case let card as SoundCard in subviews {
            if card.tag == featuredIndex {
                card.frame = NSRect(x: 0, y: 0, width: bounds.width, height: 62)
            } else if let position = gridPositions[card.tag] {
                let column = position % 3
                let x = (CGFloat(column) * pitch).rounded()
                let right = column == 2 ? bounds.width : (CGFloat(column + 1) * pitch).rounded() - 10
                card.frame = NSRect(x: x, y: CGFloat(70 + (position / 3) * 70), width: right - x, height: 62)
            }
        }
    }
}

final class SoundCard: NSButton {
    override var isFlipped: Bool { true }
    let preset: SoundPreset
    private let badge: NSTextField
    var selected = false { didSet { needsDisplay = true; badge.stringValue = selected ? "●" : (tag == 0 ? "" : "NEW") } }

    init(preset: SoundPreset, index: Int, frame: NSRect, target: AnyObject, action: Selector) {
        self.preset = preset
        badge = NSTextField(labelWithString: index == 0 ? "" : "NEW")
        super.init(frame: frame)
        title = ""
        isBordered = false
        tag = index
        self.target = target
        self.action = action
        setAccessibilityLabel(preset.name + ". " + preset.detail)
        let icon = NSImageView(frame: NSRect(x: 14, y: 11, width: 23, height: 23))
        icon.image = NSImage(systemSymbolName: preset.symbol, accessibilityDescription: nil)
        icon.contentTintColor = preset.color
        addSubview(icon)
        let name = NSTextField(labelWithString: preset.name)
        name.font = .systemFont(ofSize: 13, weight: .semibold)
        name.textColor = .white
        name.frame = NSRect(x: 47, y: 13, width: frame.width - 81, height: 20)
        name.autoresizingMask = [.width]
        addSubview(name)
        let detail = NSTextField(labelWithString: preset.detail)
        detail.font = .systemFont(ofSize: 10.5)
        detail.textColor = NSColor(white: 0.66, alpha: 1)
        detail.frame = NSRect(x: 14, y: frame.height - 26, width: frame.width - 25, height: 18)
        detail.autoresizingMask = [.width]
        addSubview(detail)
        badge.frame = NSRect(x: frame.width - 36, y: 19, width: 29, height: 14)
        badge.autoresizingMask = [.minXMargin]
        badge.font = .systemFont(ofSize: 8, weight: .bold)
        badge.textColor = preset.color
        addSubview(badge)
    }
    required init?(coder: NSCoder) { fatalError("init(coder:) is not supported") }
    override func hitTest(_ point: NSPoint) -> NSView? {
        let local = convert(point, from: superview)
        return bounds.contains(local) ? self : nil
    }
    override func draw(_ dirtyRect: NSRect) {
        let rect = bounds.insetBy(dx: 1, dy: 1)
        (selected ? preset.color.withAlphaComponent(0.12) : NSColor(white: isHighlighted ? 0.16 : 0.115, alpha: 1)).setFill()
        let path = NSBezierPath(roundedRect: rect, xRadius: 12, yRadius: 12)
        path.fill()
        (selected ? preset.color.withAlphaComponent(0.85) : NSColor(white: 0.22, alpha: 1)).setStroke()
        path.lineWidth = selected ? 1.5 : 1
        path.stroke()
    }
}
