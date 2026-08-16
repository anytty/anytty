package apilayer

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/anytty/anytty/proto/apipb"
)

type fakeAdmission struct {
	capabilities []apipb.ApiCapability
	err          error
	authorize    func(*apipb.CommandEnvelope) error
	acquired     int
	released     int
}

func (admission *fakeAdmission) Acquire(_ context.Context, command *apipb.CommandEnvelope, required apipb.ApiCapability) (AdmissionLease, error) {
	if admission.err != nil {
		return nil, admission.err
	}
	if required != apipb.ApiCapability_API_CAPABILITY_UNSPECIFIED && !containsTestCapability(admission.capabilities, required) {
		return nil, ErrAdmissionUnsupportedCapability
	}
	if admission.authorize != nil {
		if err := admission.authorize(command); err != nil {
			return nil, err
		}
	}
	admission.acquired++
	return &fakeAdmissionLease{admission: admission}, nil
}

type fakeAdmissionLease struct {
	admission *fakeAdmission
	released  bool
}

func (lease *fakeAdmissionLease) Release() {
	if lease == nil || lease.released {
		return
	}
	lease.released = true
	lease.admission.released++
}

type fakeOperationController struct {
	operations []*apipb.OperationStamp
	err        error
	onCancel   func()
}

func (controller *fakeOperationController) CancelOperation(_ context.Context, operation *apipb.OperationStamp) error {
	if controller.onCancel != nil {
		controller.onCancel()
	}
	controller.operations = append(controller.operations, operation)
	return controller.err
}

type fakeResourceController struct {
	resources []*apipb.ResourceHandle
	err       error
}

func (controller *fakeResourceController) ReleaseResource(_ context.Context, resource *apipb.ResourceHandle) error {
	controller.resources = append(controller.resources, resource)
	return controller.err
}

func TestServiceExecutesTypedCancelUnderAdmissionLease(t *testing.T) {
	admission := admissionWith(apipb.ApiCapability_API_CAPABILITY_OPERATION_CANCELLATION)
	var snapshotOperation *apipb.OperationStamp
	admission.authorize = func(command *apipb.CommandEnvelope) error {
		snapshotOperation = command.GetCancelOperation().GetOperation()
		return nil
	}
	operations := &fakeOperationController{onCancel: func() {
		if admission.acquired != 1 || admission.released != 0 {
			t.Fatalf("admission lease not held during controller call: %d/%d", admission.acquired, admission.released)
		}
	}}
	service := NewService(admission, operations, nil, nil)
	result := service.Execute(context.Background(), cancelCommand("request-1", 1))
	if result.GetAcknowledge() == nil || result.GetError() != nil || result.GetOriginSession().GetGeneration() != 7 {
		t.Fatalf("cancel result=%#v", result)
	}
	if len(operations.operations) != 1 || operations.operations[0].GetSession().GetEndpointId() != "studio" ||
		operations.operations[0].GetSession().GetRouteId() != "ssh" || operations.operations[0].GetSession().GetGeneration() != 7 ||
		operations.operations[0].GetOperationId() != "operation-1" {
		t.Fatalf("cancel operations=%#v", operations.operations)
	}
	if operations.operations[0] != snapshotOperation {
		t.Fatalf("cancel cloned private snapshot operation: controller=%p admission=%p", operations.operations[0], snapshotOperation)
	}
	if admission.acquired != 1 || admission.released != 1 {
		t.Fatalf("admission acquire/release=%d/%d", admission.acquired, admission.released)
	}
}

func TestServiceEchoesClientOwnedOriginGeneration(t *testing.T) {
	operations := &fakeOperationController{}
	command := cancelCommand("request-origin", 1)
	command.GetContext().GetSession().Generation = 6
	command.GetCancelOperation().GetOperation().GetSession().Generation = 6
	result := NewService(admissionWith(apipb.ApiCapability_API_CAPABILITY_OPERATION_CANCELLATION), operations, nil, nil).Execute(context.Background(), command)
	if result.GetAcknowledge() == nil || result.GetOriginSession().GetGeneration() != 6 || len(operations.operations) != 1 {
		t.Fatalf("origin session result=%#v calls=%d", result, len(operations.operations))
	}
}

func TestServiceRejectsOperationFenceThatDiffersFromRequest(t *testing.T) {
	operations := &fakeOperationController{}
	command := cancelCommand("request-operation", 1)
	command.GetCancelOperation().GetOperation().GetSession().Generation = 6
	result := NewService(admissionWith(apipb.ApiCapability_API_CAPABILITY_OPERATION_CANCELLATION), operations, nil, nil).Execute(context.Background(), command)
	if result.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST || result.GetError().GetAttempted() {
		t.Fatalf("operation fence result=%#v", result)
	}
}

