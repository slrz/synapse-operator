package synapseconf

import (
	"crypto/ed25519"
	"encoding/base64"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

// TestGenerateHomeserverYAMLNoEmptySecrets ensures that we properly handle
// zero-valued secrets (by using a generated random string instead) and not
// just dump the empty string into the output YAML.
func TestGenerateHomeserverYAMLNoEmptySecrets(t *testing.T) {
	c := &HomeserverConfig{
		ServerName: "example.com",

		// Secrets at their zero values.
		RegistrationSharedSecret: "",
		MacaroonSecretKey:        "",
		FormSecret:               "",
	}

	p, err := GenerateHomeserverYAML(c)
	if err != nil {
		t.Fatalf("GenerateHomeserverYAML: %v", err)
	}

	var secrets struct {
		RegistrationSharedSecret string `yaml:"registration_shared_secret"`
		MacaroonSecretKey        string `yaml:"macaroon_secret_key"`
		FormSecret               string `yaml:"form_secret"`
	}

	if err := yaml.Unmarshal(p, &secrets); err != nil {
		t.Fatalf("yaml.Unmarshal: %v", err)
	}

	if s := secrets.RegistrationSharedSecret; len(s) != 64 {
		t.Errorf("registration_shared_secret: expect random string of length 64, got %q", s)
	}
	if s := secrets.MacaroonSecretKey; len(s) != 64 {
		t.Errorf("macaroon_secret_key: expect random string of length 64, got %q", s)
	}
	if s := secrets.FormSecret; len(s) != 64 {
		t.Errorf("form_secret: expect random string of length 64, got %q", s)
	}
}

func TestGenerateSigningKey(t *testing.T) {
	p, err := GenerateSigningKey("a_xyzw")
	if err != nil {
		t.Fatalf("GenerateSigningKey: %v", err)
	}

	parts := strings.SplitN(string(p), " ", 3)
	if len(parts) != 3 ||
		parts[0] != "ed25519" ||
		parts[1] != "a_xyzw" ||
		len(parts[2]) == 0 || parts[2][len(parts[2])-1] != '\n' {

		t.Fatalf(`expect "ed25519 a_xyzw <base64-encoded-sk>\n", got %q`, p)
	}

	b64Key := parts[2]
	b64Key = b64Key[:len(b64Key)-1] // strip trailing linefeed
	sk, err := base64.StdEncoding.WithPadding(base64.NoPadding).DecodeString(b64Key)
	if err != nil {
		t.Fatalf("DecodeString: %v", err)
	}
	if len(sk) != ed25519.SeedSize {
		t.Errorf("expect key of size %d, got %d", ed25519.SeedSize, len(sk))
	}
}
