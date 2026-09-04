package apimapping

import (
	"errors"
	"fmt"
	"math"
	"time"

	corev2 "github.com/anytty/anytty/core"
	corehistory "github.com/anytty/anytty/core/history"
	"github.com/anytty/anytty/proto/apipb"
)

const (
	maxTerminalInputBytes = 1 << 20
	maxPathPrefixBytes    = 4096
	maxPathEntries        = 1000
	maxCommandArguments   = 256
	maxEnvironmentEntries = 4096
	maxTagEntries         = 256
	maxAggregateTextBytes = 1 << 20
	maxMetadataTextBytes  = 4096
)

// RequestContextForCommand 返回 envelope 顶层唯一的公共 request context。
// context 位于 oneof 外，旧服务即使不认识未来 command 也能保留 request correlation。
func RequestContextForCommand(command *apipb.CommandEnvelope) *apipb.RequestContext {
	if command == nil {
		return nil
	}
	return command.GetContext()
}

// ValidateTerminalCommand 校验 terminal/path command 的 identity、枚举、resource 和 operation fence。
func ValidateTerminalCommand(command *apipb.CommandEnvelope) error {
	requestContext := RequestContextForCommand(command)
	if err := ValidateRequestContext(requestContext); err != nil {
		return err
	}
	if !terminalCommandPayloadPresent(command) {
		return validation("command", "typed command payload is required")
	}
	switch value := command.GetCommand().(type) {
	case *apipb.CommandEnvelope_TerminalDefaults, *apipb.CommandEnvelope_TerminalList:
		return requireSession(requestContext)
	case *apipb.CommandEnvelope_TerminalCreate:
		if err := requireSession(requestContext); err != nil {
			return err
		}
		return ValidateTerminalCreateSpec(value.TerminalCreate.GetTerminal())
	case *apipb.CommandEnvelope_TerminalGet:
		return validateTerminalRefForContext(value.TerminalGet.GetTerminal(), requestContext)
	case *apipb.CommandEnvelope_TerminalRestart:
		return validateTerminalRefForContext(value.TerminalRestart.GetTerminal(), requestContext)
	case *apipb.CommandEnvelope_TerminalKill:
		return validateTerminalRefForContext(value.TerminalKill.GetTerminal(), requestContext)
	case *apipb.CommandEnvelope_TerminalRemove:
		return validateTerminalRefForContext(value.TerminalRemove.GetTerminal(), requestContext)
	case *apipb.CommandEnvelope_TerminalSetMetadata:
		if err := validateTerminalRefForContext(value.TerminalSetMetadata.GetTerminal(), requestContext); err != nil {
			return err
		}
		if len(value.TerminalSetMetadata.GetName()) > maxMetadataTextBytes {
			return validation("terminal_set_metadata.name", "exceeds 4096 bytes")
		}
		return validateStringMap("terminal_set_metadata.tags", value.TerminalSetMetadata.GetTags())
	case *apipb.CommandEnvelope_TerminalSetTags:
		if err := validateTerminalRefForContext(value.TerminalSetTags.GetTerminal(), requestContext); err != nil {
			return err
		}
		return validateStringMap("terminal_set_tags.tags", value.TerminalSetTags.GetTags())
	case *apipb.CommandEnvelope_TerminalAttach:
		attach := value.TerminalAttach
		if err := validateTerminalRefForContext(attach.GetTerminal(), requestContext); err != nil {
			return err
		}
		if !validAttachmentMode(attach.GetMode()) {
			return validation("terminal_attach.mode", "must be specified")
		}
		if !validResizePolicy(attach.GetResizePolicy()) {
			return validation("terminal_attach.resize_policy", "must be specified")
		}
		if attach.GetMode() == apipb.AttachmentMode_ATTACHMENT_MODE_OBSERVER && attach.GetResizePolicy() != apipb.ResizePolicy_RESIZE_POLICY_OBSERVER {
			return validation("terminal_attach.resize_policy", "observer mode requires observer resize policy")
		}
		if attach.GetSurfaceId() == "" || attach.GetViewId() == "" {
			return validation("terminal_attach.surface_id", "surface_id and view_id are required")
		}
		if len(attach.GetSurfaceId()) > maxAPIIdentityBytes || len(attach.GetViewId()) > maxAPIIdentityBytes {
			return validation("terminal_attach.surface_id", "surface_id or view_id exceeds 256 bytes")
		}
		return ValidateOperationStamp(attach.GetOperation(), requestContext.GetSession())
	case *apipb.CommandEnvelope_TerminalDetach:
		return validateAttachmentOperation(value.TerminalDetach.GetAttachment(), value.TerminalDetach.GetOperation(), requestContext)
	case *apipb.CommandEnvelope_TerminalInput:
		if len(value.TerminalInput.GetData()) == 0 {
			return validation("terminal_input.data", "must not be empty")
		}
		if len(value.TerminalInput.GetData()) > maxTerminalInputBytes {
			return validation("terminal_input.data", "exceeds 1 MiB")
		}
		return validateAttachmentOperation(value.TerminalInput.GetAttachment(), value.TerminalInput.GetOperation(), requestContext)
	case *apipb.CommandEnvelope_TerminalResize:
		resize := value.TerminalResize
		if err := ValidateTerminalSize(resize.GetSize()); err != nil {
			return err
		}
		if !validResizePolicy(resize.GetResizePolicy()) {
			return validation("terminal_resize.resize_policy", "must be specified")
		}
		return validateAttachmentOperation(resize.GetAttachment(), resize.GetOperation(), requestContext)
	case *apipb.CommandEnvelope_TerminalResizeLock:
		return validateAttachmentOperation(value.TerminalResizeLock.GetAttachment(), value.TerminalResizeLock.GetOperation(), requestContext)
	case *apipb.CommandEnvelope_PathListDirectories:
		if err := requireSession(requestContext); err != nil {
			return err
		}
		if value.PathListDirectories.GetLimit() < 0 {
			return validation("path_list_directories.limit", "must not be negative")
		}
		if value.PathListDirectories.GetLimit() > maxPathEntries {
			return validation("path_list_directories.limit", "exceeds 1000 entries")
		}
		if len(value.PathListDirectories.GetPrefix()) > maxPathPrefixBytes {
			return validation("path_list_directories.prefix", "exceeds 4096 bytes")
		}
		return nil
	default:
		return validation("command", "is not a terminal or path command")
	}
}

