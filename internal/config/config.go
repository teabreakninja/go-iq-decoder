package config

// Config holds all the configuration parameters for the application.
type Config struct {
	IQSampleRate        int
	IntermediateRate    int
	OutputSampleRate    int
	SampleBlockSize     int
	FilterTaps          int
	RingBufferSize      int
	ChunkSize           int
	ChannelFilterCutoff float64
	AudioFilterCutoff   float64
	DeemphTau           float64
}

// New returns a new Config with default values.
func New() *Config {
	return &Config{
		IQSampleRate:        2_000_000,
		IntermediateRate:    240_000,
		OutputSampleRate:    48_000,
		SampleBlockSize:     4096,
		FilterTaps:          251,
		RingBufferSize:      2 * 2_000_000 * 2, // 2s of IQ (I+Q)
		ChunkSize:           8192,
		ChannelFilterCutoff: 100000.0 / float64(2_000_000),
		AudioFilterCutoff:   15000.0 / float64(240_000),
		DeemphTau:           50e-6, // 50us for Europe
	}
}
