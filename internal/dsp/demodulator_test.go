package dsp

import (
	"math"
	"testing"
)

const float32EqualityThreshold = 1e-6

func almostEqual(a, b float32) bool {
	return math.Abs(float64(a-b)) <= float32EqualityThreshold
}

// generateTestSignal creates a complex signal with a constant phase rotation.
func generateTestSignal(numSamples int, phaseIncrement float64) []complex64 {
	samples := make([]complex64, numSamples)
	for i := 0; i < numSamples; i++ {
		// e^(j*theta) = cos(theta) + j*sin(theta)
		phase := float64(i+1) * phaseIncrement
		samples[i] = complex(float32(math.Cos(phase)), float32(math.Sin(phase)))
	}
	return samples
}

func TestDemodulator_ConstantFrequency(t *testing.T) {
	demod := NewDemodulator()

	const numSamples = 128
	const phaseIncrement = math.Pi / 16 // Represents a constant frequency offset

	// Generate a signal with a constant phase rotation.
	samples := generateTestSignal(numSamples, phaseIncrement)
	output := demod.Process(samples)

	if len(output) != numSamples {
		t.Fatalf("Expected output length of %d, but got %d", numSamples, len(output))
	}

	// The first sample is compared against the zero-state, so we skip it.
	// All subsequent samples should have a phase difference equal to our increment.
	for i := 1; i < len(output); i++ {
		if !almostEqual(output[i], float32(phaseIncrement)) {
			t.Errorf("Sample %d: expected phase difference of %f, but got %f", i+1, phaseIncrement, output[i])
		}
	}
}

func TestDemodulator_PhaseWrapAround(t *testing.T) {
	demod := NewDemodulator()

	// Create a signal with a large phase jump that will wrap around.
	// A jump from +0.75π to -0.75π is a total change of -1.5π.
	// The cmplx.Phase function should report this as +0.5π.
	const phaseBeforeJump = 0.75 * math.Pi
	const phaseAfterJump = -0.75 * math.Pi
	const expectedWrappedPhase = 0.5 * math.Pi

	samples := []complex64{
		complex(float32(math.Cos(0)), float32(math.Sin(0))),                             // Sample 0: Phase = 0
		complex(float32(math.Cos(phaseBeforeJump)), float32(math.Sin(phaseBeforeJump))), // Sample 1: Phase = +0.75π
		complex(float32(math.Cos(phaseAfterJump)), float32(math.Sin(phaseAfterJump))),   // Sample 2: Phase = -0.75π
	}

	output := demod.Process(samples)

	if len(output) < 3 {
		t.Fatalf("Expected at least 3 output samples, got %d", len(output))
	}

	// output[1] is the phase diff between samples[1] and samples[0]
	if !almostEqual(output[1], float32(phaseBeforeJump)) {
		t.Errorf("Expected phase diff at output[1] to be %f, but got %f", phaseBeforeJump, output[1])
	}

	// output[2] is the phase diff between samples[2] and samples[1], which should wrap.
	if !almostEqual(output[2], float32(expectedWrappedPhase)) {
		t.Errorf("Expected wrapped phase diff at output[2] to be %f, but got %f", expectedWrappedPhase, output[2])
	}
}

func TestDemodulator_Statefulness(t *testing.T) {
	const numSamples = 256
	const phaseIncrement = -math.Pi / 8
	const chunkSize = 64

	// Generate a full signal for reference.
	fullSignal := generateTestSignal(numSamples, phaseIncrement)

	// --- Process the signal in one go ---
	referenceDemod := NewDemodulator()
	referenceOutput := referenceDemod.Process(fullSignal)

	// --- Process the signal in chunks and verify statefulness ---
	chunkedDemod := NewDemodulator()
	chunkedOutput := make([]float32, 0, numSamples)

	for i := 0; i < numSamples; i += chunkSize {
		end := i + chunkSize
		if end > numSamples {
			end = numSamples
		}
		chunk := fullSignal[i:end]
		outputChunk := chunkedDemod.Process(chunk)
		chunkedOutput = append(chunkedOutput, outputChunk...)
	}

	// --- Compare the results ---
	if len(referenceOutput) != len(chunkedOutput) {
		t.Fatalf("Mismatched output lengths: reference=%d, chunked=%d", len(referenceOutput), len(chunkedOutput))
	}

	// The first sample of the first chunk will differ due to initial state.
	// All subsequent samples should be identical.
	for i := 1; i < len(referenceOutput); i++ {
		if !almostEqual(referenceOutput[i], chunkedOutput[i]) {
			t.Fatalf("Mismatch at sample %d: reference=%f, chunked=%f", i, referenceOutput[i], chunkedOutput[i])
		}
	}
}
