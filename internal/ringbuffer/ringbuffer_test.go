package ringbuffer

import (
	"sync"
	"testing"
)

func TestRingBuffer_ConcurrentReadWrite(t *testing.T) {
	// Use a large number of samples to ensure goroutines have to wait for each other,
	// forcing the wait conditions in Read and Write to be exercised.
	const totalSamples = 200000
	const bufferSize = 8192
	const writeChunkSize = 256
	const readChunkSize = 192 // Use different, non-aligned chunk sizes to stress test the logic.

	// We are removing the debug prints for the test to avoid flooding the test output.
	// If you need to debug the test itself, you can re-enable them temporarily.
	// originalFmtPrintf := fmt.Printf
	// fmt.Printf = func(format string, a ...interface{}) (n int, err error) {
	// 	return 0, nil
	// }
	// defer func() {
	// 	fmt.Printf = originalFmtPrintf
	// }()

	rb := New(bufferSize)

	// Generate the source data that the writer will send.
	// Using sequential numbers makes it easy to verify correctness later.
	sourceData := make([]int16, totalSamples)
	for i := 0; i < totalSamples; i++ {
		sourceData[i] = int16(i)
	}

	// This slice will hold the data the reader receives.
	// It's protected by a mutex because it's written to from the reader goroutine.
	destData := make([]int16, 0, totalSamples)
	var destMutex sync.Mutex

	var wg sync.WaitGroup
	wg.Add(2)

	// --- Writer Goroutine ---
	go func() {
		defer wg.Done()
		writtenCount := 0
		for writtenCount < totalSamples {
			end := writtenCount + writeChunkSize
			if end > totalSamples {
				end = totalSamples
			}
			chunk := sourceData[writtenCount:end]
			rb.Write(chunk)
			writtenCount = end
		}
		// Signal that the writer is done.
		rb.Close()
	}()

	// --- Reader Goroutine ---
	go func() {
		defer wg.Done()
		readCount := 0
		for readCount < totalSamples {
			chunk := rb.Read(readChunkSize)
			// If the chunk is nil, the buffer is closed and empty.
			if chunk == nil {
				break
			}

			destMutex.Lock()
			destData = append(destData, chunk...)
			destMutex.Unlock()

			readCount += len(chunk)
		}
	}()

	// Wait for both the reader and writer to finish their work.
	wg.Wait()

	// --- Verification ---
	if len(destData) != totalSamples {
		t.Fatalf("Data loss detected: expected %d samples, but got %d", totalSamples, len(destData))
	}

	for i := 0; i < totalSamples; i++ {
		if sourceData[i] != destData[i] {
			t.Fatalf("Data corruption at index %d: expected %d, but got %d", i, sourceData[i], destData[i])
		}
	}
}
