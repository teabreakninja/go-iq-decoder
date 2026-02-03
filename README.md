# Go IQ Decoder

A software-defined radio (SDR) FM demodulator written in Go that processes IQ samples and outputs playable audio in real-time.

## Overview

This project reads raw IQ samples (either from a `.iq` file or WAV container), performs FM demodulation through a multi-stage digital signal processing pipeline, and plays the resulting audio through your system's audio output.

The processing pipeline consists of:
1. **Channel filtering and decimation** (2 MHz → 240 kHz) - Isolates the FM station
2. **FM demodulation** - Extracts audio from the carrier signal using phase differentiation
3. **Audio filtering and resampling** (240 kHz → 48 kHz) - Produces clean, playable audio with de-emphasis

## Project Structure

```
go-iq-decoder/
├── cmd/
│   └── go-audio-mini-project/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration parameters
│   ├── dsp/
│   │   ├── deemphasis.go        # De-emphasis filter
│   │   ├── demodulator.go       # FM demodulator
│   │   ├── dsp.go               # DSP utilities
│   │   ├── fir.go               # FIR filter implementation
│   │   └── *_test.go            # Unit tests
│   └── ringbuffer/
│       ├── ringbuffer.go        # Thread-safe ring buffer
│       └── ringbuffer_test.go   # Unit tests
└── README.md
```

## Libraries Used

### Why Ebitengine Oto (github.com/ebitengine/oto/v3)?

**Oto v3** is chosen for audio output because it provides:

- **Cross-platform compatibility** - Works seamlessly on Windows, macOS, and Linux without platform-specific code
- **Low-level control** - Direct access to the audio hardware with minimal latency
- **Simple, modern API** - Clean Go-native interface without C bindings complexity
- **Real-time streaming** - Designed for continuous audio streams, perfect for SDR applications
- **Active maintenance** - Part of the Ebitengine ecosystem, well-maintained and widely used in Go audio/game projects

Unlike older audio libraries that require CGO or have heavyweight dependencies, Oto v3 uses **purego** to interface with system audio APIs, making builds faster and deployment simpler.

### go-audio/wav (github.com/go-audio/wav)

Handles WAV file parsing and PCM data extraction. This allows the project to accept both raw `.iq` files and IQ data wrapped in WAV containers, providing flexibility in input formats.

### go-audio/audio (github.com/go-audio/audio)

Provides standard audio buffer types and utilities for working with PCM data. Used in conjunction with the WAV decoder for efficient sample processing.

## Configuration

Default configuration (`internal/config/config.go`):

- **IQ Sample Rate**: 2 MHz (typical RTL-SDR output)
- **Intermediate Rate**: 240 kHz (after channel filtering)
- **Output Sample Rate**: 48 kHz (standard audio playback rate)
- **Filter Taps**: 251 (high-quality FIR filters)
- **De-emphasis**: 50 µs (European FM standard)

## Building

```bash
go build -o go-audio-mini-project.exe ./cmd/go-audio-mini-project
```

Or for release builds with optimizations:

```bash
go build -ldflags="-s -w" -o go-audio-mini-project-release.exe ./cmd/go-audio-mini-project
```

## Running

Place your IQ sample file (named `sample2.iq` or modify `main.go`) in the project directory and run:

```bash
./go-audio-mini-project.exe
```

The program will:
1. Open the IQ file
2. Initialize audio output via Oto v3
3. Start three concurrent goroutines:
   - File reader (populates ring buffer)
   - DSP processor (demodulates FM signal)
   - Audio player (streams to speakers)
4. Run continuously until the file ends

## Input Format

Accepts two input formats:

1. **Raw IQ files** (`.iq`) - 16-bit little-endian interleaved I/Q samples
2. **WAV files** - 16-bit PCM containing interleaved I/Q data (detected automatically)

## Technical Details

### Multi-stage Processing

The two-stage decimation approach prevents aliasing while efficiently reducing the sample rate from 2 MHz to 48 kHz:

- **Stage 1**: FIR low-pass filter + decimation by ~8.3x
- **Stage 2**: FIR low-pass filter + decimation by 5x

### FM Demodulation

Uses **phase differentiation** to extract the instantaneous frequency from the complex IQ signal. This converts the frequency-modulated carrier into an audio waveform.

### De-emphasis

Applies a 50 µs de-emphasis filter to compensate for the pre-emphasis applied during FM transmission, restoring flat frequency response.

## License

GPL3
