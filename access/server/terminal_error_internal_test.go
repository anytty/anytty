package server

import (
	"fmt"
	"testing"

	terminalprovider "github.com/anytty/anytty/access/provider/terminal"
	"github.com/anytty/anytty/internal/providerproto"
	"github.com/anytty/anytty/proto/access/apipb"
)

// TestProviderErrorResultMapsTypedAndWireCodes 锁定访问 wire 的错误码语义：
// typed sentinel 与 CodedError wire code 映射到既有 ApiErrorCode（access/server
// 不依赖 pool/provider 实现包）。
func TestProviderErrorResultMapsTypedAndWireCodes(t *testing.T) {
	cases := []struct {
		name      string
		err       error
		want      apipb.ApiErrorCode
		retryable bool
	}{
		{
			name: "typed not found keeps wire code",
			err: fmt.Errorf("%w: %w", terminalprovider.ErrNotFound,
				&terminalprovider.CodedError{Code: providerproto.ErrorNotFound, Message: "terminal not found"}),
			want: apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND,
		},
		{
			name: "typed conflict keeps wire code",
			err: fmt.Errorf("%w: %w", terminalprovider.ErrConflict,
				&terminalprovider.CodedError{Code: providerproto.ErrorConflict, Message: "terminal exists"}),
			want: apipb.ApiErrorCode_API_ERROR_CODE_CONFLICT,
		},
		{
			name: "typed unavailable is retryable",
			err: fmt.Errorf("%w: %w", terminalprovider.ErrUnavailable,
				&terminalprovider.CodedError{Code: providerproto.ErrorUnavailable, Message: "provider closed"}),
			want:      apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE,
			retryable: true,
		},
		{
			name:      "unsupported capability",
			err:       terminalprovider.ErrUnsupported,
			want:      apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE,
			retryable: true,
		},
		{
			name: "bad request",
			err:  &terminalprovider.CodedError{Code: providerproto.ErrorBadRequest, Message: "bad"},
			want: apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST,
		},
		{
			name: "forbidden",
			err:  &terminalprovider.CodedError{Code: providerproto.ErrorForbidden, Message: "forbidden"},
			want: apipb.ApiErrorCode_API_ERROR_CODE_FORBIDDEN,
		},
		{
			name: "stale resource",
			err:  &terminalprovider.CodedError{Code: providerproto.ErrorStaleResource, Message: "stale"},
			want: apipb.ApiErrorCode_API_ERROR_CODE_STALE_RESOURCE,
		},
		{
			name:      "exhausted is retryable",
			err:       &terminalprovider.CodedError{Code: providerproto.ErrorExhausted, Message: "exhausted"},
			want:      apipb.ApiErrorCode_API_ERROR_CODE_RESOURCE_EXHAUSTED,
			retryable: true,
		},
		{
			name: "unknown wire code is internal",
			err:  &terminalprovider.CodedError{Code: 999, Message: "boom"},
			want: apipb.ApiErrorCode_API_ERROR_CODE_INTERNAL,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			result := providerErrorResult(&apipb.CommandEnvelope{}, testCase.err)
			if got := result.GetError().GetCode(); got != testCase.want {
				t.Fatalf("code = %s, want %s", got, testCase.want)
			}
			if got := result.GetError().GetRetryable(); got != testCase.retryable {
				t.Fatalf("retryable = %v, want %v", got, testCase.retryable)
			}
		})
	}
}
