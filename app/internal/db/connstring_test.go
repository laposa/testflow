package db

import "testing"

func TestConnString(t *testing.T) {
	tests := []struct {
		c    Config
		want string
	}{
		{Config{URL: "postgres://x/y"}, "postgres://x/y"},
		{Config{Host: "localhost", Name: "testflow"}, "postgres://localhost:5432/testflow"},
		{Config{Host: "db", Port: 5433, Name: "tf", User: "u", Password: "p@ss"}, "postgres://db:5433/tf"},
		{Config{Host: "::1", Port: 5432, Name: "tf"}, "postgres://[::1]:5432/tf"},
	}
	for _, tt := range tests {
		if got := tt.c.connString(); got != tt.want {
			t.Errorf("connString(%+v) = %q, want %q", tt.c, got, tt.want)
		}
	}
}
