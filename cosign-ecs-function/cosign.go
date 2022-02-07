package main

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	ecrlogin "github.com/awslabs/amazon-ecr-credential-helper/ecr-login"
	"github.com/awslabs/amazon-ecr-credential-helper/ecr-login/api"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sigstore/cosign/pkg/cosign"
	ociremote "github.com/sigstore/cosign/pkg/oci/remote"
	sigs "github.com/sigstore/cosign/pkg/signature"
	sigstoresigs "github.com/sigstore/sigstore/pkg/signature"
	"log"
	"os"
)

func getKey(ctx context.Context, accountID, region string) (sigstoresigs.Verifier, error) {
	keyARN := os.Getenv("COSIGN_KEY_ARN")
	keyPEM := os.Getenv("COSIGN_KEY_PEM")
	if len(keyARN) != 0 && len(keyPEM) != 0 {
		return nil, errors.New("Must provide exactly one of COSIGN_KEY_ARN or COSIGN_KEY_PEM.")
	} else if len(keyARN) != 0 {
		log.Printf("[INFO] Key Alias ARN: %v", keyARN)
		return sigs.LoadPublicKey(ctx, fmt.Sprintf("awskms:///%s", keyARN))
	} else if len(keyPEM) != 0 {
		log.Printf("[INFO] Key Alias PEM: %v", keyPEM)
		return sigs.LoadPublicKeyRaw([]byte(keyPEM), crypto.SHA256)
	} else {
		return nil, errors.New("Must provide either COSIGN_KEY_ARN or COSIGN_KEY_PEM.")
	}
}

func Verify(containerImage string, pubKey sigstoresigs.Verifier) (bool, error) {

	log.Printf("[INFO] Veriying Container Image: %v", containerImage)

	ctx := context.TODO()

	ref, err := name.ParseReference(containerImage)
	if err != nil {
		return false, err
	}

	ecrHelper := ecrlogin.ECRHelper{ClientFactory: api.DefaultClientFactory{}}

	opts := []remote.Option{
		remote.WithAuthFromKeychain(authn.NewKeychainFromHelper(ecrHelper)),
		remote.WithContext(ctx),
	}

	co := &cosign.CheckOpts{
		ClaimVerifier:      cosign.SimpleClaimVerifier,
		RegistryClientOpts: []ociremote.Option{ociremote.WithRemoteOptions(opts...)},
		SigVerifier:        pubKey,
	}

	//Verify Image
	log.Println("[INFO] COSIGN Verifying sig")
	verifiedSigs, _, err := cosign.VerifyImageSignatures(ctx, ref, co)
	if err != nil {
		log.Printf("[ERROR] COSIGN error: %v", err)
		return false, err
	}

	return len(verifiedSigs) > 0, err
}
