package dsp

// FIRFilter implements a stateful, block-based Finite Impulse Response filter.
type FIRFilter struct {
	taps  []float64
	state []float32
}

// NewFIRFilter creates a new FIR filter with the given taps.
func NewFIRFilter(taps []float64) *FIRFilter {
	return &FIRFilter{
		taps:  taps,
		state: make([]float32, len(taps)-1),
	}
}

// Process filters a block of input samples and updates the filter's internal state.
func (f *FIRFilter) Process(input []float32, ratio float64) []float32 {
	invRatio := 1.0 / ratio

	buffer := make([]float32, len(f.state)+len(input))
	copy(buffer, f.state)
	copy(buffer[len(f.state):], input)

	// This is the correct, conservative calculation for the number of output samples
	// that can be safely produced from the given buffer.
	outputLen := int(float64(len(buffer)-len(f.taps)+1) * ratio)
	if outputLen <= 0 {
		f.state = buffer // Not enough data, save for next time
		return nil
	}
	output := make([]float32, outputLen)

	for i := 0; i < outputLen; i++ {
		inPos := float64(i) * invRatio
		start := int(inPos)

		var acc float32
		for j, tap := range f.taps {
			acc += buffer[start+j] * float32(tap)
		}
		output[i] = acc
	}

	// The state for the next run is the last (filter_length - 1) samples of the buffer.
	f.state = buffer[len(buffer)-(len(f.taps)-1):]
	return output
}
