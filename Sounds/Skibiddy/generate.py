#!/usr/bin/env python3
"""Recreate Keybed's robot vocal samples using the locally installed Fred voice."""
import math
from pathlib import Path
import struct
import subprocess
import tempfile
import wave


def generate():
    destination = Path(__file__).resolve().parent
    phrases = {
        "skibiddy": "skibiddy",
        "toilet": "toilet",
        "dop": "dop",
        "dop-dop": "dop dop",
        "yes-yes": "yes yes",
        "skibiddy-toilet": "skibiddy toilet",
    }
    with tempfile.TemporaryDirectory(prefix="keybed-vocals-") as temporary:
        for name, phrase in phrases.items():
            aiff = str(Path(temporary) / (name + ".aiff"))
            wav = str(Path(temporary) / (name + ".wav"))
            subprocess.run(["/usr/bin/say", "-v", "Fred", "-r", "430", "-o", aiff, phrase], check=True)
            subprocess.run(["/usr/bin/afconvert", "-f", "WAVE", "-d", "LEI16@44100", aiff, wav], check=True)
            with wave.open(wav, "rb") as source:
                assert source.getsampwidth() == 2 and source.getnchannels() == 1
                raw = source.readframes(source.getnframes())
            frames = list(struct.unpack("<" + "h" * (len(raw) // 2), raw))
            peak = max(abs(value) for value in frames)
            first = next(index for index, value in enumerate(frames) if abs(value) > peak * 0.012)
            last = max(index for index, value in enumerate(frames) if abs(value) > peak * 0.008)
            frames = frames[max(0, first - 16):last + 1]
            # A little saturation gives the deliberately robotic chant more bite.
            frames = [int(math.tanh(value / peak * 1.4) * 24575) for value in frames]
            for index in range(min(110, len(frames))):
                frames[-1 - index] = int(frames[-1 - index] * index / 110)
            path = destination / (name + ".wav")
            with wave.open(str(path), "wb") as output:
                output.setnchannels(1)
                output.setsampwidth(2)
                output.setframerate(44100)
                output.writeframes(struct.pack("<" + "h" * len(frames), *frames))
            print(f"{path.name}: {len(frames) / 44100:.3f}s")


if __name__ == "__main__":
    generate()
