package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSubmitJob_BodySizeLimit(t *testing.T) {
	const limit = 1 << 20 // API contract: 1 MiB total request body.
	const prefix = `{"language":"python","code":"print('ok')","stdin":"`
	const suffix = `"}`

	bodyOfSize := func(size int) string {
		return prefix + strings.Repeat("x", size-len(prefix)-len(suffix)) + suffix
	}

	smallBody := `{"language":"python","code":"print('ok')"}`

	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "one_byte_below_limit",
			body:       bodyOfSize(limit - 1),
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "exactly_at_limit",
			body:       bodyOfSize(limit),
			wantStatus: http.StatusAccepted,
		},
		{
			name:       "one_byte_above_limit",
			body:       bodyOfSize(limit + 1),
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name: "oversized_code",
			body: `{"language":"python","code":"` +
				strings.Repeat("x", limit) + `"}`,
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name: "oversized_trailing_whitespace",
			body: smallBody +
				strings.Repeat(" ", limit-len(smallBody)+1),
			wantStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:       "extra_json_value",
			body:       smallBody + `{}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range tests {
		for _, unknownLength := range []bool{false, true} {
			name := tc.name + "/known_length"
			if unknownLength {
				name = tc.name + "/unknown_length"
			}

			t.Run(name, func(t *testing.T) {
				handler, mux := setupAPI()

				req := httptest.NewRequest(
					http.MethodPost,
					"/api/jobs",
					strings.NewReader(tc.body),
				)
				req.Header.Set("Content-Type", "application/json")
				if unknownLength {
					req.ContentLength = -1
				}

				recorder := httptest.NewRecorder()
				mux.ServeHTTP(recorder, req)

				if recorder.Code != tc.wantStatus {
					t.Fatalf(
						"expected HTTP %d, got %d: %s",
						tc.wantStatus,
						recorder.Code,
						recorder.Body.String(),
					)
				}

				wantJobs := 0
				if tc.wantStatus == http.StatusAccepted {
					wantJobs = 1
				}

				if got := handler.store.Count(); got != wantJobs {
					t.Errorf("stored jobs: expected %d, got %d", wantJobs, got)
				}
				if got := handler.queue.Len(); got != wantJobs {
					t.Errorf("queued jobs: expected %d, got %d", wantJobs, got)
				}
			})
		}
	}
}
