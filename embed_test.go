package situkang

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func testDist() fstest.MapFS {
	return fstest.MapFS{
		"index.html":    &fstest.MapFile{Data: []byte("<!doctype html><title>SiTukang</title>")},
		"assets/app.js": &fstest.MapFile{Data: []byte("console.log('ok')")},
		"favicon.svg":   &fstest.MapFile{Data: []byte("<svg></svg>")},
	}
}

func TestSPAHandlerServesAssetsAndFallsBack(t *testing.T) {
	h := NewSPAHandler(testDist())

	cases := []struct {
		path        string
		wantStatus  int
		wantBodyHas string
	}{
		{"/", http.StatusOK, "SiTukang"},
		{"/pekerja/12", http.StatusOK, "SiTukang"},
		{"/assets/app.js", http.StatusOK, "console.log"},
		{"/favicon.svg", http.StatusOK, "<svg>"},
		{"/assets/missing.js", http.StatusNotFound, ""},
	}

	for _, tc := range cases {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, tc.path, nil)
		h.ServeHTTP(rec, req)

		if rec.Code != tc.wantStatus {
			t.Errorf("%s: status = %d, want %d", tc.path, rec.Code, tc.wantStatus)
			continue
		}
		if body := rec.Body.String(); !strings.Contains(body, tc.wantBodyHas) {
			t.Errorf("%s: body %q does not contain %q", tc.path, body, tc.wantBodyHas)
		}
	}
}

func TestSPAHandlerRejectsMutations(t *testing.T) {
	h := NewSPAHandler(testDist())

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/", nil))

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if allow := rec.Header().Get("Allow"); allow != "GET, HEAD" {
		t.Fatalf("Allow = %q, want %q", allow, "GET, HEAD")
	}
}
