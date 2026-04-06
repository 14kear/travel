package batchexec

import (
	"testing"

	utls "github.com/refraction-networking/utls"
)

// TestChrome146SpecBuilds verifies the spec can be constructed without panicking
// and that the resulting struct has the expected shape.
func TestChrome146SpecBuilds(t *testing.T) {
	spec := Chrome146Spec()

	if len(spec.CipherSuites) == 0 {
		t.Fatal("Chrome146Spec: empty cipher suites")
	}
	if len(spec.Extensions) == 0 {
		t.Fatal("Chrome146Spec: empty extensions")
	}
	if len(spec.CompressionMethods) == 0 {
		t.Fatal("Chrome146Spec: empty compression methods")
	}
}

// TestChrome146SpecHasPQKeyShare verifies X25519MLKEM768 is present in key shares.
func TestChrome146SpecHasPQKeyShare(t *testing.T) {
	spec := Chrome146Spec()

	found := false
	for _, ext := range spec.Extensions {
		ks, ok := ext.(*utls.KeyShareExtension)
		if !ok {
			continue
		}
		for _, share := range ks.KeyShares {
			if share.Group == utls.X25519MLKEM768 {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("Chrome146Spec: X25519MLKEM768 key share not found — required for PQ fingerprint match")
	}
}

// TestChrome146SpecHasPQSupportedGroup verifies X25519MLKEM768 is in supported_groups.
func TestChrome146SpecHasPQSupportedGroup(t *testing.T) {
	spec := Chrome146Spec()

	found := false
	for _, ext := range spec.Extensions {
		sc, ok := ext.(*utls.SupportedCurvesExtension)
		if !ok {
			continue
		}
		for _, g := range sc.Curves {
			if g == utls.X25519MLKEM768 {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("Chrome146Spec: X25519MLKEM768 not in supported_groups — required for PQ fingerprint match")
	}
}

// TestChrome146SpecHasALPS verifies the ALPS extension (new codepoint 17613) is present.
func TestChrome146SpecHasALPS(t *testing.T) {
	spec := Chrome146Spec()

	found := false
	for _, ext := range spec.Extensions {
		if _, ok := ext.(*utls.ApplicationSettingsExtensionNew); ok {
			found = true
			break
		}
	}
	if !found {
		t.Error("Chrome146Spec: ApplicationSettingsExtensionNew (ALPS codepoint 17613) not found")
	}
}

// TestChrome146SpecHasECH verifies a GREASE ECH extension is present.
func TestChrome146SpecHasECH(t *testing.T) {
	spec := Chrome146Spec()

	found := false
	for _, ext := range spec.Extensions {
		if _, ok := ext.(*utls.GREASEEncryptedClientHelloExtension); ok {
			found = true
			break
		}
	}
	if !found {
		t.Error("Chrome146Spec: GREASEEncryptedClientHelloExtension (ECH) not found")
	}
}

// TestChrome146SpecALPNIncludesH2 verifies ALPN advertises h2.
func TestChrome146SpecALPNIncludesH2(t *testing.T) {
	spec := Chrome146Spec()

	for _, ext := range spec.Extensions {
		alpn, ok := ext.(*utls.ALPNExtension)
		if !ok {
			continue
		}
		for _, proto := range alpn.AlpnProtocols {
			if proto == "h2" {
				return
			}
		}
		t.Errorf("Chrome146Spec: ALPN extension found but h2 not in protocols: %v", alpn.AlpnProtocols)
		return
	}
	t.Error("Chrome146Spec: ALPN extension not found")
}
