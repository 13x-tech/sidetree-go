package sidetree

import "testing"

// TestProtocolParameters pins the consensus-critical Sidetree v1 constants to
// their canonical values (decentralized-identity/sidetree v1.0.6
// protocol-parameters.json). A change here changes which anchors we accept
// relative to the canonical ION network, so it must be deliberate.
func TestProtocolParameters(t *testing.T) {
	cases := []struct {
		name string
		got  int
		want int
	}{
		{"MaxOperationsPerBatch", MaxOperationsPerBatch, 10000},
		{"MaxNumberOfOperationsForNoValueTimeLock", MaxNumberOfOperationsForNoValueTimeLock, 100},
		{"ValueTimeLockAmountMultiplier", ValueTimeLockAmountMultiplier, 60000},
		{"MaxDeltaSizeInBytes", MaxDeltaSizeInBytes, 1000},
		{"MaxCASURILength", MaxCASURILength, 100},
		{"MaxEncodedRevealValueLength", MaxEncodedRevealValueLength, 50},
		{"MaxWriterLockIDInBytes", MaxWriterLockIDInBytes, 200},
		{"MaxCoreIndexFileSizeInBytes", MaxCoreIndexFileSizeInBytes, 1000000},
		{"MaxProvisionalIndexFileSizeInBytes", MaxProvisionalIndexFileSizeInBytes, 1000000},
		{"MaxProofFileSizeInBytes", MaxProofFileSizeInBytes, 2500000},
		{"MaxChunkFileSizeInBytes", MaxChunkFileSizeInBytes, 10000000},
		{"MaxMemoryDecompressionFactor", MaxMemoryDecompressionFactor, 3},
		{"MaxNumberOfTransactionsPerTransactionTime", MaxNumberOfTransactionsPerTransactionTime, 300},
		{"MaxNumberOfOperationsPerTransactionTime", MaxNumberOfOperationsPerTransactionTime, 600000},
		{"SHA256MultihashCode", SHA256MultihashCode, 18},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
	if NormalizedFeeToPerOperationFeeMultiplier != 0.001 {
		t.Errorf("NormalizedFeeToPerOperationFeeMultiplier = %v, want 0.001", NormalizedFeeToPerOperationFeeMultiplier)
	}
}
