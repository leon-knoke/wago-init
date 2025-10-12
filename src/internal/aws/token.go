package aws

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/ecr"
)

const ProfileName = "wago-init"

func FetchLoginPassword(ctx context.Context, region, accessID, secretKey string) (string, error) {
	region = strings.TrimSpace(region)
	accessID = strings.TrimSpace(accessID)
	secretKey = strings.TrimSpace(secretKey)

	if region == "" {
		return "", errors.New("aws region is required")
	}
	if accessID == "" {
		return "", errors.New("aws access id is required")
	}
	if secretKey == "" {
		return "", errors.New("aws access key is required")
	}

	cfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessID, secretKey, "")),
	)
	if err != nil {
		return "", fmt.Errorf("load aws config: %w", err)
	}

	client := ecr.NewFromConfig(cfg)
	output, err := client.GetAuthorizationToken(ctx, &ecr.GetAuthorizationTokenInput{})
	if err != nil {
		return "", fmt.Errorf("request authorization token: %w", err)
	}

	if len(output.AuthorizationData) == 0 || output.AuthorizationData[0].AuthorizationToken == nil {
		return "", errors.New("authorization token not returned")
	}

	decoded, err := base64.StdEncoding.DecodeString(*output.AuthorizationData[0].AuthorizationToken)
	if err != nil {
		return "", fmt.Errorf("decode authorization token: %w", err)
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", errors.New("unexpected authorization token format")
	}

	return parts[1], nil
}
