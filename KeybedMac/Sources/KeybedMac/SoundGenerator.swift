import Foundation

enum SoundGenerator {
    static let names = ["press_key1", "press_key2", "press_key3", "press_key4", "press_key5",
                        "press_space", "press_enter", "press_back", "release_key",
                        "release_space", "release_enter", "release_back"]

    // Offline synthesis: no oscillators, random generation or DSP on a key press.
    static func make(style: String, key: String, variant: Int) -> [Float] {
        let release = key.hasPrefix("release")
        let wide = key.hasSuffix("space") || key.hasSuffix("enter")
        let pitch = (wide ? 0.78 : 1.0) * (release ? 1.28 : 1.0) * (1 + Double(variant % 5 - 2) * 0.022)
        let duration = release ? 0.055 : 0.13
        let count = Int(duration * 44_100)
        var seed = UInt32(7_919 + variant * 1_009)
        var lowNoise = 0.0
        var result = [Float](repeating: 0, count: count)
        for frame in 0..<count {
            let t = Double(frame) / 44_100
            seed = 1_664_525 &* seed &+ 1_013_904_223
            let noise = Double(seed) / Double(UInt32.max) * 2 - 1
            lowNoise += 0.10 * (noise - lowNoise)
            let click = noise * exp(-t * 900)
            let life = release ? 1.7 : 1.0
            let envelope = exp(-t * 45 * life)
            let phase = 2 * Double.pi * pitch
            let value: Double
            switch style {
            case "thock":
                value = (sin(phase * (150 * t + 65 * t * exp(-t * 50)) + 0.35) * 0.80
                    + sin(phase * 310 * t) * 0.20 + lowNoise * 0.45) * exp(-t * 38 * life) + click * 0.16
            case "marble":
                value = (sin(phase * 430 * t + 0.4) * 0.65 + sin(phase * 1_170 * t) * 0.23
                    + sin(phase * 1_870 * t) * 0.08 + lowNoise * 0.35) * exp(-t * 58 * life) + click * 0.12
            case "typewriter":
                let clack = noise * (exp(-t * 260) + 0.6 * exp(-abs(t - 0.013) * 650))
                value = clack * 0.62 + (sin(phase * 1_480 * t + 0.5) * 0.20
                    + sin(phase * 2_370 * t) * 0.18) * exp(-t * 55 * life)
            case "bubble":
                let bubblePhase = phase * (560 * t + 65 * (1 - exp(-t * 30)))
                value = sin(bubblePhase + 0.4) * envelope + click * 0.06
            case "pixel":
                let note = t < 0.024 ? 660.0 : 990.0
                let square = sin(phase * note * t + 0.4) >= 0 ? 0.65 : -0.65
                value = (square + sin(phase * note * t) * 0.2) * exp(-t * 65 * life)
            default: // Birdy: a brief upward chirp, with a soft percussive onset.
                let chirp = phase * (1_400 * t + 8_000 * t * t)
                value = (sin(chirp + 0.4) * 0.82 + sin(chirp * 1.5) * 0.12) * exp(-t * 48 * life) + click * 0.08
            }
            result[frame] = Float(value)
        }
        return result
    }

    static func reshape(_ frames: [Float], key: String, variant: Int) -> [Float] {
        let release = key.hasPrefix("release")
        let wide = key.hasSuffix("space") || key.hasSuffix("enter")
        let speed = (wide ? 0.88 : 1.0) * (release ? 1.3 : 1.0) * (1 + Double(variant % 5 - 2) * 0.012)
        let count = max(1, Int(Double(frames.count) / speed))
        return (0..<count).map { frame in
            let position = Double(frame) * speed
            let lower = min(frames.count - 1, Int(position))
            let upper = min(frames.count - 1, lower + 1)
            return frames[lower] + (frames[upper] - frames[lower]) * Float(position - Double(lower))
        }
    }
}
