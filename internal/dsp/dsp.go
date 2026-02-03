package dsp

import "math"

// DesignFIRLowPass creates a low-pass FIR filter using the windowed-sinc method.
func DesignFIRLowPass(numTaps int, cutoff float64) []float64 {
	taps := make([]float64, numTaps)
	M := float64(numTaps - 1)
	// The cutoff frequency must be normalized to the Nyquist frequency (0.5 * sample_rate)
	fc := cutoff * 2
	for n := 0; n < numTaps; n++ {
		x := float64(n) - M/2
		if x == 0 {
			taps[n] = fc
		} else {
			taps[n] = fc * math.Sin(math.Pi*fc*x) / (math.Pi * fc * x)
		}
		// Apply Hamming window
		taps[n] *= 0.54 - 0.46*math.Cos(2*math.Pi*float64(n)/M)
	}
	// Normalize
	sum := 0.0
	for _, t := range taps {
		sum += t
	}
	for i := range taps {
		taps[i] /= sum
	}
	return taps
}

// Resample changes the sample rate of a signal using a windowed-sinc function.
func Resample(input []float32, ratio float64) []float32 {
	const windowSize = 16 // Number of taps on each side of the sample.

	outputLen := int(float64(len(input)) * ratio)
	if outputLen == 0 {
		return nil
	}
	output := make([]float32, outputLen)
	invRatio := 1.0 / ratio

	for i := range output {
		inPos := float64(i) * invRatio
		centerIndex := int(math.Round(inPos))

		var acc, sumTaps float32
		for j := -windowSize; j < windowSize; j++ {
			inputIndex := centerIndex + j
			if inputIndex < 0 || inputIndex >= len(input) {
				continue
			}

			sincPos := inPos - float64(inputIndex)
			piSincPos := math.Pi * sincPos
			sinc := float32(1.0)
			if piSincPos != 0 {
				sinc = float32(math.Sin(piSincPos) / piSincPos)
			}

			window := 0.54 - 0.46*math.Cos(2*math.Pi*float64(j+windowSize)/float64(2*windowSize))
			tap := sinc * float32(window)

			acc += input[inputIndex] * tap
			sumTaps += tap
		}
		if sumTaps != 0 {
			output[i] = acc / sumTaps
		}
	}
	return output
}