func TestServiceUsesAdmissionCapabilitiesInsteadOfClientClaims(t *testing.T) {
	operations := &fakeOperationController{}
	result := NewService(admissionWith(), operations, nil, nil).Execute(context.Background(), cancelCommand("request-capability", 1))
	if result.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_UNSUPPORTED_CAPABILITY || len(operations.operations) != 0 {
		t.Fatalf("capability result=%#v calls=%d", result, len(operations.operations))
	}
}

func TestServiceMapsAdmissionAccessFailures(t *testing.T) {
	unauthorized := NewService(&fakeAdmission{err: ErrAdmissionUnauthorized}, nil, nil, nil).Execute(context.Background(), cancelCommand("request-auth", 1))
	if unauthorized.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_UNAUTHORIZED {
		t.Fatalf("unauthorized result=%#v", unauthorized)
	}
	forbidden := NewService(&fakeAdmission{err: ErrAdmissionForbidden}, nil, nil, nil).Execute(context.Background(), cancelCommand("request-forbidden", 1))
	if forbidden.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_FORBIDDEN {
		t.Fatalf("forbidden result=%#v", forbidden)
	}
}

func TestServiceDelegatesCommandLevelAuthorizationToAdmission(t *testing.T) {
	admission := admissionWith(apipb.ApiCapability_API_CAPABILITY_OPERATION_CANCELLATION)
	admission.authorize = func(command *apipb.CommandEnvelope) error {
		if command.GetCancelOperation() != nil {
			return ErrAdmissionForbidden
		}
		return nil
	}
	operations := &fakeOperationController{}
	result := NewService(admission, operations, nil, nil).Execute(context.Background(), cancelCommand("request-scope", 1))
	if result.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_FORBIDDEN || len(operations.operations) != 0 {
		t.Fatalf("command authorization result=%#v calls=%d", result, len(operations.operations))
	}
}

func TestServiceReportsControllerFailureAsAttempted(t *testing.T) {
	operations := &fakeOperationController{err: errors.New("write failed")}
	result := NewService(admissionWith(apipb.ApiCapability_API_CAPABILITY_OPERATION_CANCELLATION), operations, nil, nil).Execute(context.Background(), cancelCommand("request-3", 1))
	if result.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_INTERNAL || !result.GetError().GetAttempted() {
		t.Fatalf("controller failure=%#v", result)
	}
}

func TestServiceRequiresVersionAndControllers(t *testing.T) {
	resources := &fakeResourceController{}
	unsupportedVersion := NewService(admissionWith(apipb.ApiCapability_API_CAPABILITY_RESOURCE_LIFECYCLE), nil, resources, nil).Execute(context.Background(), releaseCommand("request-4", 2))
	if unsupportedVersion.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_UNSUPPORTED_VERSION {
		t.Fatalf("version result=%#v", unsupportedVersion)
	}
	unavailable := NewService(admissionWith(apipb.ApiCapability_API_CAPABILITY_RESOURCE_LIFECYCLE), nil, nil, nil).Execute(context.Background(), releaseCommand("request-6", 1))
	if unavailable.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE || unavailable.GetError().GetAttempted() || !unavailable.GetError().GetRetryable() {
		t.Fatalf("unavailable result=%#v", unavailable)
	}
}

func TestServiceRejectsResourceFromDifferentSession(t *testing.T) {
	resources := &fakeResourceController{}
	command := releaseCommand("request-resource-session", 1)
	command.GetReleaseResource().GetResource().GetSession().Generation = 6
	result := NewService(admissionWith(apipb.ApiCapability_API_CAPABILITY_RESOURCE_LIFECYCLE), nil, resources, nil).Execute(context.Background(), command)
	if result.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST || len(resources.resources) != 0 {
		t.Fatalf("resource ownership result=%#v calls=%d", result, len(resources.resources))
	}
}

func TestServiceReleasesResourceFromPrivateEnvelopeSnapshot(t *testing.T) {
	resources := &fakeResourceController{}
	command := releaseCommand("request-resource", 1)
	admission := admissionWith(apipb.ApiCapability_API_CAPABILITY_RESOURCE_LIFECYCLE)
	var snapshotResource *apipb.ResourceHandle
	admission.authorize = func(command *apipb.CommandEnvelope) error {
		snapshotResource = command.GetReleaseResource().GetResource()
		return nil
	}
	result := NewService(admission, nil, resources, nil).Execute(context.Background(), command)
	if result.GetAcknowledge() == nil || len(resources.resources) != 1 {
		t.Fatalf("release result=%#v resources=%#v", result, resources.resources)
	}
	if resources.resources[0] != snapshotResource {
		t.Fatalf("release cloned private snapshot resource: controller=%p admission=%p", resources.resources[0], snapshotResource)
	}
	resources.resources[0].OpaqueToken[0] = 'X'
	if string(command.GetReleaseResource().GetResource().GetOpaqueToken()) != "resource-1" {
		t.Fatalf("controller mutated client command: %#v", command.GetReleaseResource().GetResource())
	}
}

