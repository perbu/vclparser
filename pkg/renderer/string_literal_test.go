package renderer

import (
	"strings"
	"testing"

	"github.com/perbu/vclparser/pkg/parser"
)

// VCL strings carry no escape sequences: varnish's lexer copies the bytes
// between the delimiters verbatim. These tests pin that contract, which the
// renderer previously broke by re-escaping every backslash — silently turning
// "\.jpg$" into a pattern that matches a backslash followed by any character.

func TestRenderPreservesStringLiteralsVerbatim(t *testing.T) {
	tests := []struct {
		name string
		stmt string
		// want is the literal, delimiters included, that must appear
		// unchanged in the rendered output.
		want string
	}{
		{
			name: "escaped dot in a regex",
			stmt: `if (req.url ~ "\.jpg$") { return (pass); }`,
			want: `"\.jpg$"`,
		},
		{
			name: "escaped parens and plus, as in upstream devicedetect.vcl",
			stmt: `if (req.http.User-Agent ~ "\(compatible; Googlebot/2.1; \+http://www.google.com/bot.html\)") { return (pass); }`,
			want: `"\(compatible; Googlebot/2.1; \+http://www.google.com/bot.html\)"`,
		},
		{
			name: "character class shorthand",
			stmt: `if (req.url ~ "^/v\d+/") { return (pass); }`,
			want: `"^/v\d+/"`,
		},
		{
			name: "trailing backslash",
			stmt: `if (req.url ~ "c:\\") { return (pass); }`,
			want: `"c:\\"`,
		},
		{
			name: "no escapes at all",
			stmt: `if (req.url ~ "^/api/") { return (pass); }`,
			want: `"^/api/"`,
		},
		{
			name: "empty string",
			stmt: `set req.http.X-Empty = "";`,
			want: `""`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := wrapInSub(tt.stmt)
			program, err := parser.Parse(src, "test.vcl")
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			rendered := Render(program)
			if !strings.Contains(rendered, tt.want) {
				t.Errorf("rendered output lost the literal\n  want substring: %s\n  rendered:\n%s", tt.want, rendered)
			}
		})
	}
}

func TestRenderPreservesLongStrings(t *testing.T) {
	src := `vcl 4.1;

sub vcl_synth {
    synthetic( {"<html>
<body>Not found</body>
</html>"} );
}
`
	program, err := parser.Parse(src, "test.vcl")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	rendered := Render(program)

	// The long form must survive: the content spans lines and so cannot be
	// represented as a "..." string at all.
	if !strings.Contains(rendered, `{"`) || !strings.Contains(rendered, `"}`) {
		t.Errorf("long-string delimiters were dropped:\n%s", rendered)
	}
	if strings.Contains(rendered, `\n`) {
		t.Errorf("newlines were escaped into \\n, which VCL does not decode:\n%s", rendered)
	}
	if !strings.Contains(rendered, "<body>Not found</body>") {
		t.Errorf("long-string content was altered:\n%s", rendered)
	}
}

// TestRenderIsIdempotent is the general guard. Any escaping applied on the way
// out that the parser does not undo on the way back in compounds with every
// cycle, so a single extra round trip exposes it: the old renderer turned
// "\.jpg$" into "\\.jpg$" and then "\\\\.jpg$".
func TestRenderIsIdempotent(t *testing.T) {
	sources := map[string]string{
		"regex escapes": wrapInSub(`if (req.url ~ "\.jpg$" && req.http.host ~ "^www\.example\.com$") { return (pass); }`),
		"long string": `vcl 4.1;

sub vcl_synth {
    synthetic( {"line one
line two"} );
}
`,
		"backend with quotes in a header": wrapInSub(`set req.http.X-Thing = "a/b\c";`),
		"acl and backend": `vcl 4.1;

acl purgers {
    "127.0.0.1";
    "192.168.0.0"/24;
}

backend default {
    .host = "127.0.0.1";
    .port = "8080";
}
`,
	}

	for name, src := range sources {
		t.Run(name, func(t *testing.T) {
			first, err := renderOnce(src, "test.vcl")
			if err != nil {
				t.Fatalf("first render: %v", err)
			}
			second, err := renderOnce(first, "rendered.vcl")
			if err != nil {
				t.Fatalf("re-parsing our own output failed: %v\noutput:\n%s", err, first)
			}
			if first != second {
				t.Errorf("render is not idempotent\n--- first ---\n%s\n--- second ---\n%s", first, second)
			}
		})
	}
}

func TestFormatStringLiteral(t *testing.T) {
	tests := []struct {
		name       string
		value      string
		preferLong bool
		want       string
	}{
		{name: "plain short", value: `abc`, want: `"abc"`},
		{name: "backslash is not escaped", value: `\.jpg$`, want: `"\.jpg$"`},
		{name: "empty", value: ``, want: `""`},
		{name: "prefers long when asked", value: `abc`, preferLong: true, want: `{"abc"}`},
		{
			name:  "embedded quote forces long form",
			value: `say "hi"`,
			want:  `{"say "hi""}`,
		},
		{
			name:  "newline forces long form",
			value: "one\ntwo",
			want:  "{\"one\ntwo\"}",
		},
		{
			name:  "carriage return forces long form",
			value: "one\r\ntwo",
			want:  "{\"one\r\ntwo\"}",
		},
		{
			name:       "long form declined when the value closes the delimiter",
			value:      `ends with "}`,
			preferLong: true,
			want:       `ends with "}`, // unrepresentable; checked loosely below
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatStringLiteral(tt.value, tt.preferLong)

			if tt.name == "long form declined when the value closes the delimiter" {
				// No VCL delimiter can carry this value. Assert only that we
				// do not silently corrupt the content.
				if !strings.Contains(got, tt.value) {
					t.Errorf("content was altered: got %q, want it to contain %q", got, tt.value)
				}
				return
			}

			if got != tt.want {
				t.Errorf("FormatStringLiteral(%q, %v) = %q, want %q", tt.value, tt.preferLong, got, tt.want)
			}
		})
	}
}

func wrapInSub(stmt string) string {
	return "vcl 4.1;\n\nsub vcl_recv {\n    " + stmt + "\n}\n"
}

func renderOnce(src, filename string) (string, error) {
	program, err := parser.Parse(src, filename)
	if err != nil {
		return "", err
	}
	return Render(program), nil
}
