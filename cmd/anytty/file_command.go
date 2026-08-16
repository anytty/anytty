package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	pathpkg "path"
	"path/filepath"
	"strings"
	"time"

	protocoladapter "github.com/anytty/anytty/client/adapter/protocol"
	endpointdomain "github.com/anytty/anytty/client/endpoint"
	clientruntime "github.com/anytty/anytty/client/runtime"
	"github.com/anytty/anytty/internal/protocol"
	"github.com/anytty/anytty/proto/apipb"
	"github.com/anytty/anytty/proto/wire"
	"github.com/anytty/anytty/shared/filepublish"
	"github.com/anytty/anytty/shared/securefs"
	"github.com/spf13/cobra"
)

type fileCommandRuntime struct {
	socket  *string
	logFile *string
	timeout time.Duration
}

const maxCLIFileChunkBytes = 1 << 20

type fileEntryView struct {
	Path       string `json:"path"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Size       int64  `json:"size"`
	Mode       uint32 `json:"mode"`
	ModifiedAt string `json:"modified_at,omitempty"`
	LinkTarget string `json:"link_target,omitempty"`
}

type fileOperationView struct {
	Path       string `json:"path"`
	TargetPath string `json:"target_path,omitempty"`
	Success    bool   `json:"success"`
	ErrorCode  string `json:"error_code,omitempty"`
	Error      string `json:"error,omitempty"`
}

type fileTransferView struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          string `json:"kind"`
	EndpointID    string `json:"endpoint_id"`
	RemotePath    string `json:"remote_path"`
	LocalPath     string `json:"local_path,omitempty"`
	Size          int64  `json:"size"`
	SHA256        string `json:"sha256"`
}

func newFileCommand(socket, logFile *string) *cobra.Command {
	runtime := &fileCommandRuntime{socket: socket, logFile: logFile}
	command := &cobra.Command{Use: "file", Short: "Operate on an endpoint daemon file system"}
	command.PersistentFlags().DurationVar(&runtime.timeout, "timeout", 2*time.Minute, "operation timeout")
	command.AddCommand(
		newFileListCommand(runtime), newFileStatCommand(runtime), newFileCatCommand(runtime),
		newFileDownloadCommand(runtime), newFileUploadCommand(runtime), newFileMkdirCommand(runtime),
		newFileRenameCommand(runtime), newFileCopyMoveCommand(runtime, false), newFileCopyMoveCommand(runtime, true),
		newFileRemoveCommand(runtime),
	)
	return command
}

func (runtime *fileCommandRuntime) open(ctx context.Context, cmd *cobra.Command, endpointValue string) (*protocoladapter.ApplicationClient, endpointdomain.Endpoint, func(), error) {
	if runtime.timeout <= 0 {
		return nil, endpointdomain.Endpoint{}, func() {}, usageCLIError("--timeout must be positive")
	}
	registry, err := loadNormalizedConnectionRegistry()
	if err != nil {
		return nil, endpointdomain.Endpoint{}, func() {}, err
	}
	endpoint, err := resolveEndpointConfig(endpointValue, registry)
	if err != nil {
		return nil, endpointdomain.Endpoint{}, func() {}, err
	}
	cmd.Root().SilenceUsage = true
	var client *protocoladapter.ApplicationClient
	var closeClient func()
	client, closeClient, err = openEndpointProtocolClient(ctx, endpoint, *runtime.socket, *runtime.logFile)
	if err != nil {
		return nil, endpointdomain.Endpoint{}, func() {}, classifyCLIError(err)
	}
	return client, endpoint, closeClient, nil
}

func (runtime *fileCommandRuntime) context(cmd *cobra.Command) (context.Context, context.CancelFunc) {
	return context.WithTimeout(cmd.Context(), runtime.timeout)
}

func newFileListCommand(runtime *fileCommandRuntime) *cobra.Command {
	var cursor string
	var limit int
	var all, jsonOutput bool
	command := &cobra.Command{
		Use: "list ENDPOINT [PATH]", Short: "List a daemon-owned directory", Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 || limit > 10000 {
				return usageCLIError("--limit must be 1..10000")
			}
			ctx, cancel := runtime.context(cmd)
			defer cancel()
			client, endpoint, closeClient, err := runtime.open(ctx, cmd, args[0])
			if err != nil {
				return err
			}
			defer closeClient()
			path := ""
			if len(args) == 2 {
				path = args[1]
			} else {
				defaults, err := client.TerminalDefaults(ctx, &apipb.TerminalDefaultsCommand{})
				if err != nil {
					return classifyCLIError(err)
				}
				path = defaults.GetDefaults().GetDefaultCwd()
			}
			entries := make([]fileEntryView, 0)
			next := cursor
			seen := make(map[string]struct{})
			for {
				result, err := client.ApplicationSession.FileList(ctx, &apipb.FileListCommand{Path: path, Cursor: next, Limit: int32(limit)})
				if err != nil {
					return classifyCLIError(err)
				}
				for _, entry := range result.GetEntries() {
					entries = append(entries, fileEntryProjection(entry))
				}
				next = result.GetNextCursor()
				if !all || next == "" {
					break
				}
				if _, duplicate := seen[next]; duplicate {
					return &cliError{code: 6, message: "daemon returned a repeated file cursor"}
				}
				seen[next] = struct{}{}
			}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(struct {
					SchemaVersion int             `json:"schema_version"`
					Kind          string          `json:"kind"`
					EndpointID    string          `json:"endpoint_id"`
					Path          string          `json:"path"`
					Items         []fileEntryView `json:"items"`
					NextCursor    string          `json:"next_cursor,omitempty"`
				}{1, "file_list", string(endpoint.ID), path, entries, next})
			}
			rows := make([][]string, 0, len(entries))
			for _, entry := range entries {
				rows = append(rows, []string{entry.Type, fmt.Sprintf("%d", entry.Size), entry.ModifiedAt, entry.Name})
			}
			return writeCLITable(cmd.OutOrStdout(), []string{"TYPE", "SIZE", "MODIFIED", "NAME"}, rows)
		},
	}
	command.Flags().StringVar(&cursor, "cursor", "", "daemon-issued pagination cursor")
	command.Flags().IntVar(&limit, "limit", 200, "entries per page")
	command.Flags().BoolVar(&all, "all", false, "read all pages")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func newFileStatCommand(runtime *fileCommandRuntime) *cobra.Command {
	var jsonOutput bool
	command := &cobra.Command{
		Use: "stat ENDPOINT PATH", Short: "Show daemon-owned file metadata", Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := runtime.context(cmd)
			defer cancel()
			client, endpoint, closeClient, err := runtime.open(ctx, cmd, args[0])
			if err != nil {
				return err
			}
			defer closeClient()
			result, err := client.ApplicationSession.FileStat(ctx, &apipb.FileStatCommand{Path: args[1]})
			if err != nil {
				return classifyCLIError(err)
			}
			view := fileEntryProjection(result.GetEntry())
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(struct {
					SchemaVersion int           `json:"schema_version"`
					Kind          string        `json:"kind"`
					EndpointID    string        `json:"endpoint_id"`
					Item          fileEntryView `json:"item"`
				}{1, "file_stat", string(endpoint.ID), view})
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Path", Value: view.Path},
				cliField{Label: "Type", Value: view.Type},
				cliField{Label: "Size", Value: fmt.Sprintf("%d bytes", view.Size)},
				cliField{Label: "Mode", Value: fmt.Sprintf("%04o", view.Mode)},
				cliField{Label: "Modified", Value: view.ModifiedAt},
			)
		},
	}
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func newFileCatCommand(runtime *fileCommandRuntime) *cobra.Command {
	return &cobra.Command{
		Use: "cat ENDPOINT PATH", Short: "Stream a daemon-owned file to stdout", Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := runtime.context(cmd)
			defer cancel()
			client, _, closeClient, err := runtime.open(ctx, cmd, args[0])
			if err != nil {
				return err
			}
			defer closeClient()
			_, err = downloadEndpointFile(ctx, client, args[1], cmd.OutOrStdout())
			return classifyCLIError(err)
		},
	}
}

func newFileDownloadCommand(runtime *fileCommandRuntime) *cobra.Command {
	var overwrite, jsonOutput bool
	command := &cobra.Command{
		Use: "download ENDPOINT REMOTE [LOCAL]", Short: "Download and verify a daemon-owned file", Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := runtime.context(cmd)
			defer cancel()
			client, endpoint, closeClient, err := runtime.open(ctx, cmd, args[0])
			if err != nil {
				return err
			}
			defer closeClient()
			localPath := filepath.Base(args[1])
			if len(args) == 3 {
				localPath = args[2]
			}
			result, err := downloadEndpointFileAtomic(ctx, client, args[1], localPath, overwrite)
			if err != nil {
				if errors.Is(err, os.ErrExist) {
					return &cliError{code: 4, message: fmt.Sprintf("local file %s already exists; use --overwrite", localPath), cause: err}
				}
				return classifyCLIError(err)
			}
			view := fileTransferView{1, "file_download", string(endpoint.ID), args[1], localPath, result.Size, hex.EncodeToString(result.SHA256)}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(view)
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Local path", Value: localPath},
				cliField{Label: "Status", Value: "downloaded"},
				cliField{Label: "Size", Value: fmt.Sprintf("%d bytes", view.Size)},
				cliField{Label: "SHA-256", Value: view.SHA256},
			)
		},
	}
	command.Flags().BoolVar(&overwrite, "overwrite", false, "replace an existing local file")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func newFileUploadCommand(runtime *fileCommandRuntime) *cobra.Command {
	var overwrite, jsonOutput bool
	command := &cobra.Command{
		Use: "upload ENDPOINT LOCAL [REMOTE]", Short: "Upload a local file and wait for daemon checksum confirmation", Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := runtime.context(cmd)
			defer cancel()
			client, endpoint, closeClient, err := runtime.open(ctx, cmd, args[0])
			if err != nil {
				return err
			}
			defer closeClient()
			remotePath := ""
			if len(args) == 3 {
				remotePath = args[2]
			} else {
				defaults, err := client.TerminalDefaults(ctx, &apipb.TerminalDefaultsCommand{})
				if err != nil {
					return classifyCLIError(err)
				}
				remotePath = pathpkg.Join(defaults.GetDefaults().GetDefaultCwd(), filepath.Base(args[1]))
			}
			result, err := uploadEndpointFile(ctx, client, args[1], remotePath, overwrite)
			if err != nil {
				return classifyCLIError(err)
			}
			view := fileTransferView{1, "file_upload", string(endpoint.ID), result.Path, args[1], result.Size, hex.EncodeToString(result.SHA256)}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(view)
			}
			return writeCLIFields(cmd.OutOrStdout(),
				cliField{Label: "Remote path", Value: result.Path},
				cliField{Label: "Status", Value: "uploaded"},
				cliField{Label: "Size", Value: fmt.Sprintf("%d bytes", result.Size)},
				cliField{Label: "SHA-256", Value: view.SHA256},
			)
		},
	}
	command.Flags().BoolVar(&overwrite, "overwrite", false, "replace an existing remote file")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func newFileMkdirCommand(runtime *fileCommandRuntime) *cobra.Command {
	var parents, jsonOutput bool
	command := &cobra.Command{
		Use: "mkdir ENDPOINT PATH", Short: "Create a daemon-owned directory", Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFileSingleMutation(cmd, runtime, args[0], jsonOutput, func(ctx context.Context, client *protocoladapter.ApplicationClient) (*apipb.FileOperationResult, error) {
				return client.ApplicationSession.FileMkdir(ctx, &apipb.FileMkdirCommand{Path: args[1], Recursive: parents})
			})
		},
	}
	command.Flags().BoolVarP(&parents, "parents", "p", false, "create parent directories")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func newFileRenameCommand(runtime *fileCommandRuntime) *cobra.Command {
	var overwrite, jsonOutput bool
	command := &cobra.Command{
		Use: "rename ENDPOINT OLD NEW", Short: "Rename a daemon-owned path", Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFileSingleMutation(cmd, runtime, args[0], jsonOutput, func(ctx context.Context, client *protocoladapter.ApplicationClient) (*apipb.FileOperationResult, error) {
				return client.ApplicationSession.FileRename(ctx, &apipb.FileRenameCommand{Path: args[1], NewPath: args[2], Overwrite: overwrite})
			})
		},
	}
	command.Flags().BoolVar(&overwrite, "overwrite", false, "replace an existing target")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func newFileCopyMoveCommand(runtime *fileCommandRuntime, move bool) *cobra.Command {
	var overwrite, jsonOutput bool
	verb := "copy"
	if move {
		verb = "move"
	}
	command := &cobra.Command{
		Use: verb + " ENDPOINT SRC... DEST_DIR", Short: strings.ToUpper(verb[:1]) + verb[1:] + " daemon-owned paths", Args: cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := runtime.context(cmd)
			defer cancel()
			client, endpoint, closeClient, err := runtime.open(ctx, cmd, args[0])
			if err != nil {
				return err
			}
			defer closeClient()
			paths := append([]string(nil), args[1:len(args)-1]...)
			var result *apipb.FileBatchResult
			if move {
				result, err = client.ApplicationSession.FileMove(ctx, &apipb.FileMoveCommand{Paths: paths, TargetDirectory: args[len(args)-1], Overwrite: overwrite})
			} else {
				result, err = client.ApplicationSession.FileCopy(ctx, &apipb.FileCopyCommand{Paths: paths, TargetDirectory: args[len(args)-1], Overwrite: overwrite})
			}
			if err != nil {
				return classifyCLIError(err)
			}
			return writeFileOperationResults(cmd, endpoint.ID, "file_"+verb, result.GetResults(), jsonOutput)
		},
	}
	command.Flags().BoolVar(&overwrite, "overwrite", false, "replace existing targets")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func newFileRemoveCommand(runtime *fileCommandRuntime) *cobra.Command {
	var recursive, jsonOutput bool
	command := &cobra.Command{
		Use: "remove ENDPOINT PATH...", Short: "Remove daemon-owned paths", Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := runtime.context(cmd)
			defer cancel()
			client, endpoint, closeClient, err := runtime.open(ctx, cmd, args[0])
			if err != nil {
				return err
			}
			defer closeClient()
			results := make([]*apipb.FileOperationResult, 0, len(args)-1)
			for _, path := range args[1:] {
				result, requestErr := client.ApplicationSession.FileDelete(ctx, &apipb.FileDeleteCommand{Path: path, Recursive: recursive})
				if requestErr != nil {
					return classifyCLIError(requestErr)
				}
				results = append(results, result)
			}
			return writeFileOperationResults(cmd, endpoint.ID, "file_remove", results, jsonOutput)
		},
	}
	command.Flags().BoolVarP(&recursive, "recursive", "r", false, "remove directories recursively")
	command.Flags().BoolVar(&jsonOutput, "json", false, "print machine-readable JSON")
	return command
}

func runFileSingleMutation(cmd *cobra.Command, runtime *fileCommandRuntime, endpointValue string, jsonOutput bool, operation func(context.Context, *protocoladapter.ApplicationClient) (*apipb.FileOperationResult, error)) error {
	ctx, cancel := runtime.context(cmd)
	defer cancel()
	client, endpoint, closeClient, err := runtime.open(ctx, cmd, endpointValue)
	if err != nil {
		return err
	}
	defer closeClient()
	result, err := operation(ctx, client)
	if err != nil {
		return classifyCLIError(err)
	}
	return writeFileOperationResults(cmd, endpoint.ID, "file_operation", []*apipb.FileOperationResult{result}, jsonOutput)
}

func writeFileOperationResults(cmd *cobra.Command, endpointID endpointdomain.EndpointID, kind string, results []*apipb.FileOperationResult, jsonOutput bool) error {
	views := make([]fileOperationView, 0, len(results))
	failures := 0
	for _, result := range results {
		views = append(views, fileOperationView{result.GetPath(), result.GetTargetPath(), result.GetSuccess(), result.GetErrorCode(), result.GetErrorMessage()})
		if !result.GetSuccess() {
			failures++
		}
	}
	if jsonOutput {
		if err := json.NewEncoder(cmd.OutOrStdout()).Encode(struct {
			SchemaVersion int                 `json:"schema_version"`
			Kind          string              `json:"kind"`
			EndpointID    string              `json:"endpoint_id"`
			Results       []fileOperationView `json:"results"`
		}{1, kind, string(endpointID), views}); err != nil {
			return err
		}
	} else {
		rows := make([][]string, 0, len(views))
		for _, view := range views {
			status := "ok"
			if !view.Success {
				status = view.ErrorCode + ": " + view.Error
			}
			rows = append(rows, []string{view.Path, view.TargetPath, status})
		}
		if err := writeCLITable(cmd.OutOrStdout(), []string{"PATH", "TARGET", "STATUS"}, rows); err != nil {
			return err
		}
	}
	if failures > 0 {
		code := 8
		if len(results) == 1 {
			code = 4
		}
		return &cliError{code: code, message: fmt.Sprintf("%d of %d file operations failed", failures, len(results))}
	}
	return nil
}

func downloadEndpointFile(ctx context.Context, client *protocoladapter.ApplicationClient, remotePath string, writer io.Writer) (protocol.FileTransferResult, error) {
	result, err := client.ApplicationSession.FileDownloadOpen(ctx, &apipb.FileDownloadOpenCommand{Path: remotePath})
	if err != nil {
		return protocol.FileTransferResult{}, err
	}
	opened := result.GetTransfer()
	if err := validateFileTransferOpen(opened, -1, false); err != nil {
		cancelFileTransfer(client, opened.GetResource())
		return protocol.FileTransferResult{}, err
	}
	completed := false
	defer func() {
		if !completed {
			cancelFileTransfer(client, opened.GetResource())
		}
	}()
	stream, err := client.OpenResourceStream(opened.GetResource())
	if err != nil {
		return protocol.FileTransferResult{}, err
	}
	defer stream.Close()
	hash := sha256.New()
	offset := opened.GetOffset()
	for {
		typ, payload, err := stream.Receive(ctx)
		if err != nil {
			return protocol.FileTransferResult{}, err
		}
		switch typ {
		case wire.TypeFileData:
			data, err := protocol.DecodeFileTransferData(payload)
			if err != nil || data.Offset != offset {
				return protocol.FileTransferResult{}, fmt.Errorf("invalid download data at offset %d", offset)
			}
			if _, err := writer.Write(data.Data); err != nil {
				return protocol.FileTransferResult{}, err
			}
			_, _ = hash.Write(data.Data)
			offset += int64(len(data.Data))
			ack, err := protocol.EncodeFileTransferAck(protocol.FileTransferAck{Offset: offset, WindowBytes: int64(len(data.Data))})
			if err != nil {
				return protocol.FileTransferResult{}, err
			}
			if err := stream.Send(ctx, wire.TypeFileAck, ack); err != nil {
				return protocol.FileTransferResult{}, err
			}
		case wire.TypeFileFinish:
			finish, err := protocol.DecodeFileTransferFinish(payload)
			if err != nil {
				return protocol.FileTransferResult{}, err
			}
			if finish.Size != offset || finish.Size != opened.GetSize() || !bytes.Equal(finish.SHA256, hash.Sum(nil)) {
				return protocol.FileTransferResult{}, fmt.Errorf("download size or SHA-256 mismatch")
			}
			completed = true
			return protocol.FileTransferResult{Path: opened.GetPath(), Size: finish.Size, SHA256: finish.SHA256}, nil
		case wire.TypeError:
			message, decodeErr := protocol.DecodeErrorPayload(payload)
			if decodeErr != nil {
				return protocol.FileTransferResult{}, decodeErr
			}
			return protocol.FileTransferResult{}, &protocol.RequestError{Code: message.Error.Code, Message: message.Error.Message}
		default:
			return protocol.FileTransferResult{}, fmt.Errorf("unexpected download frame type %d", typ)
		}
	}
}

func downloadEndpointFileAtomic(ctx context.Context, client *protocoladapter.ApplicationClient, remotePath, localPath string, overwrite bool) (protocol.FileTransferResult, error) {
	localPath = filepath.Clean(strings.TrimSpace(localPath))
	if localPath == "." || localPath == "" {
		return protocol.FileTransferResult{}, usageCLIError("local download path is required")
	}
	if !overwrite {
		if _, err := os.Lstat(localPath); err == nil {
			return protocol.FileTransferResult{}, os.ErrExist
		} else if !os.IsNotExist(err) {
			return protocol.FileTransferResult{}, err
		}
	}
	parent := filepath.Dir(localPath)
	temporary, err := os.CreateTemp(parent, ".anytty-download-*")
	if err != nil {
		return protocol.FileTransferResult{}, err
	}
	temporaryPath := temporary.Name()
	committed := false
	defer func() {
		_ = temporary.Close()
		if !committed {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := securefs.SecureFile(temporaryPath); err != nil {
		return protocol.FileTransferResult{}, err
	}
	result, err := downloadEndpointFile(ctx, client, remotePath, temporary)
	if err != nil {
		return protocol.FileTransferResult{}, err
	}
	if err := temporary.Sync(); err != nil {
		return protocol.FileTransferResult{}, err
	}
	if err := temporary.Close(); err != nil {
		return protocol.FileTransferResult{}, err
	}
	if overwrite {
		err = filepublish.Rename(temporaryPath, localPath)
	} else {
		err = os.Link(temporaryPath, localPath)
		if err == nil {
			_ = os.Remove(temporaryPath)
		}
	}
	if err != nil {
		return protocol.FileTransferResult{}, err
	}
	if err := filepublish.SyncDirectory(parent); err != nil {
		return protocol.FileTransferResult{}, err
	}
	committed = true
	return result, nil
}

func uploadEndpointFile(ctx context.Context, client *protocoladapter.ApplicationClient, localPath, remotePath string, overwrite bool) (protocol.FileTransferResult, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return protocol.FileTransferResult{}, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return protocol.FileTransferResult{}, err
	}
	if !info.Mode().IsRegular() {
		return protocol.FileTransferResult{}, fmt.Errorf("upload source must be a regular file")
	}
	result, err := client.ApplicationSession.FileUploadOpen(ctx, &apipb.FileUploadOpenCommand{Path: remotePath, Size: info.Size(), Overwrite: overwrite})
	if err != nil {
		return protocol.FileTransferResult{}, err
	}
	opened := result.GetTransfer()
	if err := validateFileTransferOpen(opened, info.Size(), false); err != nil {
		cancelFileTransfer(client, opened.GetResource())
		return protocol.FileTransferResult{}, err
	}
	completed := false
	defer func() {
		if !completed {
			cancelFileTransfer(client, opened.GetResource())
		}
	}()
	stream, err := client.OpenResourceStream(opened.GetResource())
	if err != nil {
		return protocol.FileTransferResult{}, err
	}
	defer stream.Close()
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return protocol.FileTransferResult{}, err
	}
	hash := sha256.New()
	if opened.GetOffset() > 0 {
		if _, err := io.CopyN(hash, file, opened.GetOffset()); err != nil {
			return protocol.FileTransferResult{}, err
		}
	}
	chunkBytes := opened.GetChunkBytes()
	if chunkBytes == 0 || chunkBytes > maxCLIFileChunkBytes {
		return protocol.FileTransferResult{}, fmt.Errorf("invalid upload chunk size %d", chunkBytes)
	}
	buffer := make([]byte, int(chunkBytes))
	offset := opened.GetOffset()
	for offset < opened.GetSize() {
		remaining := opened.GetSize() - offset
		chunk := buffer
		if int64(len(chunk)) > remaining {
			chunk = chunk[:remaining]
		}
		n, readErr := io.ReadFull(file, chunk)
		if readErr != nil && !errors.Is(readErr, io.ErrUnexpectedEOF) {
			return protocol.FileTransferResult{}, readErr
		}
		chunk = chunk[:n]
		_, _ = hash.Write(chunk)
		payload, err := protocol.EncodeFileTransferData(protocol.FileTransferData{Offset: offset, Data: chunk})
		if err != nil {
			return protocol.FileTransferResult{}, err
		}
		if err := stream.Send(ctx, wire.TypeFileData, payload); err != nil {
			return protocol.FileTransferResult{}, err
		}
		frame, err := nextFileFrame(ctx, stream)
		if err != nil {
			return protocol.FileTransferResult{}, err
		}
		if frame.Type == wire.TypeError {
			message, decodeErr := protocol.DecodeErrorPayload(frame.Payload)
			if decodeErr != nil {
				return protocol.FileTransferResult{}, decodeErr
			}
			return protocol.FileTransferResult{}, &protocol.RequestError{Code: message.Error.Code, Message: message.Error.Message}
		}
		if frame.Type != wire.TypeFileAck {
			return protocol.FileTransferResult{}, fmt.Errorf("unexpected upload frame type %d", frame.Type)
		}
		ack, err := protocol.DecodeFileTransferAck(frame.Payload)
		if err != nil || ack.Offset != offset+int64(n) {
			return protocol.FileTransferResult{}, fmt.Errorf("invalid upload acknowledgement")
		}
		offset = ack.Offset
	}
	digest := hash.Sum(nil)
	finish, err := protocol.EncodeFileTransferFinish(protocol.FileTransferFinish{Size: opened.GetSize(), SHA256: digest})
	if err != nil {
		return protocol.FileTransferResult{}, err
	}
	if err := stream.Send(ctx, wire.TypeFileFinish, finish); err != nil {
		return protocol.FileTransferResult{}, err
	}
	frame, err := nextFileFrame(ctx, stream)
	if err != nil {
		return protocol.FileTransferResult{}, err
	}
	if frame.Type == wire.TypeError {
		message, decodeErr := protocol.DecodeErrorPayload(frame.Payload)
		if decodeErr != nil {
			return protocol.FileTransferResult{}, decodeErr
		}
		return protocol.FileTransferResult{}, &protocol.RequestError{Code: message.Error.Code, Message: message.Error.Message}
	}
	if frame.Type != wire.TypeFileResult {
		return protocol.FileTransferResult{}, fmt.Errorf("unexpected upload completion frame %d", frame.Type)
	}
	completedResult, err := protocol.DecodeFileTransferResult(frame.Payload)
	if err != nil {
		return protocol.FileTransferResult{}, err
	}
	if completedResult.Size != opened.GetSize() || !bytes.Equal(completedResult.SHA256, digest) {
		return protocol.FileTransferResult{}, fmt.Errorf("upload completion size or SHA-256 mismatch")
	}
	completed = true
	return completedResult, nil
}

func validateFileTransferOpen(opened *apipb.FileTransferHandle, expectedSize int64, allowResume bool) error {
	if opened == nil || opened.GetResource() == nil || strings.TrimSpace(opened.GetPath()) == "" {
		return fmt.Errorf("daemon returned incomplete file transfer metadata")
	}
	if opened.GetSize() < 0 || (expectedSize >= 0 && opened.GetSize() != expectedSize) {
		return fmt.Errorf("daemon returned an invalid file transfer size")
	}
	if opened.GetOffset() < 0 || opened.GetOffset() > opened.GetSize() || (!allowResume && opened.GetOffset() != 0) {
		return fmt.Errorf("daemon returned an invalid file transfer offset")
	}
	return nil
}

func cancelFileTransfer(client *protocoladapter.ApplicationClient, resource *apipb.ResourceHandle) {
	if client == nil || resource == nil {
		return
	}
	cancelContext, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _ = client.ApplicationSession.FileTransferCancel(cancelContext, &apipb.FileTransferCancelCommand{Transfer: resource})
}

func nextFileFrame(ctx context.Context, stream clientruntime.ResourceStream) (protocol.StreamFrame, error) {
	typ, payload, err := stream.Receive(ctx)
	if err != nil {
		return protocol.StreamFrame{}, err
	}
	return protocol.StreamFrame{Type: typ, Payload: payload}, nil
}

func fileEntryProjection(entry *apipb.FileEntry) fileEntryView {
	if entry == nil {
		return fileEntryView{}
	}
	modified := time.Time{}
	if entry.GetModifiedAtUnixNano() != 0 {
		modified = time.Unix(0, entry.GetModifiedAtUnixNano()).UTC()
	}
	return fileEntryView{entry.GetPath(), entry.GetName(), fileEntryTypeName(entry.GetType()), entry.GetSize(), entry.GetMode(), formatTerminalTime(modified), entry.GetLinkTarget()}
}

func fileEntryTypeName(entryType apipb.FileEntryType) string {
	switch entryType {
	case apipb.FileEntryType_FILE_ENTRY_TYPE_FILE:
		return "file"
	case apipb.FileEntryType_FILE_ENTRY_TYPE_DIRECTORY:
		return "directory"
	case apipb.FileEntryType_FILE_ENTRY_TYPE_SYMLINK:
		return "symlink"
	default:
		return "other"
	}
}
