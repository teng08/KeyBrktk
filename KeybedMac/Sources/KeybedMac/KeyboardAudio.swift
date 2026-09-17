import AVFoundation
import AudioMixer

private final class MixerStorage {
    let pointer: OpaquePointer
    init() throws {
        guard let pointer = KBMixerCreate() else { throw KeybedError(message: "Could not prepare the sound engine.") }
        self.pointer = pointer
    }
    deinit { KBMixerDestroy(pointer) }
}

// Decode, trim and process once. The audio callback only mixes preloaded PCM.
final class KeyboardAudio {
    static let intensityNames = ["Balanced", "Aggressive", "Extreme"]
    private let engine = AVAudioEngine()
    private let storage: MixerStorage
    private var mixer: OpaquePointer { storage.pointer }
    private var source: AVAudioSourceNode!
    private var sampleIDs: [[[String: Int32]]] = []
    private var preset = 0
    private let settingsLock = NSLock()
    private var intensity = 1
    private var releaseSounds = true
    private var variant = 0
    private var configurationObserver: NSObjectProtocol?
    private(set) var lastError: String?
    var isRunning: Bool { engine.isRunning }

    init(resourceURL: URL) throws {
        storage = try MixerStorage()
        let mixer = storage.pointer
        for preset in SoundPreset.all {
            let decoded = try Self.decodedSamples(preset: preset, resourceURL: resourceURL)
            var modes: [[String: Int32]] = []
            for mode in 0..<Self.intensityNames.count {
                var ids: [String: Int32] = [:]
                for (name, frames) in decoded {
                    let prepared = Self.prepare(frames, intensity: mode, release: name.hasPrefix("release"),
                                                maximumFrames: preset.id == "skibiddy" ? 44_100 : 7_938)
                    let id = prepared.withUnsafeBufferPointer { KBMixerAddSample(mixer, $0.baseAddress, UInt32($0.count)) }
                    guard id >= 0 else { throw KeybedError(message: "Could not preload \(preset.name) / \(name).") }
                    ids[name] = id
                }
                modes.append(ids)
            }
            sampleIDs.append(modes)
        }
        let format = AVAudioFormat(standardFormatWithSampleRate: 44_100, channels: 1)!
        source = AVAudioSourceNode(format: format) { silence, _, frameCount, bufferList in
            let buffers = UnsafeMutableAudioBufferListPointer(bufferList)
            guard let data = buffers[0].mData else { return noErr }
            let audible = KBMixerRender(mixer, data.assumingMemoryBound(to: Float.self), frameCount)
            silence.pointee = ObjCBool(!audible)
            return noErr
        }
        engine.attach(source)
        engine.connect(source, to: engine.mainMixerNode, format: format)
    }

    // Both desktop apps use the same decoded, normalized and processed samples.
    private static func decodedSamples(preset: SoundPreset, resourceURL: URL) throws -> [(String, [Float])] {
            if let directory = preset.recordingDirectory {
                return try SoundGenerator.names.map { name in
                    (name, try Self.decode(resourceURL.appendingPathComponent(directory + "/" + name + ".mp3")))
                }
            }
            let decoded: [(String, [Float])]
            switch preset.id {
            case "skibiddy":
                let phrases = ["skibiddy", "skibiddy", "toilet", "dop", "yes-yes", "toilet", "skibiddy-toilet", "dop-dop"]
                let vocals = try phrases.map { try Self.decode(resourceURL.appendingPathComponent("Skibiddy/" + $0 + ".wav")) }
                decoded = SoundGenerator.names.enumerated().map { index, name in
                    if name.hasPrefix("release") {
                        return (name, SoundGenerator.make(style: "bubble", key: name, variant: index))
                    }
                    return (name, SoundGenerator.reshape(vocals[index], key: name, variant: index))
                }
            case "tactile", "blue":
                let paths = preset.id == "tactile"
                    ? (1...4).map { String(format: "Tactile/stav-tactile-%02d.mp3", $0) }
                    : ["BlueSwitch/q-key.mp3", "BlueSwitch/w-key.mp3", "BlueSwitch/spacebar-key.mp3", "BlueSwitch/ctrl-key.mp3"]
                let recordings = try paths.map { try Self.decode(resourceURL.appendingPathComponent($0)) }
                decoded = SoundGenerator.names.enumerated().map { index, name in
                    let sourceIndex = preset.id == "blue" && name.hasSuffix("space") ? 2 : index % recordings.count
                    return (name, SoundGenerator.reshape(recordings[sourceIndex], key: name, variant: index))
                }
            default:
                decoded = SoundGenerator.names.enumerated().map { index, name in
                    (name, SoundGenerator.make(style: preset.id, key: name, variant: index))
                }
            }
            return decoded
    }

