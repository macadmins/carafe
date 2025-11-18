package shell

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPrependingWriter(t *testing.T) {
	buf := &bytes.Buffer{}
	prefix := "[TEST] "

	writer := NewPrependingWriter(buf, prefix)

	assert.NotNil(t, writer)
	assert.IsType(t, &prependingWriter{}, writer)
}

func TestPrependingWriter_Write(t *testing.T) {
	tests := []struct {
		name           string
		prefix         string
		input          string
		expectedOutput string
		expectedN      int
	}{
		{
			name:           "single line",
			prefix:         "[TEST] ",
			input:          "hello world\n",
			expectedOutput: "[TEST] hello world\n",
			expectedN:      12,
		},
		{
			name:           "multiple lines",
			prefix:         "[INFO] ",
			input:          "line1\nline2\nline3\n",
			expectedOutput: "[INFO] line1\n[INFO] line2\n[INFO] line3\n",
			expectedN:      18,
		},
		{
			name:           "line without trailing newline",
			prefix:         "> ",
			input:          "single line",
			expectedOutput: "> single line\n",
			expectedN:      11,
		},
		{
			name:           "empty prefix",
			prefix:         "",
			input:          "hello\n",
			expectedOutput: "hello\n",
			expectedN:      6,
		},
		{
			name:           "empty input",
			prefix:         "[TEST] ",
			input:          "",
			expectedOutput: "",
			expectedN:      0,
		},
		{
			name:           "only newlines",
			prefix:         "[TEST] ",
			input:          "\n\n\n",
			expectedOutput: "",
			expectedN:      3,
		},
		{
			name:           "line with leading/trailing whitespace",
			prefix:         "[TEST] ",
			input:          "  hello world  \n",
			expectedOutput: "[TEST] hello world\n",
			expectedN:      16,
		},
		{
			name:           "carriage return converted to newline",
			prefix:         "[TEST] ",
			input:          "line1\rline2\r",
			expectedOutput: "[TEST] line1\n[TEST] line2\n",
			expectedN:      12,
		},
		{
			name:           "mixed newlines and carriage returns",
			prefix:         "[LOG] ",
			input:          "start\r\nmiddle\nend\r",
			expectedOutput: "[LOG] start\n[LOG] middle\n[LOG] end\n",
			expectedN:      18,
		},
		{
			name:           "blank lines skipped",
			prefix:         "[TEST] ",
			input:          "line1\n\nline2\n",
			expectedOutput: "[TEST] line1\n[TEST] line2\n",
			expectedN:      13,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			writer := NewPrependingWriter(buf, tt.prefix)

			n, err := writer.Write([]byte(tt.input))

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedN, n)
			assert.Equal(t, tt.expectedOutput, buf.String())
		})
	}
}

func TestPrependingWriter_WriteWithAnsiCodes(t *testing.T) {
	tests := []struct {
		name           string
		prefix         string
		input          string
		expectedOutput string
	}{
		{
			name:           "strips ANSI color codes",
			prefix:         "[TEST] ",
			input:          "\033[31mred text\033[0m\n",
			expectedOutput: "[TEST] red text\n",
		},
		{
			name:           "strips ANSI bold codes",
			prefix:         "[LOG] ",
			input:          "\033[1mbold\033[0m\n",
			expectedOutput: "[LOG] bold\n",
		},
		{
			name:           "multiple lines with ANSI codes",
			prefix:         "> ",
			input:          "\033[32mgreen\033[0m\n\033[33myellow\033[0m\n",
			expectedOutput: "> green\n> yellow\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			writer := NewPrependingWriter(buf, tt.prefix)

			_, err := writer.Write([]byte(tt.input))

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedOutput, buf.String())
		})
	}
}

func TestPrependingWriter_MultipleWrites(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := NewPrependingWriter(buf, "[TEST] ")

	// First write
	n1, err1 := writer.Write([]byte("first line\n"))
	assert.NoError(t, err1)
	assert.Equal(t, 11, n1)

	// Second write
	n2, err2 := writer.Write([]byte("second line\n"))
	assert.NoError(t, err2)
	assert.Equal(t, 12, n2)

	// Third write
	n3, err3 := writer.Write([]byte("third line\n"))
	assert.NoError(t, err3)
	assert.Equal(t, 11, n3)

	expected := "[TEST] first line\n[TEST] second line\n[TEST] third line\n"
	assert.Equal(t, expected, buf.String())
}

func TestStripLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "text with leading whitespace",
			input:    "   hello",
			expected: "hello",
		},
		{
			name:     "text with trailing whitespace",
			input:    "world   ",
			expected: "world",
		},
		{
			name:     "text with both leading and trailing whitespace",
			input:    "  hello world  ",
			expected: "hello world",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only whitespace",
			input:    "   ",
			expected: "",
		},
		{
			name:     "text with ANSI color code",
			input:    "\033[31mred text\033[0m",
			expected: "red text",
		},
		{
			name:     "text with ANSI bold code",
			input:    "\033[1mbold text\033[0m",
			expected: "bold text",
		},
		{
			name:     "text with ANSI codes and whitespace",
			input:    "  \033[32mgreen\033[0m  ",
			expected: "green",
		},
		{
			name:     "tabs",
			input:    "\thello\t",
			expected: "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripLine(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
