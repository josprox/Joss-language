package pluginpkg

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
)

func TestSignedArchiveVerificationAndTamperDetection(t *testing.T) {
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	metadata := Metadata{Name: "vendor/demo", Version: "1.0.0", Bytecode: "bytecode/main.jbc"}
	files := map[string][]byte{"bytecode/main.jbc": []byte("compiled")}
	data, err := BuildSigned(metadata, files, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := ReadVerified(data)
	if err != nil {
		t.Fatal(err)
	}
	if archive.Metadata.KeyID == "" || archive.Metadata.Signature == "" {
		t.Fatal("signed archive has incomplete signature metadata")
	}

	archive.Files[archive.Metadata.Bytecode][0] ^= 0xff
	if err := verifySignature(archive.Metadata, archive.Files); err == nil {
		t.Fatal("tampered archive was accepted")
	}
}

func TestVerifiedReaderRejectsUnsignedArchive(t *testing.T) {
	data, err := Build(Metadata{Name: "vendor/demo", Version: "1.0.0", Bytecode: "main.jbc"}, map[string][]byte{"main.jbc": []byte("x")})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ReadVerified(data); err == nil {
		t.Fatal("unsigned archive was accepted")
	}
}
