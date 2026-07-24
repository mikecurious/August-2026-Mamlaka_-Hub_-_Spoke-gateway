package merchants

import "testing"

func TestMpesaMerchantReference(t *testing.T) {
	tests := []struct {
		name       string
		merchantID string
		secureID   string
		want       string
	}{
		{
			name:       "app transaction",
			merchantID: "app",
			secureID:   "3HjaAY77X_4oAW=",
			want:       "fusionfi-3HjaAY77X_4oAW=",
		},
		{
			name:       "fusionfi transaction",
			merchantID: "fusionfi",
			secureID:   "3HjaAY77X_4oAW=",
			want:       "fusionfi-3HjaAY77X_4oAW=",
		},
		{
			name:       "surrounding whitespace",
			merchantID: " app ",
			secureID:   " 3HjaAY77X_4oAW= ",
			want:       "fusionfi-3HjaAY77X_4oAW=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := mpesaMerchantReference(tt.merchantID, tt.secureID); got != tt.want {
				t.Fatalf("mpesaMerchantReference(%q, %q) = %q, want %q", tt.merchantID, tt.secureID, got, tt.want)
			}
		})
	}
}

func TestBuildMpesaAccountReferenceUsesFusionfiForApp(t *testing.T) {
	got := buildMpesaAccountReference("app", "ORDER123")
	if want := "fusionfi-ORDER123"; got != want {
		t.Fatalf("buildMpesaAccountReference(%q, %q) = %q, want %q", "app", "ORDER123", got, want)
	}
}
