package main

import (
	"context"
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
	log.Printf("[INFO] Key Alias: %v", kmsKeyAlias)

	keyID := fmt.Sprintf("arn:aws:kms:%v:%v:alias/%v", region, accountID, kmsKeyAlias)
	log.Printf("[INFO] Key ID: %v", keyID)
	GetPublicKeyInput := kms.GetPublicKeyInput{
		KeyId: aws.String(keyID),
	}

	mySession := session.Must(session.NewSession())
	svc := kms.New(mySession)
	kmsResult, err := svc.GetPublicKey(&GetPublicKeyInput)
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
			log.Printf("[EEROR] Accessing Key %v", err.Error())
		}
		return false, err
	}

	log.Printf("[INFO] KMS Key Info: %v", kmsResult)
	ctx := context.TODO()

	//	var key = []byte(`
	//-----BEGIN PUBLIC KEY-----
	//MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAk5ZYsYTo8KjzDO/e62Ll
	//b2bn5WEwpXtMwyKgHh3qCSvSut8GPMIpfHhpJ1XiJ3eJdTqjKIDYtRC+73nlu3WL
	//23jO0l3onGzUTjVumF1oGRmFWj87ZQp4P+SwjwsrF2fxZ8HjJ9JX88yOVNrJrxMf
	//zGqK/NBhPXFSVhIMUO0/Q8apvkDU2hdUpYd7mkXTiZQTgYEWkuGYTD7PPdUxUspg
	//W1BbpGHiiQ4El8xdTJrMGvBb8Saa0eDdf848mjUQv1o3aCxj+ha4iFjiA0nDsb7P
	//h5Fpu+95f47KZHkXHAH+DLI+g5YeJA/+Wmqa8872iyLBt4TOpATpLwqnIktTaymh
	//4ysoQDCBdyYUxqXqV2uWTGeV8QoDBF+RgWeNGAFZDZi6eA7BwkuQj1zCPIhIzREN
	//lHlmo5pNP5bLpfD8r/5EY9F9LkzMO+hY1vv0Y8wLUaB62BWI5dFRAHgAYrGrGvdG
	//00yMojvN8Mbz3LqaRq610rB4bh6MYYr27JhQJt1EFexhvD1UazoMp02w34W98XXP
	//FpHkmDuIykcJfKk/cJMPj2+F+7i3ZJbrMu3c+dAe/ce9nzPq+A+2xgjVZJ6gpoMD
	//gimNXDD4GO9VbC11+EOOd9WdVMI95e3Z6rPzk7HuhIjuZi/lOCmzJR0vSpjrlaTw
	//4m9Tymqqe+YbxXhsSnFYptcCAwEAAQ==
	//-----END PUBLIC KEY-----
	//`)
	//
	//	err = os.WriteFile("/tmp/publickey", key, 0644)
	//	if err != nil {
	//		log.Println("error writing public key")
	//	}

	pubKey, err := sigs.LoadPublicKey(ctx, fmt.Sprintf("awskms:///%s", *kmsResult.KeyId))
	if err != nil {
		return false, err
	}

	ref, err := name.ParseReference(containerImage)
	if err != nil {
		return false, err
	}
	repoName := os.Getenv("REPO_NAME")
	imageIDs, _ := doesImageExist(repoName)
	imageDigestByAWS, _ := findImageDigestByTag(imageIDs, "0.0.1")

	ecrHelper := ecrlogin.ECRHelper{ClientFactory: api.DefaultClientFactory{}}

	img, err := remote.Get(ref, remote.WithAuthFromKeychain(authn.NewKeychainFromHelper(ecrHelper)))
	if err != nil {
		log.Printf("[ERROR] REMOTE GET Error Getting Ref %v %v", ref, err)
	}

	log.Printf("[INFO] REMOTE GET Image Manifest %v", string(img.Manifest))

	image, _ := img.Image()
	digest, _ := image.Digest()

	log.Printf("[INFO] REMOTE GET Remote Get Image Digest %v", digest)
	log.Printf("[INFO] AWS SDK Image Digest: %v", imageDigestByAWS)

	if digest.String() == imageDigestByAWS {
		log.Printf("[INFO] Remote has and AWS has are same!!!!!")
	} else {
		log.Printf("[ERROR] Remote Get and AWS Digests are not the same")
	}

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
	}

	return len(verifiedSigs) > 0, err
}

func doesImageExist(imageName string) ([]*ecr.ImageIdentifier, error) {
	svc := ecr.New(session.New())
	input := &ecr.ListImagesInput{
		RepositoryName: aws.String(imageName),
	}

	result, err := svc.ListImages(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecr.ErrCodeServerException:
				log.Println(ecr.ErrCodeServerException, aerr.Error())
			case ecr.ErrCodeInvalidParameterException:
				log.Println(ecr.ErrCodeInvalidParameterException, aerr.Error())
			case ecr.ErrCodeRepositoryNotFoundException:
				log.Println(ecr.ErrCodeRepositoryNotFoundException, aerr.Error())
			default:
				log.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Println(err.Error())
		}
		return nil, err
	}
	return result.ImageIds, nil
}

func findImageDigestByTag(image []*ecr.ImageIdentifier, tag string) (string, error) {
	for i := 0; i < len(image); i++ {
		if *image[i].ImageTag == tag {
			return *image[i].ImageDigest, nil
		}
	}
	return "", errors.New("image digest not found with provided tag")
}
