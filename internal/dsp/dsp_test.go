package dsp

import (
	"testing"
)

// TestDesignFIRLowPass checks the properties of the generated FIR filter.
func TestDesignFIRLowPass(t *testing.T) {
	const numTaps = 51
	const cutoff = 0.1

	taps := DesignFIRLowPass(numTaps, cutoff)

	if len(taps) != numTaps {
		t.Fatalf("Expected %d taps, but got %d", numTaps, len(taps))
	}

	// 1. Check for symmetry (property of linear-phase FIR filters)
	for i := 0; i < numTaps/2; i++ {
		if !almostEqual(float32(taps[i]), float32(taps[numTaps-1-i])) {
			t.Errorf("Filter is not symmetric. Tap %d (%f) != Tap %d (%f)", i, taps[i], numTaps-1-i, taps[numTaps-1-i])
		}
	}

	// 2. Check that the sum of taps is 1.0 (for DC gain of 1)
	var sum float64
	for _, tap := range taps {
		sum += tap
	}
	if !almostEqual(float32(sum), 1.0) {
		t.Errorf("Expected sum of taps to be 1.0, but got %f", sum)
	}
}

// TestFIRFilter_DecimationAndState checks the decimating filter.
func TestFIRFilter_DecimationAndState(t *testing.T) {
	taps := []float64{0.1, 0.2, 0.4, 0.2, 0.1}
	ratio := 0.5 // Decimate by 2

	input := make([]float32, 100)
	for i := range input {
		input[i] = float32(i)
	}

	// Process in one go
	fir1 := NewFIRFilter(taps)
	fullOutput := fir1.Process(input, ratio)

	// Process in chunks
	fir2 := NewFIRFilter(taps)
	chunk1 := fir2.Process(input[:50], ratio)
	chunk2 := fir2.Process(input[50:], ratio)
	chunkedOutput := append(chunk1, chunk2...)

	if len(fullOutput) != len(chunkedOutput) {
		t.Fatalf("Mismatched lengths: full=%d, chunked=%d", len(fullOutput), len(chunkedOutput))
	}

	for i := range fullOutput {
		if !almostEqual(fullOutput[i], chunkedOutput[i]) {
			t.Errorf("Mismatch at index %d: full=%f, chunked=%f", i, fullOutput[i], chunkedOutput[i])
		}
	}
}

// TestDeemphasis checks the de-emphasis filter's response to a step input.
func TestDeemphasis(t *testing.T) {
	const sampleRate = 48000
	const tau = 50e-6 // 50us

	deemph := NewDeemphasis(sampleRate, tau)

	// Apply a step input (a constant value of 1.0)
	input := 1.0
	var lastOutput float64

	// The output should be an exponential curve approaching 1.0
	// It should always be increasing and never exceed the input value.
	for i := 0; i < 100; i++ {
		output := deemph.Filter(input)
		if i > 0 {
			if output < lastOutput {
				t.Fatalf("De-emphasis output decreased on step input at sample %d", i)
			}
		}
		if output > input {
			t.Fatalf("De-emphasis output exceeded input value at sample %d", i)
		}
		lastOutput = output
	}

	// After many samples, it should be very close to the final value.
	for i := 0; i < sampleRate; i++ { // Run for 1s
		deemph.Filter(input)
	}

	finalOutput := deemph.Filter(input)
	if !almostEqual(float32(finalOutput), 1.0) {
		t.Errorf("Expected de-emphasis to settle near 1.0, but got %f", finalOutput)
	}
}
