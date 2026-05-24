package testutil

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Request faz uma chamada HTTP autenticada contra o httptest.Server e
// devolve status + body. O token vai como `Authorization: Bearer ...`
// (passe "" para anônimo). `body` pode ser nil, []byte ou qualquer
// estrutura JSON-serializável.
func Request(t *testing.T, srv *httptest.Server, method, path, token string, body interface{}) (status int, respBody []byte) {
	t.Helper()
	var bodyReader io.Reader
	if body != nil {
		switch b := body.(type) {
		case []byte:
			bodyReader = bytes.NewReader(b)
		case string:
			bodyReader = bytes.NewReader([]byte(b))
		default:
			j, err := json.Marshal(body)
			if err != nil {
				t.Fatalf("marshal body: %v", err)
			}
			bodyReader = bytes.NewReader(j)
		}
	}
	req, err := http.NewRequest(method, srv.URL+path, bodyReader)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := srv.Client().Do(req)
	if err != nil {
		t.Fatalf("do request: %v", err)
	}
	defer resp.Body.Close()
	respBody, _ = io.ReadAll(resp.Body)
	return resp.StatusCode, respBody
}

// DecodeJSON deserializa o body em `dest` ou falha o teste.
func DecodeJSON(t *testing.T, body []byte, dest interface{}) {
	t.Helper()
	if err := json.Unmarshal(body, dest); err != nil {
		t.Fatalf("unmarshal %q: %v", string(body), err)
	}
}
