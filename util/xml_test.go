package util_test

import (
	"testing"

	"github.com/mook/fanficupdates/util"
	"github.com/stretchr/testify/require"
)

func TestPrettyXML(t *testing.T) {
	cases := []struct{ name, input, expected string }{
		{
			name:     "simple",
			input:    "<elem/>",
			expected: "<elem></elem>",
		},
		{
			name:     "whitespace",
			input:    "<elem>  \r\n  </elem>",
			expected: "<elem></elem>",
		},
		{
			name:     "chardata",
			input:    "<elem>  text  </elem>",
			expected: "<elem>text</elem>",
		},
		{
			name:     "mixed",
			input:    "<elem>  text <child/> more  </elem>",
			expected: "<elem>text\n  <child></child>more\n</elem>",
		},
		{
			name:     "namespace",
			input:    `<elem xmlns="foo"><child xmlns="bar"/></elem>`,
			expected: "<elem xmlns=\"foo\">\n  <child xmlns=\"bar\"></child>\n</elem>",
		},
	}
	for _, testcase := range cases {
		t.Run(testcase.name, func(t *testing.T) {
			result, err := util.PrettyXML([]byte(testcase.input))
			require.NoError(t, err)
			require.Equal(t, testcase.expected, string(result))
		})
	}
}
