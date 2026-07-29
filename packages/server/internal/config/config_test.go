package config

import "testing"

func TestLoadReasoningBudget(t *testing.T) {
	cases := []struct {
		name string
		set  bool
		val  string
		want int
	}{
		{name: "unset defaults to 0", set: false, want: 0},
		{name: "parsed", set: true, val: "2048", want: 2048},
		{name: "invalid falls back to 0", set: true, val: "notanint", want: 0},
		{name: "zero disables", set: true, val: "0", want: 0},
		{name: "at ceiling passes through", set: true, val: "32768", want: maxReasoningBudget},
		{name: "above ceiling clamps", set: true, val: "1000000", want: maxReasoningBudget},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				t.Setenv("AIR_REASONING_BUDGET", tc.val)
			} else {
				t.Setenv("AIR_REASONING_BUDGET", "")
			}
			if got := Load().ReasoningBudget; got != tc.want {
				t.Fatalf("ReasoningBudget = %d, want %d", got, tc.want)
			}
		})
	}
}
