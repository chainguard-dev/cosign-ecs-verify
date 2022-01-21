package main

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"os"
)
import "log"

func Verify(containerImage, region, accountID string) error {

	log.Printf("Veriying Container Image: %v", containerImage)

	//Generate the public key from KMS Alias
	kmsKeyAlias := os.Getenv("COSIGN_KEY")
	if len(kmsKeyAlias) == 0 {
		return errors.New("KMS Alias is empty")
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
				fmt.Println(kms.ErrCodeNotFoundException, aerr.Error())
			case kms.ErrCodeDisabledException:
				fmt.Println(kms.ErrCodeDisabledException, aerr.Error())
			case kms.ErrCodeKeyUnavailableException:
				fmt.Println(kms.ErrCodeKeyUnavailableException, aerr.Error())
			case kms.ErrCodeDependencyTimeoutException:
				fmt.Println(kms.ErrCodeDependencyTimeoutException, aerr.Error())
			case kms.ErrCodeUnsupportedOperationException:
				fmt.Println(kms.ErrCodeUnsupportedOperationException, aerr.Error())
			case kms.ErrCodeInvalidArnException:
				fmt.Println(kms.ErrCodeInvalidArnException, aerr.Error())
			case kms.ErrCodeInvalidGrantTokenException:
				fmt.Println(kms.ErrCodeInvalidGrantTokenException, aerr.Error())
			case kms.ErrCodeInvalidKeyUsageException:
				fmt.Println(kms.ErrCodeInvalidKeyUsageException, aerr.Error())
			case kms.ErrCodeInternalException:
				fmt.Println(kms.ErrCodeInternalException, aerr.Error())
			case kms.ErrCodeInvalidStateException:
				fmt.Println(kms.ErrCodeInvalidStateException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return err
	}

	fmt.Println(result)

	//Verify Image

	return nil
}