func terminalCommandPayloadPresent(command *apipb.CommandEnvelope) bool {
	switch value := command.GetCommand().(type) {
	case *apipb.CommandEnvelope_TerminalDefaults:
		return value.TerminalDefaults != nil
	case *apipb.CommandEnvelope_TerminalCreate:
		return value.TerminalCreate != nil
	case *apipb.CommandEnvelope_TerminalList:
		return value.TerminalList != nil
	case *apipb.CommandEnvelope_TerminalGet:
		return value.TerminalGet != nil
	case *apipb.CommandEnvelope_TerminalRestart:
		return value.TerminalRestart != nil
	case *apipb.CommandEnvelope_TerminalKill:
		return value.TerminalKill != nil
	case *apipb.CommandEnvelope_TerminalRemove:
		return value.TerminalRemove != nil
	case *apipb.CommandEnvelope_TerminalSetMetadata:
		return value.TerminalSetMetadata != nil
	case *apipb.CommandEnvelope_TerminalSetTags:
		return value.TerminalSetTags != nil
	case *apipb.CommandEnvelope_TerminalAttach:
		return value.TerminalAttach != nil
	case *apipb.CommandEnvelope_TerminalDetach:
		return value.TerminalDetach != nil
	case *apipb.CommandEnvelope_TerminalInput:
		return value.TerminalInput != nil
	case *apipb.CommandEnvelope_TerminalResize:
		return value.TerminalResize != nil
	case *apipb.CommandEnvelope_TerminalResizeLock:
		return value.TerminalResizeLock != nil
	case *apipb.CommandEnvelope_PathListDirectories:
		return value.PathListDirectories != nil
	default:
		return false
	}
}

// ValidateTerminalCreateSpec 校验 daemon-local terminal 创建规格。
func ValidateTerminalCreateSpec(spec *apipb.TerminalCreateSpec) error {
	if spec == nil {
		return validation("terminal_create.terminal", "is required")
	}
	if spec.GetTerminalId() == "" {
		return validation("terminal_create.terminal.terminal_id", "is required")
	}
	if len(spec.GetCommand()) == 0 {
		return validation("terminal_create.terminal.command", "must not be empty")
	}
	if len(spec.GetTerminalId()) > 256 {
		return validation("terminal_create.terminal.terminal_id", "exceeds 256 bytes")
	}
	if len(spec.GetName()) > maxMetadataTextBytes || len(spec.GetCwd()) > maxMetadataTextBytes {
		return validation("terminal_create.terminal", "name or cwd exceeds 4096 bytes")
	}
	if err := validateStringSlice("terminal_create.terminal.command", spec.GetCommand(), maxCommandArguments); err != nil {
		return err
	}
	if err := validateStringSlice("terminal_create.terminal.env", spec.GetEnv(), maxEnvironmentEntries); err != nil {
		return err
	}
	if err := validateStringMap("terminal_create.terminal.tags", spec.GetTags()); err != nil {
		return err
	}
	if spec.GetScrollbackRows() < 0 {
		return validation("terminal_create.terminal.scrollback_rows", "must not be negative")
	}
	if spec.GetScrollbackMaxBytes() < 0 {
		return validation("terminal_create.terminal.scrollback_max_bytes", "must not be negative")
	}
	if spec.GetScrollbackMaxAgeSeconds() < 0 || spec.GetScrollbackMaxAgeSeconds() > math.MaxInt64/int64(time.Second) {
		return validation("terminal_create.terminal.scrollback_max_age_seconds", "is outside supported duration range")
	}
	return ValidateTerminalSize(spec.GetSize())
}

