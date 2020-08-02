// Package synapseconf holds utilities for configuring the Synapse Matrix home
// server.
package synapseconf

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"text/template"
)

// GenerateSigningKey generates an ed25519 private key suitably encoded for use
// as synapse signing key.
func GenerateSigningKey(keyID string) ([]byte, error) {
	_, sk, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	b64Key := base64.StdEncoding.WithPadding(base64.NoPadding).EncodeToString(sk.Seed())

	var b bytes.Buffer
	fmt.Fprintf(&b, "ed25519 %s %s\n", keyID, b64Key)

	return b.Bytes(), nil
}

// A HomeserverConfig describes the basic configuration for a Synapse
// homeserver.
type HomeserverConfig struct {
	// public DNS name
	ServerName string
	// redirect web clients to this URL (probably a riot-web instance)
	WebClientLocation string
	// URI to reach an admin with (example: mailto:admin@example.com)
	AdminContact string
	// whether to report anonymous usage statistics
	ReportStats bool

	// Various secrets Synapse uses. If left at their zero values a
	// securely generated random string is used instead.
	RegistrationSharedSecret string
	MacaroonSecretKey        string
	FormSecret               string

	// If set, configure for Postgres DB. Otherwise, use sqlite3.
	PostgresConfig *PostgresConfig

	// included verbatim at the tail of homeserver.yaml
	IncludeConfigYAML []byte
}

// A PostgresConfig has the parameters for connecting to a Postgres database.
type PostgresConfig struct {
	User     string
	Password string
	Database string
	Host     string
	Port     string
}

//go:generate go run bake.go -o homeserver.yaml.go homeserverYAMLTemplateText:homeserver.yaml.in

var homeserverYAMLTemplate = template.Must(
	template.New("homeserver.yaml").Parse(homeserverYAMLTemplateText),
)

// GenerateHomeserverYAML outputs a homeserver.yaml using the provided HomeserverConfig.
func GenerateHomeserverYAML(config *HomeserverConfig) ([]byte, error) {
	// Make a copy of the passed in config to allow for adjustments
	c := new(HomeserverConfig)
	*c = *config

	if c.RegistrationSharedSecret == "" {
		c.RegistrationSharedSecret = randomString(64)
	}
	if c.MacaroonSecretKey == "" {
		c.MacaroonSecretKey = randomString(64)
	}
	if c.FormSecret == "" {
		c.FormSecret = randomString(64)
	}

	var b bytes.Buffer
	if err := homeserverYAMLTemplate.Execute(&b, c); err != nil {
		return nil, err
	}

	return b.Bytes(), nil
}

// RandomString generates a printable random string of length n using a
// cryptographically-secure RNG.
func randomString(n int) string {
	scratch := make([]byte, (n+3)/4*3)
	if _, err := rand.Read(scratch); err != nil {
		panic(err)
	}

	return base64.URLEncoding.EncodeToString(scratch)[:n]
}