    static func exportWindowsSoundBank(resourceURL: URL, output: URL) throws {
        var data = Data("KBPCM001".utf8)
        func append(_ value: UInt32) {
            var littleEndian = value.littleEndian
            withUnsafeBytes(of: &littleEndian) { data.append(contentsOf: $0) }
        }
        append(44_100)
        append(UInt32(SoundPreset.all.count))
        append(UInt32(intensityNames.count))
        append(UInt32(SoundGenerator.names.count))
        for preset in SoundPreset.all {
            let decoded = try decodedSamples(preset: preset, resourceURL: resourceURL)
            for mode in intensityNames.indices {
                for (name, frames) in decoded {
                    let prepared = prepare(frames, intensity: mode, release: name.hasPrefix("release"),
                                           maximumFrames: preset.id == "skibiddy" ? 44_100 : 7_938)
                    append(UInt32(prepared.count))
                    let words = prepared.map { $0.bitPattern.littleEndian }
                    words.withUnsafeBytes { data.append(contentsOf: $0) }
                }
            }
        }
        try FileManager.default.createDirectory(at: output.deletingLastPathComponent(), withIntermediateDirectories: true)
        try data.write(to: output, options: .atomic)
        let count = SoundPreset.all.count * intensityNames.count * SoundGenerator.names.count
        print("Exported \(count) preloaded Windows samples to \(output.path).")
    }

    deinit {
        if let configurationObserver { NotificationCenter.default.removeObserver(configurationObserver) }
        engine.stop()
        if let source { engine.detach(source) }
    }

    private static func decode(_ url: URL) throws -> [Float] {
        guard FileManager.default.isReadableFile(atPath: url.path) else {
            throw KeybedError(message: "The recording is missing or unreadable: \(url.path)")
        }
        let file = try AVAudioFile(forReading: url)
        guard file.length > 0,
              let buffer = AVAudioPCMBuffer(pcmFormat: file.processingFormat, frameCapacity: AVAudioFrameCount(file.length)) else {
            throw KeybedError(message: "The recording \(url.lastPathComponent) is empty.")
        }
        try file.read(into: buffer)
        guard let channels = buffer.floatChannelData else { throw KeybedError(message: "Cannot decode \(url.lastPathComponent).") }
        let count = Int(buffer.frameLength)
        let channelCount = Int(buffer.format.channelCount)
        var mono = [Float](repeating: 0, count: count)
        for frame in 0..<count {
            for channel in 0..<channelCount { mono[frame] += channels[channel][frame] / Float(channelCount) }
        }
        let ratio = buffer.format.sampleRate / 44_100
        if abs(ratio - 1) < 0.0001 { return mono }
        let convertedCount = max(1, Int(Double(count) / ratio))
        return (0..<convertedCount).map { frame in
            let position = Double(frame) * ratio
            let lower = min(count - 1, Int(position))
            let upper = min(count - 1, lower + 1)
            return mono[lower] + (mono[upper] - mono[lower]) * Float(position - Double(lower))
        }
    }

    private static func prepare(_ frames: [Float], intensity: Int, release: Bool, maximumFrames: Int) -> [Float] {
        let peak = frames.reduce(Float(0)) { max($0, abs($1)) }
        guard peak > 0.00001, let first = frames.firstIndex(where: { abs($0) >= peak * 0.018 }) else { return [0] }
        // Retain 0.36 ms before the attack; remove MP3 encoder silence.
        let start = max(0, first - 16)
        let last = frames.lastIndex(where: { abs($0) >= peak * 0.008 }) ?? frames.count - 1
        let end = min(last + 1, start + (release ? 3_969 : maximumFrames))
        var result = Array(frames[start..<max(start + 1, end)])
        let brightness: Float = [0.12, 0.7, 1.25][intensity]
        let drive: Float = [1.0, 1.65, 2.5][intensity]
        let level: Float = [0.40, 0.62, 0.78][intensity] * (release ? 0.55 : 1)
        var body: Float = 0
        for index in result.indices {
            let value = result[index] / peak
            body += 0.24 * (value - body)
            result[index] = tanh((value + brightness * (value - body)) * drive) * level
            let remaining = result.count - 1 - index
            if remaining < 110 { result[index] *= Float(remaining) / 110 }
        }
        return result
    }

    func start() throws {
        engine.prepare()
        do {
            try engine.start()
            lastError = nil
        } catch {
            lastError = error.localizedDescription
            throw error
        }
        if configurationObserver == nil {
            configurationObserver = NotificationCenter.default.addObserver(forName: .AVAudioEngineConfigurationChange,
                object: engine, queue: .main) { [weak self] _ in
                    do { try self?.start() } catch { self?.lastError = error.localizedDescription }
                }
        }
    }

