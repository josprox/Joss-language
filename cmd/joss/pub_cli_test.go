package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jossecurity/joss/pkg/pluginpkg"
)

func TestVerifyPublishArtifactRequiresValidSignedJP(t *testing.T) {
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	data, err := pluginpkg.BuildSigned(
		pluginpkg.Metadata{Name: "vendor/demo", Version: "1.2.3", Bytecode: "main.jbc"},
		map[string][]byte{"main.jbc": []byte("compiled")},
		key,
	)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		_, _ = response.Write(data)
	}))
	defer server.Close()
	digest := sha256.Sum256(data)
	keyID, err := verifyPublishArtifact(server.URL, hex.EncodeToString(digest[:]), "vendor/demo", "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if keyID == "" {
		t.Fatal("verified artifact did not return key id")
	}
}