// ValidateTerminalSize 校验 terminal cell 尺寸。
func ValidateTerminalSize(size *apipb.TerminalSize) error {
	if size == nil || size.GetCols() == 0 || size.GetRows() == 0 {
		return validation("terminal.size", "cols and rows must be greater than zero")
	}
	if size.GetCols() > 65535 || size.GetRows() > 65535 {
		return validation("terminal.size", "cols and rows exceed daemon limits")
	}
	return nil
}

// ValidateTerminalAttachResult 校验 controller 发布的 attachment handle 与已授权原始 attach command 完全一致。
// opaque token 真伪由 owning resource registry 保证；这里拒绝畸形、跨 session 或未知 enum 结果。
func ValidateTerminalAttachResult(command *apipb.TerminalAttachCommand, result *apipb.TerminalAttachResult, session *apipb.EndpointSessionStamp) error {
	if command == nil {
		return fmt.Errorf("terminal attach publication requires the authorized command")
	}
	if result == nil || result.GetAttachment() == nil {
		return fmt.Errorf("terminal attach controller returned no attachment handle")
	}
	handle := result.GetAttachment()
	if err := ValidateResourceHandle(handle.GetResource()); err != nil {
		return fmt.Errorf("invalid terminal attachment resource: %w", err)
	}
	if handle.GetResource().GetKind() != apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT {
		return fmt.Errorf("terminal attachment result has the wrong resource kind")
	}
	if !SessionStampsEqual(handle.GetResource().GetSession(), session) {
		return fmt.Errorf("terminal attachment result belongs to a different origin session")
	}
	if handle.GetTerminal().GetEndpointId() != command.GetTerminal().GetEndpointId() || handle.GetTerminal().GetTerminalId() != command.GetTerminal().GetTerminalId() {
		return fmt.Errorf("terminal attachment result identifies a different terminal")
	}
	if err := ValidateOperationStamp(handle.GetOperation(), session); err != nil {
		return fmt.Errorf("invalid terminal attachment operation: %w", err)
	}
	if handle.GetOperation().GetOperationId() != command.GetOperation().GetOperationId() {
		return fmt.Errorf("terminal attachment result identifies a different operation")
	}
	if handle.GetSurfaceId() != command.GetSurfaceId() || handle.GetViewId() != command.GetViewId() {
		return fmt.Errorf("terminal attachment result identifies a different surface or view")
	}
	if result.GetMode() != command.GetMode() || !validAttachmentMode(result.GetMode()) {
		return fmt.Errorf("terminal attachment result mode differs from the authorized command")
	}
	if result.GetResizePolicy() != command.GetResizePolicy() || !validResizePolicy(result.GetResizePolicy()) {
		return fmt.Errorf("terminal attachment result resize policy differs from the authorized command")
	}
	if err := ValidateTerminalSize(result.GetSize()); err != nil {
		return fmt.Errorf("invalid terminal attachment result size: %w", err)
	}
	return nil
}

