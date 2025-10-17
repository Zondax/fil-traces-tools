package internal

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadAddressFile(t *testing.T) {
	tests := []struct {
		name        string
		fileContent string
		expected    []string
		expectError bool
	}{
		{
			name: "valid addresses with newlines",
			fileContent: `f1234
f5678
f9012`,
			expected: []string{"f1234", "f5678", "f9012"},
		},
		{
			name: "addresses with empty lines",
			fileContent: `f1234

f5678

f9012`,
			expected: []string{"f1234", "f5678", "f9012"},
		},
		{
			name: "addresses with whitespace",
			fileContent: `  f1234  
	f5678	
f9012   `,
			expected: []string{"f1234", "f5678", "f9012"},
		},
		{
			name:        "single address",
			fileContent: `f1234`,
			expected:    []string{"f1234"},
		},
		{
			name:        "empty file",
			fileContent: ``,
			expected:    []string{},
		},
		{
			name: "only whitespace and newlines",
			fileContent: `
  
		
`,
			expected: []string{},
		},
		{
			name: "addresses with trailing newline",
			fileContent: `f1234
f5678
`,
			expected: []string{"f1234", "f5678"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "addresses.txt")
			err := os.WriteFile(tmpFile, []byte(tt.fileContent), 0600)
			require.NoError(t, err)

			// Test the function
			addresses, err := ReadAddressFile(tmpFile)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expected, addresses)
			}
		})
	}

	t.Run("file not found", func(t *testing.T) {
		_, err := ReadAddressFile("/non/existent/file.txt")
		assert.Error(t, err)
	})
}
