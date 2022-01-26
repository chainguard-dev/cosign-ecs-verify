package main

import (
	"context"
	"fmt"
	ecrlogin "github.com/awslabs/amazon-ecr-credential-helper/ecr-login"
	"github.com/awslabs/amazon-ecr-credential-helper/ecr-login/api"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sigstore/cosign/pkg/cosign"
	ociremote "github.com/sigstore/cosign/pkg/oci/remote"
	sigs "github.com/sigstore/cosign/pkg/signature"
	"log"
)

func Verify(containerImage, keyID string) (bool, error) {

	log.Printf("[INFO] Veriying Container Image: %v", containerImage)

	ctx := context.TODO()

	pubKey, err := sigs.LoadPublicKey(ctx, fmt.Sprintf("awskms:///%s", keyID))
	if err != nil {
		return false, err
	}

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
