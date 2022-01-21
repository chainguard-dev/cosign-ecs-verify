package main

import (
	"context"
	"crypto"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecr"
	"github.com/aws/aws-sdk-go/service/kms"
	ecrlogin "github.com/awslabs/amazon-ecr-credential-helper/ecr-login"
	"github.com/awslabs/amazon-ecr-credential-helper/ecr-login/api"
	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/sigstore/cosign/pkg/cosign"
	ociremote "github.com/sigstore/cosign/pkg/oci/remote"
	sigs "github.com/sigstore/cosign/pkg/signature"
	"github.com/sigstore/sigstore/pkg/signature"

	"log"
	"os"
)

func Verify(containerImage, region, accountID string) (bool, error) {

	log.Printf("Veriying Container Image: %v", containerImage)

	//Generate the public key from KMS Alias
	kmsKeyAlias := os.Getenv("COSIGN_KEY")
	if len(kmsKeyAlias) == 0 {
		return false, errors.New("KMS Alias is empty")
	}
	log.Printf("Key Alias: %v", kmsKeyAlias)

	keyID := fmt.Sprintf("arn:aws:kms:%v:%v:alias/%v", region, accountID, kmsKeyAlias)
	GetPublicKeyInput := kms.GetPublicKeyInput{
		KeyId: &keyID,
	}

	mySession := session.Must(session.NewSession())
	svc := kms.New(mySession)
	result, err := svc.GetPublicKey(&GetPublicKeyInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case kms.ErrCodeNotFoundException:
				log.Println(kms.ErrCodeNotFoundException, aerr.Error())
			case kms.ErrCodeDisabledException:
				log.Println(kms.ErrCodeDisabledException, aerr.Error())
			case kms.ErrCodeKeyUnavailableException:
				log.Println(kms.ErrCodeKeyUnavailableException, aerr.Error())
			case kms.ErrCodeDependencyTimeoutException:
				log.Println(kms.ErrCodeDependencyTimeoutException, aerr.Error())
			case kms.ErrCodeUnsupportedOperationException:
				log.Println(kms.ErrCodeUnsupportedOperationException, aerr.Error())
			case kms.ErrCodeInvalidArnException:
				log.Println(kms.ErrCodeInvalidArnException, aerr.Error())
			case kms.ErrCodeInvalidGrantTokenException:
				log.Println(kms.ErrCodeInvalidGrantTokenException, aerr.Error())
			case kms.ErrCodeInvalidKeyUsageException:
				log.Println(kms.ErrCodeInvalidKeyUsageException, aerr.Error())
			case kms.ErrCodeInternalException:
				log.Println(kms.ErrCodeInternalException, aerr.Error())
			case kms.ErrCodeInvalidStateException:
				log.Println(kms.ErrCodeInvalidStateException, aerr.Error())
			default:
				log.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Println(err.Error())
		}
		return false, err
	}

	log.Printf("KMS Key Info: %v", result)
	ctx := context.TODO()

	var pubKey signature.Verifier
	pubKey, err = sigs.PublicKeyFromKeyRefWithHashAlgo(ctx, fmt.Sprintf("awskms:///alias//%v", kmsKeyAlias), crypto.SHA256)
	if err != nil {
		return false, err
	}
	var Keychain = authn.NewKeychainFromHelper(ecrlogin.ECRHelper{ClientFactory: api.DefaultClientFactory{}})

	//var remoteOp = []ociremote.Option{
	//	ociremote.WithRemoteOptions(remote.WithAuthFromKeychain(authn.NewMultiKeychain(authn.DefaultKeychain, Keychain)), remote.WithContext(ctx)),
	//}

	opts := []remote.Option{
		remote.WithAuthFromKeychain(Keychain),
		remote.WithContext(ctx),
	}
	co := &cosign.CheckOpts{
		RegistryClientOpts: []ociremote.Option{ociremote.WithRemoteOptions(opts...)},
		SigVerifier:        pubKey,
	}

	ref, err := name.ParseReference(containerImage)
	if err != nil {
		return false, err
	}
	repoName := os.Getenv("REPO_NAME")
	doesImageExist(repoName)

	//Verify Image
	log.Println("Verifying sig")
	_, verified, err := cosign.VerifyImageSignatures(ctx, ref, co)

	return verified, err
}

func doesImageExist(imageName string) {
	svc := ecr.New(session.New())
	input := &ecr.ListImagesInput{
		RepositoryName: aws.String(imageName),
	}

	result, err := svc.ListImages(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeServerException:
				fmt.Println(ecr.ErrCodeServerException, aerr.Error())
			case ecr.ErrCodeInvalidParameterException:
				fmt.Println(ecr.ErrCodeInvalidParameterException, aerr.Error())
			case ecr.ErrCodeRepositoryNotFoundException:
				fmt.Println(ecr.ErrCodeRepositoryNotFoundException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}

	fmt.Println(result)
}
