package toolkit

import "testing"

func TestTools_RandomString(t *testing.T) {
	var tools Tools

	testcase := []struct {
		name           string
		input          int
		expectedLength int
	}{
		{
			name:           "zero length",
			input:          0,
			expectedLength: 0,
		},
		{
			name:           "100 length",
			input:          100,
			expectedLength: 100,
		},
	}

	for _, tc := range testcase {
		got := tools.RandomString(tc.input)

		if len(got) != tc.expectedLength {
			t.Errorf("expecting RandomString to return a string with length of %d, got %d", tc.expectedLength, len(got))
		}
	}
}
