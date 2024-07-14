package spotsync

import (
	"testing"

	"github.com/tschroed/spotsync"
)

func TestCanonicalizeName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{
			input: "10_000 Maniacs",
			want:  "10000maniacs",
		},
		{
			input: "Powerpuff Girls - Heroes & Vil",
			want:  "powerpuffgirlsheroesvil",
		},
		{
			input: "Live! - (CD1_ Boston 1979)",
			want:  "livecd1boston1979",
		},
		{
			input: "Lord For £39",
			want:  "lordfor39",
		},
		{
			input: "Carta de conduçao (Butterkeks",
			want:  "cartadeconduçaobutterkeks",
		},
		{
			input: "䩄䬠湥慴潲",
			want:  "䩄䬠湥慴潲",
		},
	}

	for _, tc := range cases {
		if c := spotsync.CanonicalizeName(tc.input); c != tc.want {
			t.Errorf("CanonicalizeName(%v): got %s, want %s", tc.input, c, tc.want)
		}
	}
}
