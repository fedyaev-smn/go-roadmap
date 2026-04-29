package main

import "testing"

func TestListenAddr(t *testing.T) {
	// Do not use t.Parallel() here: t.Setenv changes process-wide environment.

	tests := []struct {
		name string
		addr string // value for ADDR (empty = unset/cleared for this case)
		port string // value for PORT
		want string
	}{
		{
			name: "default when addr and port empty",
			addr: "",
			port: "",
			want: ":8080",
		},
		{
			name: "port only",
			addr: "",
			port: "3000",
			want: ":3000",
		},
		{
			name: "addr takes precedence over port",
			addr: ":9000",
			port: "3000",
			want: ":9000",
		},
		{
			name: "port trimmed",
			addr: "",
			port: " 3000 ",
			want: ":3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ADDR", tt.addr)
			t.Setenv("PORT", tt.port)
			if got := listenAddr(); got != tt.want {
				t.Errorf("listenAddr() = %q, want %q", got, tt.want)
			}
		})
	}
}
