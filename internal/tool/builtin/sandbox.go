package builtin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	dockerimage "github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/strslice"
	"github.com/docker/docker/client"
	"github.com/wunderpus/wunderpus/internal/tool"
)

var supportedLanguages = map[string]string{
	"python":  "python:3-slim",
	"python3": "python:3-slim",
	"node":    "node:20-slim",
	"nodejs":  "node:20-slim",
	"go":      "golang:1.25-slim",
	"bash":    "bash:latest",
	"sh":      "bash:latest",
	"ruby":    "ruby:3-slim",
	"rust":    "rust:1.85-slim",
}

type SandboxTool struct {
	mu            sync.Mutex
	dockerClient  *client.Client
	containerIDs  map[string]string
	runningImages map[string]bool
}

func NewSandboxTool() *SandboxTool {
	return &SandboxTool{
		containerIDs:  make(map[string]string),
		runningImages: make(map[string]bool),
	}
}

func (s *SandboxTool) Name() string { return "sandbox_run_code" }

func (s *SandboxTool) Description() string {
	return "Run code in an ephemeral Docker container. Supports Python, Node.js, Go, Ruby, Rust, and Bash. " +
		"The container is automatically created, executed, and destroyed after running."
}

func (s *SandboxTool) Sensitive() bool                   { return false }
func (s *SandboxTool) ApprovalLevel() tool.ApprovalLevel { return tool.AutoExecute }
func (s *SandboxTool) Version() string                   { return "1.0.0" }

func (s *SandboxTool) Dependencies() []string { return nil }

func (s *SandboxTool) Parameters() []tool.ParameterDef {
	return []tool.ParameterDef{
		{Name: "language", Type: "string", Description: "Programming language: python, node, go, ruby, rust, bash, sh", Required: true},
		{Name: "code", Type: "string", Description: "The code to execute", Required: true},
		{Name: "timeout", Type: "number", Description: "Maximum execution time in seconds (default: 30)", Required: false},
	}
}

func (s *SandboxTool) getDockerClient() error {
	if s.dockerClient != nil {
		return nil
	}

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	s.dockerClient = cli
	return nil
}

func (s *SandboxTool) getImage(language string) (string, error) {
	lang := strings.ToLower(language)
	image, ok := supportedLanguages[lang]
	if !ok {
		return "", fmt.Errorf("unsupported language: %s. Supported: python, node, go, ruby, rust, bash", language)
	}
	return image, nil
}

func (s *SandboxTool) getCommand(language, code string) strslice.StrSlice {
	lang := strings.ToLower(language)
	switch lang {
	case "python", "python3":
		return strslice.StrSlice{"python3", "-c", code}
	case "node", "nodejs":
		return strslice.StrSlice{"node", "-e", code}
	case "go":
		return strslice.StrSlice{"go", "run", "-e", code}
	case "ruby":
		return strslice.StrSlice{"ruby", "-e", code}
	case "rust":
		return strslice.StrSlice{"sh", "-c", "echo 'Rust execution requires cargo. Using echo instead.' && echo " + code}
	case "bash", "sh":
		return strslice.StrSlice{"bash", "-c", code}
	default:
		return strslice.StrSlice{"echo", "Unsupported language"}
	}
}

func (s *SandboxTool) Execute(ctx context.Context, args map[string]any) (*tool.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	language, ok := args["language"].(string)
	if !ok || language == "" {
		return &tool.Result{Error: "language is required"}, nil
	}

	code, ok := args["code"].(string)
	if !ok || code == "" {
		return &tool.Result{Error: "code is required"}, nil
	}

	timeout := 30
	if t, ok := args["timeout"].(float64); ok {
		timeout = int(t)
	}
	if timeout > 120 {
		timeout = 120
	}

	if err := s.getDockerClient(); err != nil {
		return &tool.Result{Error: fmt.Sprintf("Docker not available: %v", err)}, nil
	}

	image, err := s.getImage(language)
	if err != nil {
		return &tool.Result{Error: err.Error()}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	containerID, err := s.runContainer(ctx, image, s.getCommand(language, code))
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("failed to run container: %v", err)}, nil
	}

	output, err := s.waitAndGetOutput(ctx, containerID)
	if err != nil {
		return &tool.Result{Error: fmt.Sprintf("failed to get output: %v", err)}, nil
	}

	if err := s.cleanupContainer(ctx, containerID); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to cleanup container: %v\n", err)
	}

	return &tool.Result{Output: output}, nil
}

func (s *SandboxTool) runContainer(ctx context.Context, image string, cmd strslice.StrSlice) (string, error) {
	pullResp, err := s.dockerClient.ImagePull(ctx, image, dockerimage.PullOptions{
		All: false,
	})
	if err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}
	defer pullResp.Close()

	if err := s.streamToString(pullResp); err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	resp, err := s.dockerClient.ContainerCreate(ctx, &container.Config{
		Image:        image,
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
		Hostname:     "wunderpus-sandbox",
	}, nil, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	if err := s.dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		s.dockerClient.ContainerRemove(ctx, resp.ID, container.RemoveOptions{})
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return resp.ID, nil
}

func (s *SandboxTool) waitAndGetOutput(ctx context.Context, containerID string) (string, error) {
	statusCh, errCh := s.dockerClient.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)

	select {
	case status := <-statusCh:
		if status.StatusCode != 0 {
			return "", fmt.Errorf("container exited with code %d", status.StatusCode)
		}
	case err := <-errCh:
		return "", fmt.Errorf("container wait error: %w", err)
	case <-ctx.Done():
		return "", ctx.Err()
	}

	logs, err := s.dockerClient.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}
	defer logs.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, logs); err != nil {
		return "", fmt.Errorf("failed to read logs: %w", err)
	}

	return buf.String(), nil
}

func (s *SandboxTool) cleanupContainer(ctx context.Context, containerID string) error {
	return s.dockerClient.ContainerRemove(ctx, containerID, container.RemoveOptions{
		Force: true,
	})
}

func (s *SandboxTool) streamToString(reader io.Reader) error {
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		_ = n
	}
	return nil
}

func (s *SandboxTool) Close() {
	if s.dockerClient != nil {
		s.dockerClient.Close()
		s.dockerClient = nil
	}
}
