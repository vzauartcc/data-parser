package config

import "testing"

func TestIsPointInAirspace(t *testing.T) {
	tests := []struct {
		name     string
		lat      float64
		lon      float64
		expected bool
	}{
		{
			name:     "Point in the middle of airspace",
			lat:      -88.0,
			lon:      42.0,
			expected: true,
		},
		{
			name:     "Point far outside (Atlantic Ocean)",
			lat:      -40.0,
			lon:      20.0,
			expected: false,
		},
		{
			name:     "Point just outside northern boundary",
			lat:      -89.0,
			lon:      45.0,
			expected: false,
		},
		{
			name:     "Point near the southern edge (Inside)",
			lat:      -88.0,
			lon:      40.1,
			expected: true,
		},
		{
			name:     "Point near the southern edge (Outside)",
			lat:      -88.0,
			lon:      39.9,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPointInAirspace(tt.lat, tt.lon)
			if result != tt.expected {
				t.Errorf("IsPointInAirspace(%f, %f) = %v; want %v",
					tt.lat, tt.lon, result, tt.expected)
			}
		})
	}
}
