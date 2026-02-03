package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ebitengine/oto/v3"
	"github.com/go-audio/audio"
	"github.com/go-audio/wav"

	"go-audio-mini-project/internal/config"
	"go-audio-mini-project/internal/dsp"
	"go-audio-mini-project/internal/ringbuffer"
)

func main() {
	// Get default configuration
	cfg := config.New()

	fmt.Println("Opening file...")
	file, err := os.Open("sample2.iq")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	fmt.Println("Creating ring buffer...")
	rb := ringbuffer.New(cfg.RingBufferSize)

	decoder := wav.NewDecoder(file)

	fmt.Println("Setting up audio...")
	// Setup Oto v3 context
	ctx, ready, err := oto.NewContext(&oto.NewContextOptions{
		SampleRate:   cfg.OutputSampleRate,
		ChannelCount: 1,
		Format:       oto.FormatSignedInt16LE,
	})
	if err != nil {
		panic(err)
	}
	<-ready

	reader, writer := io.Pipe()
	player := ctx.NewPlayer(reader)
	defer player.Close()

	go readFileIntoBuffer(file, decoder, rb, cfg)

	go player.Play()

	fmt.Println("Starting processing...")
	go processIQ(rb, writer, cfg)

	select {} // Block forever
}

// Read the file or IO stream into the ring buffer
// For the file, it may be in a WAV container, so we need to handle that
func readFileIntoBuffer(file *os.File, decoder *wav.Decoder, rb *ringbuffer.RingBuffer, cfg *config.Config) {
	defer rb.Close() // Ensure the buffer is closed when this function exits.
	if !decoder.IsValidFile() {
		fmt.Println("Not a valid WAV file, reading raw IQ...")
		buf := make([]byte, cfg.ChunkSize)
		for {
			n, err := file.Read(buf)
			if n > 0 {
				// Convert the []byte slice to []int16 before writing to the ring buffer.
				int16Buf := make([]int16, n/2)
				for i := 0; i < n/2; i++ {
					int16Buf[i] = int16(binary.LittleEndian.Uint16(buf[i*2 : i*2+2]))
				}
				rb.Write(int16Buf)
			}
			if err == io.EOF {
				break
			} else if err != nil {
				fmt.Println("File read error:", err)
				break
			}
		}

	} else {
		fmt.Println("Reading IQ from WAV file...")
		// Move to start of PCM/IQ data
		if err := decoder.FwdToPCM(); err != nil {
			log.Fatal("Failed to seek to PCM data:", err)
		}

		// Detect and print the audio format to confirm our assumptions.
		fmt.Printf("[INFO] Detected WAV format: Bit Depth: %d, Sample Rate: %d, Channels: %d\n",
			decoder.BitDepth, decoder.SampleRate, decoder.NumChans)

		if decoder.BitDepth != 16 {
			log.Fatalf("FATAL: This program is hardcoded to process 16-bit audio, but detected %d-bit.", decoder.BitDepth)
		}

		// Preallocate reusable buffer for streamed PCM data
		buf := &audio.IntBuffer{
			Format: decoder.Format(),
			Data:   make([]int, cfg.ChunkSize*2), // 2 = I+Q
		}

		fmt.Println("Adding to ring buffer...")
		for {
			n, err := decoder.PCMBuffer(buf)
			if err == io.EOF {
				fmt.Println("End of WAV file reached")
				break
			}

			samples := make([]int16, n)
			for i := 0; i < n; i += int(decoder.NumChans) {
				samples[i] = int16(buf.Data[i])
				samples[i+1] = int16(buf.Data[i+1])
			}
			rb.Write(samples)
		}
	}
}

func processIQ(rb *ringbuffer.RingBuffer, writer *io.PipeWriter, cfg *config.Config) {
	frameSize := cfg.SampleBlockSize * 2 // We need two int16 samples (I and Q) per complex sample.

	// --- Stage 1: Channel Selection Filter ---
	// This filter selects the ~200kHz FM station from the 2MHz SDR stream.
	channelTaps := dsp.DesignFIRLowPass(cfg.FilterTaps, cfg.ChannelFilterCutoff)
	channelFilterI := dsp.NewFIRFilter(channelTaps)
	channelFilterQ := dsp.NewFIRFilter(channelTaps)

	// --- Stage 2: FM Demodulator ---
	demod := dsp.NewDemodulator()

	// --- Stage 3: Audio Filtering and De-emphasis ---
	audioTaps := dsp.DesignFIRLowPass(cfg.FilterTaps, cfg.AudioFilterCutoff)
	audioFilter := dsp.NewFIRFilter(audioTaps)
	deemph := dsp.NewDeemphasis(cfg.OutputSampleRate, cfg.DeemphTau)
	var blockCounter int64
	var clippedSamples int64

	for {
		blockCounter++
		raw := rb.Read(frameSize)
		// If Read returns nil, the buffer is closed and empty, so we can exit the loop.
		if raw == nil {
			fmt.Println("Processor: End of stream, exiting.")
			break
		}

		if len(raw) < frameSize {
			continue
		}

		I := make([]float32, cfg.SampleBlockSize)
		Q := make([]float32, cfg.SampleBlockSize)

		for i := 0; i < cfg.SampleBlockSize; i++ {
			iVal := raw[2*i]
			qVal := raw[2*i+1]
			I[i] = float32(iVal) / 32768.0
			Q[i] = float32(qVal) / 32768.0
		}
		var preFilterMag float32
		for i := 0; i < cfg.SampleBlockSize; i++ {
			preFilterMag += I[i]*I[i] + Q[i]*Q[i]
		}

		// === STAGE 1: Channel Filtering and Decimation (2MHz -> 240kHz) ===
		ratioStage1 := float64(cfg.IntermediateRate) / float64(cfg.IQSampleRate)
		intermediateI := channelFilterI.Process(I, ratioStage1)
		intermediateQ := channelFilterQ.Process(Q, ratioStage1)

		if intermediateI == nil {
			continue
		}

		// Combine I and Q into complex samples for the new demodulator
		complexSamples := make([]complex64, len(intermediateI))
		for i := range intermediateI {
			complexSamples[i] = complex(intermediateI[i], intermediateQ[i])
		}

		// === STAGE 2: FM Demodulation ===
		phaseDiffs := demod.Process(complexSamples)

		// === STAGE 3: Audio Filtering and Final Resampling (240kHz -> 48kHz) ===
		ratioStage2 := float64(cfg.OutputSampleRate) / float64(cfg.IntermediateRate)
		finalAudioRaw := audioFilter.Process(phaseDiffs, ratioStage2)

		if finalAudioRaw == nil {
			continue
		}

		for i, rawSample := range finalAudioRaw {
			// The scaling factor here determines the audio volume.
			audio := deemph.Filter(float64(rawSample)) * 4000.0

			// Handle clipping
			if audio > 32767 {
				clippedSamples++
				audio = 32767
			} else if audio < -32768 {
				clippedSamples++
				audio = -32768
			}

			if blockCounter%100 == 0 && i == 1 { // Periodically print clipping stats
				if clippedSamples > 0 {
					fmt.Printf("[STATS] Total clipped samples so far: %d\n", clippedSamples)
				}
			}
			var buf [2]byte
			binary.LittleEndian.PutUint16(buf[:], uint16(int16(audio)))
			_, _ = writer.Write(buf[:])
		}
	}
}