// ApplicationAdmissionFromCommand 把公共 capability 与 command target 映射为 core connection admission。
// 该函数不查询 attachment registry；resource token 的真实性只由 owning core session 验证。
func ApplicationAdmissionFromCommand(command *apipb.CommandEnvelope, capability apipb.ApiCapability) corev2.ApplicationAdmission {
	admission := corev2.ApplicationAdmission{Capability: applicationCapabilityToCore(capability)}
	if command == nil {
		return admission
	}
	switch value := command.GetCommand().(type) {
	case *apipb.CommandEnvelope_TerminalList:
		admission.Capability = corev2.ApplicationCapabilityTerminalInventory
	case *apipb.CommandEnvelope_TerminalGet:
		admission.TerminalID = value.TerminalGet.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_TerminalRestart:
		admission.TerminalID = value.TerminalRestart.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_TerminalKill:
		admission.TerminalID = value.TerminalKill.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_TerminalRemove:
		admission.TerminalID = value.TerminalRemove.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_TerminalSetMetadata:
		admission.TerminalID = value.TerminalSetMetadata.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_TerminalSetTags:
		admission.TerminalID = value.TerminalSetTags.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_TerminalAttach:
		admission.TerminalID = value.TerminalAttach.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_TerminalDetach:
		admission.ResourceToken = cloneBytes(value.TerminalDetach.GetAttachment().GetOpaqueToken())
	case *apipb.CommandEnvelope_TerminalInput:
		admission.ResourceToken = cloneBytes(value.TerminalInput.GetAttachment().GetOpaqueToken())
	case *apipb.CommandEnvelope_TerminalResize:
		admission.ResourceToken = cloneBytes(value.TerminalResize.GetAttachment().GetOpaqueToken())
	case *apipb.CommandEnvelope_TerminalResizeLock:
		admission.ResourceToken = cloneBytes(value.TerminalResizeLock.GetAttachment().GetOpaqueToken())
	case *apipb.CommandEnvelope_ReleaseResource:
		admission.ResourceToken = cloneBytes(value.ReleaseResource.GetResource().GetOpaqueToken())
		if value.ReleaseResource.GetResource().GetKind() == apipb.ResourceKind_RESOURCE_KIND_SUBSCRIPTION {
			admission.ResourceKind = corev2.ApplicationResourceKindSubscription
		}
	case *apipb.CommandEnvelope_HistoryWindow:
		admission.TerminalID = value.HistoryWindow.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_HistoryCopy:
		admission.TerminalID = value.HistoryCopy.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_HistoryRelease:
		admission.TerminalID = value.HistoryRelease.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_HistoryBacklogStatus:
		admission.TerminalID = value.HistoryBacklogStatus.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_LiveScreenNext:
		admission.TerminalID = value.LiveScreenNext.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_HistorySearch:
		admission.TerminalID = value.HistorySearch.GetTerminal().GetTerminalId()
	case *apipb.CommandEnvelope_EventSubscribe:
		admission.TerminalID = value.EventSubscribe.GetTerminal().GetTerminalId()
		admission.MachineLifecycleEventsOnly = machineLifecycleEventsOnly(value.EventSubscribe)
	case *apipb.CommandEnvelope_FileList:
		admission.FileOperation = "list"
	case *apipb.CommandEnvelope_FileStat:
		admission.FileOperation = "stat"
	case *apipb.CommandEnvelope_FilePreview:
		admission.FileOperation = "preview"
	case *apipb.CommandEnvelope_FileMkdir:
		admission.FileOperation = "mkdir"
	case *apipb.CommandEnvelope_FileRename:
		admission.FileOperation = "rename"
	case *apipb.CommandEnvelope_FileDelete:
		admission.FileOperation = "delete"
	case *apipb.CommandEnvelope_FileCopy:
		admission.FileOperation = "copy"
	case *apipb.CommandEnvelope_FileMove:
		admission.FileOperation = "move"
	case *apipb.CommandEnvelope_FileDownloadOpen:
		admission.FileOperation = "download"
	case *apipb.CommandEnvelope_FileUploadOpen:
		admission.FileOperation = "upload"
	case *apipb.CommandEnvelope_FileTransferCancel:
		admission.FileOperation = "cancel"
		admission.ResourceToken = cloneBytes(value.FileTransferCancel.GetTransfer().GetOpaqueToken())
	}
	return admission
}

func machineLifecycleEventsOnly(command *apipb.EventSubscribeCommand) bool {
	if command == nil || command.GetTerminal() != nil || command.GetStorageAppId() != "" || command.GetStorageScope() != apipb.StorageScope_STORAGE_SCOPE_UNSPECIFIED || command.GetStorageOwnerId() != "" || command.GetStorageKeyPrefix() != "" || len(command.GetTypes()) == 0 {
		return false
	}
	for _, eventType := range command.GetTypes() {
		if eventType != apipb.ApplicationEventType_APPLICATION_EVENT_TYPE_TERMINAL_LIFECYCLE {
			return false
		}
	}
	return true
}

// TerminalRecordFromProto 把已校验 create spec 转换为 core domain record。
func TerminalRecordFromProto(spec *apipb.TerminalCreateSpec) (corev2.TerminalRecord, error) {
	if err := ValidateTerminalCreateSpec(spec); err != nil {
		return corev2.TerminalRecord{}, err
	}
	return corev2.TerminalRecord{
		ID: spec.GetTerminalId(), Name: spec.GetName(), Command: append([]string(nil), spec.GetCommand()...), Tags: cloneStringMap(spec.GetTags()),
		Size: corev2.Size{Cols: uint16(spec.GetSize().GetCols()), Rows: uint16(spec.GetSize().GetRows())},
		Options: corev2.TerminalCreateOptions{
			Dir: spec.GetCwd(), Env: append([]string(nil), spec.GetEnv()...), ScrollbackSize: int(spec.GetScrollbackRows()),
			ScrollbackMaxBytes: spec.GetScrollbackMaxBytes(), ScrollbackMaxAge: time.Duration(spec.GetScrollbackMaxAgeSeconds()) * time.Second,
		},
	}, nil
}