    func configure(preset: Int, intensity: Int, volume: Double, muted: Bool, releases: Bool) {
        settingsLock.lock()
        self.preset = min(SoundPreset.all.count - 1, max(0, preset))
        self.intensity = min(2, max(0, intensity))
        releaseSounds = releases
        KBMixerSetVolume(mixer, muted ? 0 : Float(volume))
        settingsLock.unlock()
    }

    func play(keyCode: UInt16, release: Bool = false) {
        settingsLock.lock()
        defer { settingsLock.unlock() }
        if release && !releaseSounds { return }
        let key: String
        switch keyCode {
        case 49: key = "space"
        case 36, 76: key = "enter"
        case 51, 117: key = "back"
        default:
            if release { key = "key" }
            else { variant = (variant + 1) % 5; key = "key\(variant + 1)" }
        }
        if let id = sampleIDs[preset][intensity][(release ? "release_" : "press_") + key] {
            _ = KBMixerPlay(mixer, id, 1)
        }
    }

    // Exercise the production render path without keyboard access or a device.
    func selfTest() throws {
        var output = [Float](repeating: 0, count: 49_152)
        KBMixerSetVolume(mixer, 0.8)
        var signatures = Set<[Int]>()
        var checked = 0
        for (preset, modes) in sampleIDs.enumerated() {
            for (mode, ids) in modes.enumerated() {
                for (name, id) in ids.sorted(by: { $0.key < $1.key }) {
                    guard KBMixerPlay(mixer, id, 1) else { throw KeybedError(message: "Could not queue \(name).") }
                    output.withUnsafeMutableBufferPointer { _ = KBMixerRender(mixer, $0.baseAddress, UInt32($0.count)) }
                    guard output.allSatisfy({ $0.isFinite && abs($0) <= 1 }),
                          let onset = output.firstIndex(where: { abs($0) > 0.001 }), onset <= 44 else {
                        throw KeybedError(message: "Audio check failed for \(SoundPreset.all[preset].name) / \(Self.intensityNames[mode]) / \(name).")
                    }
                    if mode == 1 && name == "press_key1" {
                        signatures.insert(output.prefix(512).map { Int($0 * 10_000) })
                    }
                    if SoundPreset.all[preset].id == "skibiddy" && name == "press_enter" {
                        guard let ending = output.lastIndex(where: { abs($0) > 0.001 }), ending > 22_050 else {
                            throw KeybedError(message: "The Skibiddy Toilet phrase was truncated.")
                        }
                    }
                    checked += 1
                }
            }
        }
        guard signatures.count == SoundPreset.all.count else { throw KeybedError(message: "Sound presets must have distinct waveforms.") }
        // Verify UI settings actually select the matching preloaded bank.
        for preset in SoundPreset.all.indices {
            for intensity in Self.intensityNames.indices {
                let id = sampleIDs[preset][intensity]["press_space"]!
                _ = KBMixerPlay(mixer, id, 1)
                output.withUnsafeMutableBufferPointer { _ = KBMixerRender(mixer, $0.baseAddress, UInt32($0.count)) }
                let expected = output
                configure(preset: preset, intensity: intensity, volume: 0.8, muted: false, releases: false)
                play(keyCode: 49)
                output.withUnsafeMutableBufferPointer { _ = KBMixerRender(mixer, $0.baseAddress, UInt32($0.count)) }
                guard output == expected else { throw KeybedError(message: "Sound selection did not reach the audio engine.") }
                play(keyCode: 49, release: true)
                output.withUnsafeMutableBufferPointer { _ = KBMixerRender(mixer, $0.baseAddress, UInt32($0.count)) }
                guard output.allSatisfy({ $0 == 0 }) else { throw KeybedError(message: "Disabled key releases must be silent.") }
            }
        }
        configure(preset: 0, intensity: 1, volume: 0.8, muted: true, releases: true)
        play(keyCode: 0)
        output.withUnsafeMutableBufferPointer { _ = KBMixerRender(mixer, $0.baseAddress, UInt32($0.count)) }
        guard output.allSatisfy({ $0 == 0 }) else { throw KeybedError(message: "Mute must silence the selected sound.") }
        print("PASS: \(SoundPreset.all.count) distinct presets, \(checked) preloaded sounds; PCM attacks within 1 ms; finite, limited output.")
        print("PASS: preset and intensity switching, disabled releases and mute through the keyboard playback path.")
    }
}
