package accessruntime

import (
	"strings"
	"testing"

	cloudv1 "github.com/anytty/anytty/proto/cloud/v1"
)

func TestCloudEntitlementRuntimeStatusIsActionable(t *testing.T) {
	state, detail := cloudEntitlementRuntimeStatus(&cloudv1.CloudEntitlementFailure{
		Code:  cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_DAEMON_LIMIT_EXHAUSTED,
		Limit: 2,
	})
	if state != "quota_limited" || !strings.Contains(detail, "limit 2") || !strings.Contains(detail, "cloud.anytty.com/app/subscription") || !strings.Contains(detail, "Direct and SSH") {
		t.Fatalf("daemon limit status = %q / %q", state, detail)
	}

	state, detail = cloudEntitlementRuntimeStatus(&cloudv1.CloudEntitlementFailure{Code: cloudv1.CloudEntitlementErrorCode_CLOUD_ENTITLEMENT_ERROR_CODE_SUBSCRIPTION_INACTIVE})
	if state != "subscription_inactive" || !strings.Contains(detail, "renew") {
		t.Fatalf("subscription status = %q / %q", state, detail)
	}
}
