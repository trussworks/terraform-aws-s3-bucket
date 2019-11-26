package test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	awssdk "github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/require"
)

func GetPublicAccessBlockConfiguration(t *testing.T, region string, bucketName string) *s3.PublicAccessBlockConfiguration {
	config, err := GetPublicAccessBlockConfigurationE(t, region, bucketName)
	require.NoError(t, err)
	return config

}

func GetPublicAccessBlockConfigurationE(t *testing.T, region string, bucketName string) (*s3.PublicAccessBlockConfiguration, error) {
	s3Client, err := aws.NewS3ClientE(t, region)

	if err != nil {
		return nil, err
	}

	params := &s3.GetPublicAccessBlockInput{
		Bucket: awssdk.String(bucketName),
	}

	var publicAccessBlockConfiguration *s3.PublicAccessBlockConfiguration
	maxRetries := 3
	retryDuration, _ := time.ParseDuration("30s")
	_, err = retry.DoWithRetryE(t, "Get public access block configuration", maxRetries, retryDuration,
		func() (string, error) {
			publicAccessBlock, err := s3Client.GetPublicAccessBlock(params)
			if err != nil {
				return "", err
			}
			publicAccessBlockConfiguration = publicAccessBlock.PublicAccessBlockConfiguration
			return "Retrieved public access block configuration", nil
		},
	)

	if err != nil {
		return nil, err
	}

	return publicAccessBlockConfiguration, nil
}

func AssertS3BucketEncryptionEnabled(t *testing.T, region string, bucketName string) {
	err := AssertS3BucketEncryptionEnabledE(t, region, bucketName)
	require.NoError(t, err)
}

func AssertS3BucketEncryptionEnabledE(t *testing.T, region string, bucketName string) error {
	s3Client, err := aws.NewS3ClientE(t, region)

	if err != nil {
		return err
	}

	params := &s3.GetBucketEncryptionInput{
		Bucket: awssdk.String(bucketName),
	}

	maxRetries := 3
	retryDuration, _ := time.ParseDuration("30s")
	_, err = retry.DoWithRetryE(t, "Get bucket encryption", maxRetries, retryDuration,
		func() (string, error) {
			encryption, err := s3Client.GetBucketEncryption(params)

			if err != nil {
				return "", err
			}

			expectedEncryption := "AES256"
			for _, element := range encryption.ServerSideEncryptionConfiguration.Rules {
				actualEncryption := element.ApplyServerSideEncryptionByDefault.SSEAlgorithm
				if *actualEncryption != expectedEncryption {
					return "", fmt.Errorf("server side encyption test failed. got: %v, expected: %v", actualEncryption, expectedEncryption)
				}
			}

			return "Retrieved bucket encryption", nil
		},
	)

	return err
}

func AssertS3BucketBlockPublicACLEnabled(t *testing.T, region string, bucketName string) {
	err := AssertS3BucketPublicAccessBlockConfigurationEnabledE(t, region, bucketName, "BlockPublicAcls")
	require.NoError(t, err)
}

func AssertS3BucketBlockPublicPolicyEnabled(t *testing.T, region string, bucketName string) {
	err := AssertS3BucketPublicAccessBlockConfigurationEnabledE(t, region, bucketName, "BlockPublicPolicy")
	require.NoError(t, err)
}

func AssertS3BucketIgnorePublicACLEnabled(t *testing.T, region string, bucketName string) {
	err := AssertS3BucketPublicAccessBlockConfigurationEnabledE(t, region, bucketName, "IgnorePublicAcls")
	require.NoError(t, err)
}

func AssertS3BucketRestrictPublicBucketsEnabled(t *testing.T, region string, bucketName string) {
	err := AssertS3BucketPublicAccessBlockConfigurationEnabledE(t, region, bucketName, "RestrictPublicBuckets")
	require.NoError(t, err)
}

func AssertS3BucketPublicAccessBlockConfigurationEnabledE(t *testing.T, region string, bucketName string, configType string) error {
	config := GetPublicAccessBlockConfiguration(t, region, bucketName)

	expected := true
	switch configType {
	case "BlockPublicAcls":
		if *config.BlockPublicAcls != expected {
			return fmt.Errorf("Block public ACLs not enabled")
		}
	case "BlockPublicPolicy":
		if *config.BlockPublicPolicy != expected {
			return fmt.Errorf("Block public policy not enabled")
		}
	case "IgnorePublicAcls":
		if *config.IgnorePublicAcls != expected {
			return fmt.Errorf("Ignore public ACLs not enabled")
		}
	case "RestrictPublicBuckets":
		if *config.RestrictPublicBuckets != expected {
			return fmt.Errorf("Restrict public buckets not enabled")
		}
	default:
		return fmt.Errorf("Unrecognized public access block configuration type")
	}

	return nil
}

