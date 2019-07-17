package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		input    string
		expected []*evalCode
	}{
		{
			input: "!eval\n```\nprint('test')\n```\n",
			expected: []*evalCode{
				{
					language: "",
					contents: "print('test')\n",
				},
			},
		},
		{
			input: "!eval\n```py\nprint('test')\n```\n",
			expected: []*evalCode{
				{
					language: "py",
					contents: "print('test')\n",
				},
			},
		},
		{
			input: "__strong__\n!eval\n```py\nprint('test')\n```\n",
			expected: []*evalCode{
				{
					language: "py",
					contents: "print('test')\n",
				},
			},
		},
		{
			input: "foo bar\n!eval\n```py\nprint('test')\n```\n(baz)",
			expected: []*evalCode{
				{
					language: "py",
					contents: "print('test')\n",
				},
			},
		},
	}

	for _, tc := range testCases {
		resp, err := parseForBot(tc.input)
		require.Nil(t, err)
		require.Equal(t, tc.expected, resp)
	}
}
