// swift-tools-version:5.7
import PackageDescription

let package = Package(
    name: "KeybedMac",
    platforms: [.macOS(.v11)],
    targets: [
        .target(name: "AudioMixer", path: "Sources/AudioMixer", publicHeadersPath: "include"),
        .executableTarget(
            name: "KeybedMac",
            dependencies: ["AudioMixer"],
            path: "Sources/KeybedMac"
        )
    ]
)