func AssertS3BucketLoggingEnabled(t *testing.T, region string, bucketName string, loggingBucketName string) {
	err := AssertS3BucketLoggingEnabledE(t, region, bucketName, loggingBucketName)
	require.NoError(t, err)
}

func AssertS3BucketLoggingEnabledE(t *testing.T, region string, bucketName string, loggingBucketName string) error {
	s3Client, err := aws.NewS3ClientE(t, region)

	if err != nil {
		return err
	}

	params := &s3.GetBucketLoggingInput{
		Bucket: awssdk.String(bucketName),
	}

	bucketLogging, err := s3Client.GetBucketLogging(params)

	if err != nil {
		return err
	}

	loggingEnabled := bucketLogging.LoggingEnabled

	if loggingEnabled == nil {
		return fmt.Errorf("Logging not enabled")
	}

	actual := *loggingEnabled.TargetBucket
	expected := loggingBucketName
	if actual != expected {
		return fmt.Errorf("Logging target bucket does not match expected. Got: %v, Expected: %v", actual, expected)
	}

	return nil
}

func TestTerraformAwsS3PrivateBucket(t *testing.T) {
	t.Parallel()

	// Give this S3 Bucket a unique ID for a name tag so we can distinguish it from any other Buckets provisioned
	// in your AWS account
	testName := fmt.Sprintf("terratest-aws-s3-private-bucket-%s", strings.ToLower(random.UniqueId()))
	loggingBucket := fmt.Sprintf("%s-logs", testName)

	// Pick a random AWS region to test in. This helps ensure your code works in all regions.
	awsRegion := aws.GetRandomStableRegion(t, nil, nil)

	terraformOptions := &terraform.Options{
		// The path to where our Terraform code is located
		TerraformDir: "../examples/simple",

		// Variables to pass to our Terraform code using -var options
		Vars: map[string]interface{}{
			"test_name":      testName,
			"logging_bucket": loggingBucket,
			"region":         awsRegion,
		},

		// Environment variables to set when running Terraform
		EnvVars: map[string]string{
			"AWS_DEFAULT_REGION": awsRegion,
		},
	}

	// At the end of the test, run `terraform destroy` to clean up any resources that were created
	defer terraform.Destroy(t, terraformOptions)

	// This will run `terraform init` and `terraform apply` and fail the test if there are any errors
	terraform.InitAndApply(t, terraformOptions)

	AssertS3BucketEncryptionEnabled(t, awsRegion, testName)
	aws.AssertS3BucketVersioningExists(t, awsRegion, testName)
	AssertS3BucketBlockPublicACLEnabled(t, awsRegion, testName)
	AssertS3BucketBlockPublicPolicyEnabled(t, awsRegion, testName)
	AssertS3BucketIgnorePublicACLEnabled(t, awsRegion, testName)
	AssertS3BucketRestrictPublicBucketsEnabled(t, awsRegion, testName)
	AssertS3BucketLoggingEnabled(t, awsRegion, testName, loggingBucket)
}

func TestTerraformAwsS3PrivateBucketCustomPolicy(t *testing.T) {
	t.Parallel()

	testName := fmt.Sprintf("terratest-aws-s3-private-bucket-%s", strings.ToLower(random.UniqueId()))
	loggingBucket := fmt.Sprintf("%s-logs", testName)

	awsRegion := aws.GetRandomStableRegion(t, nil, nil)

	terraformOptions := &terraform.Options{
		// The path to where our Terraform code is located
		TerraformDir: "../examples/custom-bucket-policy",

		// Variables to pass to our Terraform code using -var options
		Vars: map[string]interface{}{
			"test_name":      testName,
			"logging_bucket": loggingBucket,
			"region":         awsRegion,
		},

		// Environment variables to set when running Terraform
		EnvVars: map[string]string{
			"AWS_DEFAULT_REGION": awsRegion,
		},
	}

	defer terraform.Destroy(t, terraformOptions)

	terraform.InitAndApply(t, terraformOptions)

	aws.AssertS3BucketPolicyExists(t, awsRegion, testName)
}
