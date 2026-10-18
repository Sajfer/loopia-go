package loopia

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
)

const (
	testUser = "user@loopiaapi"
	testPass = "secret"
)

// rpcCall is a decoded XML-RPC request received by the fake server.
type rpcCall struct {
	Method string
	Params []any
}

// fault makes the fake server reply with an XML-RPC fault.
type fault struct {
	Code    int
	Message string
}

// rawXML makes the fake server reply with the given response body verbatim.
type rawXML string

// fixture makes the fake server reply with the contents of testdata/<name>.
func fixture(t *testing.T, name string) rawXML {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return rawXML(b)
}

// fakeLoopia is an XML-RPC server that records calls and answers each one
// with the next queued reply.
type fakeLoopia struct {
	t       *testing.T
	mu      sync.Mutex
	calls   []rpcCall
	replies []any
}

// newFake starts a fake LoopiaAPI that answers calls with replies, in order,
// and returns a client pointing at it.
func newFake(t *testing.T, replies ...any) (*Client, *fakeLoopia) {
	t.Helper()
	f := &fakeLoopia{t: t, replies: replies}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	t.Cleanup(func() {
		if len(f.replies) > 0 {
			t.Errorf("%d queued replies were not used", len(f.replies))
		}
	})
	return New(testUser, testPass, WithEndpoint(srv.URL)), f
}

func (f *fakeLoopia) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		f.t.Errorf("method = %s, want POST", r.Method)
	}
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/xml") {
		f.t.Errorf("Content-Type = %q, want text/xml", ct)
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		f.t.Errorf("read request: %v", err)
		return
	}
	call, err := decodeCall(body)
	if err != nil {
		f.t.Errorf("decode request: %v\n%s", err, body)
		return
	}

	f.mu.Lock()
	f.calls = append(f.calls, call)
	if len(f.replies) == 0 {
		f.mu.Unlock()
		f.t.Errorf("unexpected call %s", call.Method)
		http.Error(w, "no reply queued", http.StatusInternalServerError)
		return
	}
	reply := f.replies[0]
	f.replies = f.replies[1:]
	f.mu.Unlock()

	w.Header().Set("Content-Type", "text/xml")
	switch v := reply.(type) {
	case rawXML:
		io.WriteString(w, string(v))
	case fault:
		fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><methodResponse><fault>%s</fault></methodResponse>`,
			encodeValue(map[string]any{"faultCode": v.Code, "faultString": v.Message}))
	default:
		fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><methodResponse><params><param>%s</param></params></methodResponse>`,
			encodeValue(v))
	}
}

// only returns the single call received, failing the test otherwise.
func (f *fakeLoopia) only() rpcCall {
	f.t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) != 1 {
		f.t.Fatalf("got %d calls, want 1", len(f.calls))
	}
	return f.calls[0]
}

// checkCall verifies the method name and the parameters following the
// credentials.
func checkCall(t *testing.T, got rpcCall, method string, params ...any) {
	t.Helper()
	if got.Method != method {
		t.Errorf("method = %q, want %q", got.Method, method)
	}
	if len(got.Params) < 2 || got.Params[0] != testUser || got.Params[1] != testPass {
		t.Fatalf("params do not start with credentials: %#v", got.Params)
	}
	if params == nil {
		params = []any{}
	}
	if rest := got.Params[2:]; !reflect.DeepEqual(rest, params) {
		t.Errorf("%s params after credentials:\n got  %#v\n want %#v", method, rest, params)
	}
}

// encodeValue encodes v as an XML-RPC <value> element.
func encodeValue(v any) string {
	var b strings.Builder
	b.WriteString("<value>")
	switch x := v.(type) {
	case string:
		b.WriteString("<string>")
		xml.EscapeText(&b, []byte(x))
		b.WriteString("</string>")
	case int:
		fmt.Fprintf(&b, "<int>%d</int>", x)
	case int64:
		fmt.Fprintf(&b, "<int>%d</int>", x)
	case float64:
		fmt.Fprintf(&b, "<double>%s</double>", strconv.FormatFloat(x, 'f', -1, 64))
	case bool:
		if x {
			b.WriteString("<boolean>1</boolean>")
		} else {
			b.WriteString("<boolean>0</boolean>")
		}
	case []any:
		b.WriteString("<array><data>")
		for _, e := range x {
			b.WriteString(encodeValue(e))
		}
		b.WriteString("</data></array>")
	case []string:
		b.WriteString("<array><data>")
		for _, e := range x {
			b.WriteString(encodeValue(e))
		}
		b.WriteString("</data></array>")
	case map[string]any:
		b.WriteString("<struct>")
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		slices.Sort(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "<member><name>%s</name>%s</member>", k, encodeValue(x[k]))
		}
		b.WriteString("</struct>")
	default:
		panic(fmt.Sprintf("encodeValue: unsupported type %T", v))
	}
	b.WriteString("</value>")
	return b.String()
}

// xValue mirrors an XML-RPC <value> element for decoding requests.
type xValue struct {
	String  *string  `xml:"string"`
	Int     *int64   `xml:"int"`
	I4      *int64   `xml:"i4"`
	Boolean *string  `xml:"boolean"`
	Double  *float64 `xml:"double"`
	Array   *struct {
		Values []xValue `xml:"data>value"`
	} `xml:"array"`
	Struct *struct {
		Members []struct {
			Name  string `xml:"name"`
			Value xValue `xml:"value"`
		} `xml:"member"`
	} `xml:"struct"`
	Text string `xml:",chardata"`
}

func (v xValue) toGo() any {
	switch {
	case v.String != nil:
		return *v.String
	case v.Int != nil:
		return *v.Int
	case v.I4 != nil:
		return *v.I4
	case v.Boolean != nil:
		return *v.Boolean == "1"
	case v.Double != nil:
		return *v.Double
	case v.Array != nil:
		a := []any{}
		for _, e := range v.Array.Values {
			a = append(a, e.toGo())
		}
		return a
	case v.Struct != nil:
		m := map[string]any{}
		for _, mem := range v.Struct.Members {
			m[mem.Name] = mem.Value.toGo()
		}
		return m
	}
	return v.Text
}

// decodeCall decodes an XML-RPC methodCall into Go values: strings, int64s,
// bools, float64s, []any and map[string]any.
func decodeCall(body []byte) (rpcCall, error) {
	var mc struct {
		Method string   `xml:"methodName"`
		Params []xValue `xml:"params>param>value"`
	}
	if err := xml.Unmarshal(body, &mc); err != nil {
		return rpcCall{}, err
	}
	call := rpcCall{Method: mc.Method, Params: []any{}}
	for _, p := range mc.Params {
		call.Params = append(call.Params, p.toGo())
	}
	return call, nil
}