// TerminalInfoToProto 把 core lifecycle snapshot 转换为 endpoint-aware public projection。
func TerminalInfoToProto(endpointID string, info corev2.TerminalInfo, attachmentCount int) (*apipb.TerminalInfo, error) {
	if endpointID == "" || info.ID == "" {
		return nil, fmt.Errorf("terminal projection requires endpoint and terminal identity")
	}
	state, err := terminalStateToProto(info.State)
	if err != nil {
		return nil, err
	}
	if attachmentCount < 0 || attachmentCount > math.MaxInt32 {
		return nil, fmt.Errorf("terminal projection exceeds public API integer range")
	}
	resources, err := terminalResourceUsageToProto(info.Resources)
	if err != nil {
		return nil, err
	}
	resourceHistory := make([]*apipb.TerminalResourceUsage, 0, len(info.ResourceHistory))
	for _, sample := range info.ResourceHistory {
		projection, err := terminalResourceUsageToProto(sample)
		if err != nil {
			return nil, err
		}
		resourceHistory = append(resourceHistory, projection)
	}
	out := &apipb.TerminalInfo{
		Ref: &apipb.TerminalRef{EndpointId: endpointID, TerminalId: info.ID}, Name: info.Name, Command: append([]string(nil), info.Command...),
		Tags: cloneStringMap(info.Tags), Size: TerminalSizeToProto(info.Size), State: state, Cwd: info.CWD, LiveCwd: info.LiveCWD,
		CreatedAtUnixNano: unixNanoOrZero(info.CreatedAt), ExitedAtUnixNano: unixNanoOrZero(info.ExitedAt), AttachmentCount: int32(attachmentCount),
		Resources: resources, ResourceHistory: resourceHistory,
		ForegroundProcess: info.ForegroundProcess, ForegroundCwd: info.ForegroundCWD,
		LastOutputAtUnixNano: unixNanoOrZero(info.LastOutputAt),
	}
	if info.ExitCode != nil {
		if *info.ExitCode < math.MinInt32 || *info.ExitCode > math.MaxInt32 {
			return nil, fmt.Errorf("terminal exit code exceeds public API integer range")
		}
		value := int32(*info.ExitCode)
		out.ExitCode = &value
	}
	return out, nil
}

func terminalResourceUsageToProto(usage corev2.TerminalResourceUsage) (*apipb.TerminalResourceUsage, error) {
	if usage.PID < math.MinInt32 || usage.PID > math.MaxInt32 || usage.CPUPercentX100 < math.MinInt32 || usage.CPUPercentX100 > math.MaxInt32 {
		return nil, fmt.Errorf("terminal resource projection exceeds public API integer range")
	}
	return &apipb.TerminalResourceUsage{
		Pid: int32(usage.PID), CpuPercentX100: int32(usage.CPUPercentX100), MemoryBytes: usage.MemoryBytes, SampledAtUnixNano: unixNanoOrZero(usage.SampledAt),
	}, nil
}

// TerminalDefaultsToProto 把 core daemon defaults 投影为公共 API result。
func TerminalDefaultsToProto(defaults corev2.TerminalDefaults) *apipb.TerminalDefaultsResult {
	return &apipb.TerminalDefaultsResult{Defaults: &apipb.TerminalDefaults{
		DefaultCommand: append([]string(nil), defaults.DefaultCommand...),
		DefaultCwd:     defaults.DefaultCWD,
		Platform:       defaults.Platform,
	}}
}

// TerminalCreateToProto 包装已经完成 endpoint-aware 映射的 terminal projection。
func TerminalCreateToProto(terminal *apipb.TerminalInfo) *apipb.TerminalCreateResult {
	return &apipb.TerminalCreateResult{Terminal: terminal}
}

// TerminalGetToProto 包装已经完成 endpoint-aware 映射的 terminal projection。
func TerminalGetToProto(terminal *apipb.TerminalInfo) *apipb.TerminalGetResult {
	return &apipb.TerminalGetResult{Terminal: terminal}
}

// TerminalListToProto 包装已经完成 endpoint-aware 映射的 terminal projections。
func TerminalListToProto(terminals []*apipb.TerminalInfo) *apipb.TerminalListResult {
	return &apipb.TerminalListResult{Terminals: terminals}
}

// PathDirectoriesToProto 把 core path completion window 投影为公共 API result。
func PathDirectoriesToProto(result corev2.PathDirectories) *apipb.PathListDirectoriesResult {
	out := &apipb.PathListDirectoriesResult{BasePath: result.BasePath, Missing: result.Missing, Truncated: result.Truncated, Entries: make([]*apipb.PathDirectoryEntry, 0, len(result.Entries))}
	for _, entry := range result.Entries {
		out.Entries = append(out.Entries, &apipb.PathDirectoryEntry{Name: entry.Name, Path: entry.Path})
	}
	return out
}

// TerminalAttachmentRequestFromProto 映射 attachment command，不建立或发布资源。
func TerminalAttachmentRequestFromProto(command *apipb.TerminalAttachCommand) corev2.TerminalAttachmentRequest {
	return corev2.TerminalAttachmentRequest{
		TerminalID: command.GetTerminal().GetTerminalId(), Mode: attachmentModeToCore(command.GetMode()),
		ResizePolicy: resizePolicyToCore(command.GetResizePolicy()), SurfaceID: command.GetSurfaceId(), ViewID: command.GetViewId(),
	}
}

