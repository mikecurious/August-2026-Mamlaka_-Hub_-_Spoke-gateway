package merchants

import (
	"encoding/json"
	"testing"
)

// TestMpesaCallbackResultCode covers the shapes Safaricom actually sends.
// The "E3008" case is the one that caused failed LIPAD collections to be
// written COMPLETE: the old `v.(float64)` discarded the comma-ok, produced 0,
// and 0 was read as success.
func TestMpesaCallbackResultCode(t *testing.T) {
	cases := []struct {
		name     string
		in       interface{}
		wantCode float64
		wantOK   bool
	}{
		{"json number success", float64(0), 0, true},
		{"json number failure", float64(1032), 1032, true},
		{"numeric string success", "0", 0, true},
		{"numeric string failure", "1032", 1032, true},
		{"numeric string padded", "  1037 ", 1037, true},
		{"alphanumeric code", "E3008", 0, false},
		{"empty string", "", 0, false},
		{"missing", nil, 0, false},
		{"unexpected type", []interface{}{1}, 0, false},
		{"json.Number success", json.Number("0"), 0, true},
		{"json.Number failure", json.Number("2001"), 2001, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, _, ok := mpesaCallbackResultCode(tc.in)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if ok && code != tc.wantCode {
				t.Fatalf("code = %v, want %v", code, tc.wantCode)
			}
		})
	}
}

// TestResultCodeE3008IsNotSuccess asserts the guard the handlers rely on:
// a non-numeric code must never satisfy `resultCodeOK && resultCode == 0`.
func TestResultCodeE3008IsNotSuccess(t *testing.T) {
	raw := `{"Body":{"stkCallback":{"ResultCode":"E3008","ResultDesc":"Error, the user has a bad debt contract."}}}`

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	body := payload["Body"].(map[string]interface{})
	stkCallback := body["stkCallback"].(map[string]interface{})

	code, rawCode, ok := mpesaCallbackResultCode(stkCallback["ResultCode"])
	if ok && code == 0 {
		t.Fatalf("E3008 treated as success (code=%v raw=%q) - this is the bug", code, rawCode)
	}
	if rawCode != "E3008" {
		t.Fatalf("raw = %q, want E3008", rawCode)
	}
}
