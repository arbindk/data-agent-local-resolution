package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: policy-tool <gen-keys|sign|verify> [args]")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "gen-keys":
		genKeys(os.Args[2:])
	case "sign":
		sign(os.Args[2:])
	case "verify":
		verify(os.Args[2:])
	default:
		fmt.Println("unknown command:", os.Args[1])
		os.Exit(1)
	}
}

func genKeys(args []string) {
	fs := flag.NewFlagSet("gen-keys", flag.ExitOnError)
	outDir := fs.String("out", "demo/keys", "output directory")
	_ = fs.Parse(args)

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}

	_ = os.MkdirAll(*outDir, 0755)

	if err := os.WriteFile(*outDir+"/ed25519-public.b64", []byte(base64.StdEncoding.EncodeToString(pub)), 0644); err != nil {
		panic(err)
	}
	if err := os.WriteFile(*outDir+"/ed25519-private.b64", []byte(base64.StdEncoding.EncodeToString(priv)), 0600); err != nil {
		panic(err)
	}

	fmt.Println("generated keys in", *outDir)
}

func sign(args []string) {
	fs := flag.NewFlagSet("sign", flag.ExitOnError)
	privateKeyPath := fs.String("private", "demo/keys/ed25519-private.b64", "private key path")
	artifactPath := fs.String("artifact", "", "artifact path")
	outPath := fs.String("out", "", "signature output path")
	_ = fs.Parse(args)

	if *artifactPath == "" || *outPath == "" {
		panic("artifact and out are required")
	}

	privBytesB64, err := os.ReadFile(*privateKeyPath)
	if err != nil {
		panic(err)
	}
	privBytes, err := base64.StdEncoding.DecodeString(string(privBytesB64))
	if err != nil {
		panic(err)
	}

	artifact, err := os.ReadFile(*artifactPath)
	if err != nil {
		panic(err)
	}

	sig := ed25519.Sign(ed25519.PrivateKey(privBytes), artifact)
	if err := os.WriteFile(*outPath, []byte(base64.StdEncoding.EncodeToString(sig)), 0644); err != nil {
		panic(err)
	}

	fmt.Println("signed artifact:", *artifactPath)
}

func verify(args []string) {
	fs := flag.NewFlagSet("verify", flag.ExitOnError)
	publicKeyPath := fs.String("public", "demo/keys/ed25519-public.b64", "public key path")
	artifactPath := fs.String("artifact", "", "artifact path")
	sigPath := fs.String("sig", "", "signature path")
	_ = fs.Parse(args)

	if *artifactPath == "" || *sigPath == "" {
		panic("artifact and sig are required")
	}

	pubBytesB64, err := os.ReadFile(*publicKeyPath)
	if err != nil {
		panic(err)
	}
	pubBytes, err := base64.StdEncoding.DecodeString(string(pubBytesB64))
	if err != nil {
		panic(err)
	}

	artifact, err := os.ReadFile(*artifactPath)
	if err != nil {
		panic(err)
	}

	sigB64, err := os.ReadFile(*sigPath)
	if err != nil {
		panic(err)
	}
	sig, err := base64.StdEncoding.DecodeString(string(sigB64))
	if err != nil {
		panic(err)
	}

	if !ed25519.Verify(ed25519.PublicKey(pubBytes), artifact, sig) {
		panic("signature verification failed")
	}

	fmt.Println("signature verified")
}
