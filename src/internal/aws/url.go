package aws

import (
	"fmt"
	"strings"
)

func GetEcrUrl(accountID, region string) string {
	accountID = strings.TrimSpace(accountID)
	region = strings.TrimSpace(region)
	return fmt.Sprintf("%s.dkr.ecr.%s.amazonaws.com", accountID, region)
}