func TestServiceHonorsContextCancellationBeforeAdmission(t *testing.T) {
	operations := &fakeOperationController{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result := NewService(admissionWith(apipb.ApiCapability_API_CAPABILITY_OPERATION_CANCELLATION), operations, nil, nil).Execute(ctx, cancelCommand("request-7", 1))
	if result.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_CANCELLED || result.GetError().GetAttempted() {
		t.Fatalf("cancelled result=%#v", result)
	}
	if len(operations.operations) != 0 {
		t.Fatalf("cancelled context reached controller: %#v", operations.operations)
	}
}

func TestServiceMapsCancellationReturnedByAdmission(t *testing.T) {
	operations := &fakeOperationController{}
	result := NewService(&fakeAdmission{err: context.Canceled}, operations, nil, nil).Execute(context.Background(), cancelCommand("request-admission-cancel", 1))
	if result.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_CANCELLED || result.GetError().GetRetryable() || len(operations.operations) != 0 {
		t.Fatalf("admission cancellation result=%#v calls=%d", result, len(operations.operations))
	}
}

func TestServicePreservesCorrelationForUnknownCommand(t *testing.T) {
	command := &apipb.CommandEnvelope{Context: requestContext("future-request", 1)}
	result := NewService(admissionWith(), nil, nil, nil).Execute(context.Background(), command)
	if result.GetRequestId() != "future-request" || result.GetOriginSession().GetGeneration() != 7 || result.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST {
		t.Fatalf("unknown command result=%#v", result)
	}
}

func TestServiceDoesNotEchoOversizedCorrelation(t *testing.T) {
	command := cancelCommand(strings.Repeat("r", 300), 1)
	command.GetContext().GetSession().EndpointId = strings.Repeat("e", 300)
	result := NewService(admissionWith(apipb.ApiCapability_API_CAPABILITY_OPERATION_CANCELLATION), nil, nil, nil).Execute(context.Background(), command)
	if result.GetRequestId() != "" || result.GetOriginSession() != nil || result.GetError().GetCode() != apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST {
		t.Fatalf("unsafe correlation response=%#v", result)
	}
}

func admissionWith(capabilities ...apipb.ApiCapability) *fakeAdmission {
	return &fakeAdmission{capabilities: append([]apipb.ApiCapability(nil), capabilities...)}
}

func containsTestCapability(capabilities []apipb.ApiCapability, required apipb.ApiCapability) bool {
	for _, capability := range capabilities {
		if capability == required {
			return true
		}
	}
	return false
}

func cancelCommand(requestID string, major uint32) *apipb.CommandEnvelope {
	contextMessage := requestContext(requestID, major)
	return &apipb.CommandEnvelope{
		Context: contextMessage,
		Command: &apipb.CommandEnvelope_CancelOperation{CancelOperation: &apipb.CancelOperationCommand{Operation: &apipb.OperationStamp{
			Session: &apipb.EndpointSessionStamp{EndpointId: "studio", RouteId: "ssh", Generation: 7}, OperationId: "operation-1",
		}}},
	}
}

func releaseCommand(requestID string, major uint32) *apipb.CommandEnvelope {
	contextMessage := requestContext(requestID, major)
	return &apipb.CommandEnvelope{
		Context: contextMessage,
		Command: &apipb.CommandEnvelope_ReleaseResource{ReleaseResource: &apipb.ReleaseResourceCommand{Resource: &apipb.ResourceHandle{
			OpaqueToken: []byte("resource-1"), Kind: apipb.ResourceKind_RESOURCE_KIND_SUBSCRIPTION,
			Session: &apipb.EndpointSessionStamp{EndpointId: "studio", RouteId: "ssh", Generation: 7}, Generation: 3,
		}}},
	}
}

func requestContext(requestID string, major uint32) *apipb.RequestContext {
	return &apipb.RequestContext{
		RequestId: requestID, ApiVersion: &apipb.ApiVersion{Major: major},
		Session: &apipb.EndpointSessionStamp{EndpointId: "studio", RouteId: "ssh", Generation: 7},
	}
}
