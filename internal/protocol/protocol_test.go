package protocol

import "testing"

func TestRPCError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *RPCError
		want string
	}{
		{
			name: "nil receiver",
			err:  nil,
			want: "<nil>",
		},
		{
			name: "code and message",
			err:  &RPCError{Code: CodeInvalidParams, Message: "direction is required"},
			want: "rpc error -32602: direction is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestRPCError_SatisfiesError verifies *RPCError is assignable to error.
func TestRPCError_SatisfiesError(t *testing.T) {
	var err error = &RPCError{Code: CodeInternalError, Message: "boom"}
	if err.Error() == "" {
		t.Fatal("expected non-empty error string")
	}
}