// TerminalAttachmentToProto 把 pending core attachment 投影为待 API Layer 校验的公开 handle。
func TerminalAttachmentToProto(origin *apipb.EndpointSessionStamp, command *apipb.TerminalAttachCommand, attachment corev2.TerminalAttachment) *apipb.TerminalAttachResult {
	return &apipb.TerminalAttachResult{
		Attachment: &apipb.AttachmentHandle{
			Resource: &apipb.ResourceHandle{OpaqueToken: cloneBytes(attachment.Token), Kind: apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT, Session: cloneSessionStamp(origin), Generation: 1},
			Terminal: cloneTerminalRef(command.GetTerminal()), Operation: cloneOperationStamp(command.GetOperation()), SurfaceId: command.GetSurfaceId(), ViewId: command.GetViewId(),
		},
		Mode: command.GetMode(), ResizePolicy: command.GetResizePolicy(), Size: TerminalSizeToProto(attachment.Size), ResizeControl: resizeControlToProto(attachment.ResizeControl),
	}
}

// TerminalResizeResultToProto 映射 daemon 确认后的 resize/control 状态。
func TerminalResizeResultToProto(result corev2.TerminalResizeResult) *apipb.TerminalResizeResult {
	return &apipb.TerminalResizeResult{Size: TerminalSizeToProto(result.Size), Resized: result.Resized, ResizeControl: resizeControlToProto(result.ResizeControl)}
}

// TerminalSizeFromProto 把已校验尺寸转换为 core value。
func TerminalSizeFromProto(size *apipb.TerminalSize) corev2.Size {
	return corev2.Size{Cols: uint16(size.GetCols()), Rows: uint16(size.GetRows())}
}

// TerminalSizeToProto 把 core cell size 转换为公共尺寸。
func TerminalSizeToProto(size corev2.Size) *apipb.TerminalSize {
	return &apipb.TerminalSize{Cols: uint32(size.Cols), Rows: uint32(size.Rows)}
}

// ResizePolicyToCore 把公共 resize enum 转换为 core attachment policy。
func ResizePolicyToCore(policy apipb.ResizePolicy) corev2.TerminalResizePolicy {
	return resizePolicyToCore(policy)
}

// CoreError 把 core stable error 分类为公共 typed error 输入。
func CoreError(err error) error {
	if err == nil {
		return nil
	}
	classified := &ClassifiedError{Err: err, Code: apipb.ApiErrorCode_API_ERROR_CODE_INTERNAL}
	var outputErr *corev2.TerminalOutputError
	if errors.As(err, &outputErr) {
		classified.Code = apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE
		classified.Retryable = outputErr.DroppedBytes == 0
		classified.SyncLost = &apipb.OutputSyncLostErrorDetail{
			TerminalId: outputErr.TerminalID, Consumer: outputErr.Consumer,
			ParserEpoch: outputErr.Epoch, DroppedBytes: outputErr.DroppedBytes,
		}
		return classified
	}
	var gapErr *corehistory.SyncGapError
	if errors.As(err, &gapErr) {
		classified.Code = apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE
		classified.SyncLost = &apipb.OutputSyncLostErrorDetail{
			Consumer: "history", GapAfterLine: uint64(gapErr.GapAfterLine),
		}
		return classified
	}
	switch {
	case errors.Is(err, corev2.ErrApplicationForbidden):
		classified.Code = apipb.ApiErrorCode_API_ERROR_CODE_FORBIDDEN
	case errors.Is(err, corev2.ErrApplicationUnsupportedCapability):
		classified.Code = apipb.ApiErrorCode_API_ERROR_CODE_UNSUPPORTED_CAPABILITY
	case errors.Is(err, corev2.ErrProtocolResourceExhausted):
		classified.Code = apipb.ApiErrorCode_API_ERROR_CODE_RESOURCE_EXHAUSTED
		classified.Retryable = true
	case errors.Is(err, corehistory.ErrHistoryCopyTooLarge), errors.Is(err, corehistory.ErrHistoryWindowTooLarge):
		classified.Code = apipb.ApiErrorCode_API_ERROR_CODE_RESOURCE_EXHAUSTED
	case errors.Is(err, corehistory.ErrHistoryStaleWindow):
		classified.Code = apipb.ApiErrorCode_API_ERROR_CODE_STALE_RESOURCE
	case errors.Is(err, corev2.ErrApplicationCancellationUnavailable), errors.Is(err, corev2.ErrServerClosed), errors.Is(err, corev2.ErrHistoryNotRebuilt), errors.Is(err, corev2.ErrTerminalOutputUnavailable):
		classified.Code = apipb.ApiErrorCode_API_ERROR_CODE_UNAVAILABLE
		classified.Retryable = true
	case errors.Is(err, corehistory.ErrHistoryWindowLimit), errors.Is(err, corehistory.ErrHistoryInvalidMutation), errors.Is(err, corehistory.ErrHistoryInvalidSearchPattern), errors.Is(err, corev2.ErrInvalidTerminalID), errors.Is(err, corev2.ErrInvalidCommand), errors.Is(err, corev2.ErrInvalidServerSize), errors.Is(err, corev2.ErrInvalidFileUploadResume):
		classified.Code = apipb.ApiErrorCode_API_ERROR_CODE_INVALID_REQUEST
	case errors.Is(err, corev2.ErrTerminalNotFound):
		classified.Code = apipb.ApiErrorCode_API_ERROR_CODE_NOT_FOUND
	case errors.Is(err, corev2.ErrDuplicateTerminal), errors.Is(err, corev2.ErrTerminalExited):
		classified.Code = apipb.ApiErrorCode_API_ERROR_CODE_CONFLICT
	}
	return classified
}

