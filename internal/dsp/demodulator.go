package dsp

import "math/cmplx"

// Demodulator implements a polar discriminator for FM demodulation.
type Demodulator struct {
	prev complex64
}

// NewDemodulator creates a new FM demodulator.
func NewDemodulator() *Demodulator {
	return &Demodulator{}
}

// Process demodulates a block of complex IQ samples into an audio signal.
func (d *Demodulator) Process(samples []complex64) []float32 {
	if len(samples) == 0 {
		return nil
	}
	// The output will have the same number of samples as the input. The first
	// output sample is the phase difference between the first input sample and
	// the state from the previous block.
	output := make([]float32, len(samples))
	prev := d.prev

	for i, current := range samples {
		// Multiply the current sample by the conjugate of the previous one.
		// The angle of the resulting complex number is the phase difference.
		prevConjugate := complex(real(prev), -imag(prev))
		p := current * prevConjugate
		output[i] = float32(cmplx.Phase(complex128(p)))
		prev = current
	}

	// Save the last sample of the current block for the next call.
	d.prev = prev
	return output
}
