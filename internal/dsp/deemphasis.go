package dsp

// Deemphasis implements a first-order low-pass filter for FM de-emphasis.
type Deemphasis struct {
	alpha float64
	prev  float64
}

// NewDeemphasis creates a new de-emphasis filter.
// sampleRate is the audio sample rate.
// tau is the time constant (e.g., 50e-6 for Europe, 75e-6 for US).
func NewDeemphasis(sampleRate int, tau float64) *Deemphasis {
	dt := 1.0 / float64(sampleRate)
	alpha := dt / (tau + dt)
	return &Deemphasis{alpha: alpha}
}

// Filter applies the de-emphasis filter to a single sample.
func (d *Deemphasis) Filter(x float64) float64 {
	d.prev += d.alpha * (x - d.prev)
	return d.prev
}
