package westafrica

import (
	"errors"
	"testing"
)

func TestParsePixelStatusBody_object(t *testing.T) {
	body := []byte(`{
		"data": {
			"transaction_id": "PIX_37068765",
			"amount": 5000,
			"state": "SUCCESSFUL",
			"currency": "XOF"
		},
		"message": "",
		"statut_code": 200
	}`)
	resp, err := parsePixelStatusBody(body, "PIX_37068765")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Data.TransactionID != "PIX_37068765" || resp.Data.State != "SUCCESSFUL" {
		t.Fatalf("unexpected: %+v", resp.Data)
	}
}

func TestParsePixelStatusBody_arrayNotFound(t *testing.T) {
	body := []byte(`{
		"data": ["PIX_22827809"],
		"message": "transaction not found",
		"statut_code": 404
	}`)
	_, err := parsePixelStatusBody(body, "PIX_22827809")
	if !errors.Is(err, ErrPixelTransactionNotFound) {
		t.Fatalf("want ErrPixelTransactionNotFound, got %v", err)
	}
}

func TestParsePixelStatusBody_arrayObjects(t *testing.T) {
	body := []byte(`{
		"data": [{
			"transaction_id": "PIX_22827809",
			"amount": 100,
			"state": "PENDING1",
			"currency": "XOF"
		}],
		"statut_code": 200
	}`)
	resp, err := parsePixelStatusBody(body, "PIX_22827809")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Data.State != "PENDING1" {
		t.Fatalf("state=%q", resp.Data.State)
	}
}