func requireSession(contextMessage *apipb.RequestContext) error {
	return ValidateSessionStamp(contextMessage.GetSession())
}

func validateTerminalRefForContext(ref *apipb.TerminalRef, contextMessage *apipb.RequestContext) error {
	if err := requireSession(contextMessage); err != nil {
		return err
	}
	if ref == nil || ref.GetEndpointId() == "" || ref.GetTerminalId() == "" {
		return validation("terminal", "endpoint_id and terminal_id are required")
	}
	if len(ref.GetEndpointId()) > maxAPIIdentityBytes || len(ref.GetTerminalId()) > maxAPIIdentityBytes {
		return validation("terminal", "endpoint_id or terminal_id exceeds 256 bytes")
	}
	if ref.GetEndpointId() != contextMessage.GetSession().GetEndpointId() {
		return validation("terminal.endpoint_id", "must match context.session.endpoint_id")
	}
	return nil
}

func validateAttachmentOperation(resource *apipb.ResourceHandle, operation *apipb.OperationStamp, contextMessage *apipb.RequestContext) error {
	if err := ValidateResourceHandle(resource); err != nil {
		return err
	}
	if resource.GetKind() != apipb.ResourceKind_RESOURCE_KIND_TERMINAL_ATTACHMENT {
		return validation("attachment.kind", "must be terminal_attachment")
	}
	if !SessionStampsEqual(resource.GetSession(), contextMessage.GetSession()) {
		return validation("attachment.session", "must match context.session")
	}
	return ValidateOperationStamp(operation, contextMessage.GetSession())
}

func validAttachmentMode(mode apipb.AttachmentMode) bool {
	return mode == apipb.AttachmentMode_ATTACHMENT_MODE_COLLABORATOR || mode == apipb.AttachmentMode_ATTACHMENT_MODE_OBSERVER
}

func validResizePolicy(policy apipb.ResizePolicy) bool {
	switch policy {
	case apipb.ResizePolicy_RESIZE_POLICY_OWNER,
		apipb.ResizePolicy_RESIZE_POLICY_FOLLOWER,
		apipb.ResizePolicy_RESIZE_POLICY_OBSERVER:
		return true
	default:
		return false
	}
}

func applicationCapabilityToCore(capability apipb.ApiCapability) corev2.ApplicationCapability {
	switch capability {
	case apipb.ApiCapability_API_CAPABILITY_RESOURCE_LIFECYCLE:
		return corev2.ApplicationCapabilityResourceLifecycle
	case apipb.ApiCapability_API_CAPABILITY_TERMINAL_LIFECYCLE:
		return corev2.ApplicationCapabilityTerminalLifecycle
	case apipb.ApiCapability_API_CAPABILITY_TERMINAL_ATTACHMENT:
		return corev2.ApplicationCapabilityTerminalAttachment
	case apipb.ApiCapability_API_CAPABILITY_PATH_QUERY:
		return corev2.ApplicationCapabilityPathQuery
	case apipb.ApiCapability_API_CAPABILITY_HISTORY:
		return corev2.ApplicationCapabilityHistory
	case apipb.ApiCapability_API_CAPABILITY_LIVE_SCREEN:
		return corev2.ApplicationCapabilityLiveScreen
	case apipb.ApiCapability_API_CAPABILITY_FILE:
		return corev2.ApplicationCapabilityFile
	case apipb.ApiCapability_API_CAPABILITY_STORAGE:
		return corev2.ApplicationCapabilityStorage
	case apipb.ApiCapability_API_CAPABILITY_EVENT_SUBSCRIPTION:
		return corev2.ApplicationCapabilityEventSubscription
	case apipb.ApiCapability_API_CAPABILITY_CLIENT_ACCESS:
		return corev2.ApplicationCapabilityClientAccess
	case apipb.ApiCapability_API_CAPABILITY_REMOTE_CONTROL:
		return corev2.ApplicationCapabilityRemoteControl
	case apipb.ApiCapability_API_CAPABILITY_BROWSER_PROXY:
		return corev2.ApplicationCapabilityBrowserProxy
	default:
		return 0
	}
}

func terminalStateToProto(state corev2.TerminalState) (apipb.TerminalState, error) {
	switch state {
	case corev2.TerminalStateCreated:
		return apipb.TerminalState_TERMINAL_STATE_CREATED, nil
	case corev2.TerminalStateRunning:
		return apipb.TerminalState_TERMINAL_STATE_RUNNING, nil
	case corev2.TerminalStateExited:
		return apipb.TerminalState_TERMINAL_STATE_EXITED, nil
	case corev2.TerminalStateRemoved:
		return apipb.TerminalState_TERMINAL_STATE_REMOVED, nil
	default:
		return apipb.TerminalState_TERMINAL_STATE_UNSPECIFIED, fmt.Errorf("unsupported core terminal state %q", state)
	}
}

