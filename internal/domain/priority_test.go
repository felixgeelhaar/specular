package domain

import (
	"testing"
)

func TestNewPriority(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    Priority
		wantErr bool
	}{
		{
			name:    "valid P0",
			value:   "P0",
			want:    PriorityP0,
			wantErr: false,
		},
		{
			name:    "valid P1",
			value:   "P1",
			want:    PriorityP1,
			wantErr: false,
		},
		{
			name:    "valid P2",
			value:   "P2",
			want:    PriorityP2,
			wantErr: false,
		},
		{
			name:    "invalid P3",
			value:   "P3",
			wantErr: true,
		},
		{
			name:    "invalid lowercase",
			value:   "p0",
			wantErr: true,
		},
		{
			name:    "invalid empty",
			value:   "",
			wantErr: true,
		},
		{
			name:    "invalid random string",
			value:   "high",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewPriority(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPriority() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("NewPriority() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPriority_Validate(t *testing.T) {
	tests := []struct {
		name     string
		priority Priority
		wantErr  bool
	}{
		{"P0 is valid", PriorityP0, false},
		{"P1 is valid", PriorityP1, false},
		{"P2 is valid", PriorityP2, false},
		{"P3 is invalid", Priority("P3"), true},
		{"empty is invalid", Priority(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.priority.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Priority.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPriority_String(t *testing.T) {
	tests := []struct {
		name     string
		priority Priority
		want     string
	}{
		{"P0", PriorityP0, "P0"},
		{"P1", PriorityP1, "P1"},
		{"P2", PriorityP2, "P2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.priority.String(); got != tt.want {
				t.Errorf("Priority.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPriority_IsHigherThan(t *testing.T) {
	tests := []struct {
		name string
		p1   Priority
		p2   Priority
		want bool
	}{
		{"P0 is higher than P1", PriorityP0, PriorityP1, true},
		{"P0 is higher than P2", PriorityP0, PriorityP2, true},
		{"P1 is higher than P2", PriorityP1, PriorityP2, true},
		{"P1 is not higher than P0", PriorityP1, PriorityP0, false},
		{"P2 is not higher than P1", PriorityP2, PriorityP1, false},
		{"P2 is not higher than P0", PriorityP2, PriorityP0, false},
		{"P0 is not higher than P0", PriorityP0, PriorityP0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p1.IsHigherThan(tt.p2); got != tt.want {
				t.Errorf("Priority.IsHigherThan() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPriority_IsLowerThan(t *testing.T) {
	tests := []struct {
		name string
		p1   Priority
		p2   Priority
		want bool
	}{
		{"P2 is lower than P1", PriorityP2, PriorityP1, true},
		{"P2 is lower than P0", PriorityP2, PriorityP0, true},
		{"P1 is lower than P0", PriorityP1, PriorityP0, true},
		{"P0 is not lower than P1", PriorityP0, PriorityP1, false},
		{"P1 is not lower than P2", PriorityP1, PriorityP2, false},
		{"P0 is not lower than P2", PriorityP0, PriorityP2, false},
		{"P0 is not lower than P0", PriorityP0, PriorityP0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p1.IsLowerThan(tt.p2); got != tt.want {
				t.Errorf("Priority.IsLowerThan() = %v, want %v", got, tt.want)
			}
		})
	}
}