func attachmentModeToCore(mode apipb.AttachmentMode) corev2.TerminalAttachmentMode {
	if mode == apipb.AttachmentMode_ATTACHMENT_MODE_OBSERVER {
		return corev2.TerminalAttachmentModeObserver
	}
	return corev2.TerminalAttachmentModeCollaborator
}

func resizePolicyToCore(policy apipb.ResizePolicy) corev2.TerminalResizePolicy {
	switch policy {
	case apipb.ResizePolicy_RESIZE_POLICY_FOLLOWER:
		return corev2.TerminalResizePolicyFollower
	case apipb.ResizePolicy_RESIZE_POLICY_OBSERVER:
		return corev2.TerminalResizePolicyObserver
	default:
		return corev2.TerminalResizePolicyOwner
	}
}

func resizeControlToProto(control *corev2.TerminalResizeControl) *apipb.ResizeControl {
	if control == nil {
		return nil
	}
	out := &apipb.ResizeControl{
		CanResize: control.CanResize, Reason: resizeReasonToProto(control.Reason), SizeLocked: control.SizeLocked,
		SurfaceId: control.SurfaceID, OwnerSurfaceId: control.OwnerSurfaceID, OwnerViewId: control.OwnerViewID,
	}
	if ownership := control.ResizeOwnership; ownership != nil {
		out.Ownership = &apipb.ResizeOwnership{
			OwnerAttachmentId: ownership.OwnerAttachmentID, OwnerSurfaceId: ownership.OwnerSurfaceID,
			OwnerViewId: ownership.OwnerViewID, Size: TerminalSizeToProto(ownership.Size), SizeLocked: ownership.SizeLocked, Epoch: ownership.Epoch,
		}
	}
	return out
}

func resizeReasonToProto(reason corev2.TerminalResizeReason) apipb.ResizeControlReason {
	switch reason {
	case corev2.TerminalResizeReasonOwner:
		return apipb.ResizeControlReason_RESIZE_CONTROL_REASON_OWNER
	case corev2.TerminalResizeReasonObserver:
		return apipb.ResizeControlReason_RESIZE_CONTROL_REASON_OBSERVER
	case corev2.TerminalResizeReasonSizeLocked:
		return apipb.ResizeControlReason_RESIZE_CONTROL_REASON_SIZE_LOCKED
	default:
		return apipb.ResizeControlReason_RESIZE_CONTROL_REASON_FOLLOWER
	}
}

func cloneTerminalRef(value *apipb.TerminalRef) *apipb.TerminalRef {
	if value == nil {
		return nil
	}
	return &apipb.TerminalRef{EndpointId: value.GetEndpointId(), TerminalId: value.GetTerminalId()}
}

func cloneOperationStamp(value *apipb.OperationStamp) *apipb.OperationStamp {
	if value == nil {
		return nil
	}
	return &apipb.OperationStamp{Session: cloneSessionStamp(value.GetSession()), OperationId: value.GetOperationId()}
}

func cloneSessionStamp(value *apipb.EndpointSessionStamp) *apipb.EndpointSessionStamp {
	if value == nil {
		return nil
	}
	return &apipb.EndpointSessionStamp{EndpointId: value.GetEndpointId(), RouteId: value.GetRouteId(), Generation: value.GetGeneration()}
}

func cloneBytes(value []byte) []byte { return append([]byte(nil), value...) }

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func unixNanoOrZero(value time.Time) int64 {
	if value.IsZero() {
		return 0
	}
	return value.UnixNano()
}

func validateStringSlice(field string, values []string, maxItems int) error {
	if len(values) > maxItems {
		return validation(field, fmt.Sprintf("exceeds %d items", maxItems))
	}
	total := 0
	for index, value := range values {
		total += len(value)
		if total > maxAggregateTextBytes {
			return validation(field, "exceeds 1 MiB total text")
		}
		if len(value) > maxMetadataTextBytes {
			return validation(fmt.Sprintf("%s[%d]", field, index), "exceeds 4096 bytes")
		}
	}
	return nil
}

func validateStringMap(field string, values map[string]string) error {
	if len(values) > maxTagEntries {
		return validation(field, "exceeds 256 entries")
	}
	total := 0
	for key, value := range values {
		if len(key) == 0 || len(key) > 256 {
			return validation(field, "contains an empty or oversized key")
		}
		if len(value) > maxMetadataTextBytes {
			return validation(field, "contains a value exceeding 4096 bytes")
		}
		total += len(key) + len(value)
		if total > maxAggregateTextBytes {
			return validation(field, "exceeds 1 MiB total text")
		}
	}
	return nil
}
