# GitHub Actions CI Fix Plan

## Identified Errors

### 1. Lint Job Failure
- **Error**: `golangci-lint: the Go language version (go1.24) used to build golangci-lint is lower than the targeted Go version (1.25.0)`
- **Location**: `1_lint.txt:200-201`

### 2. Test Jobs Failure (Go 1.24 & 1.23)
- **Error**: `go: no such tool "covdata"` 
- **Location**: Multiple packages fail with this error when running tests with coverage
- **Root cause**: Go 1.25.0 toolchain incompatibility

### 3. Generate Job Failure
- **Error**: `exec: "stringer": executable file not found in $PATH`
- **Location**: `4_generate.txt:262-263`

---

## Root Cause

All jobs specify a specific Go version (1.23 or 1.24) but the `GOTOOLCHAIN=auto` environment variable causes Go to automatically download a newer version (1.25.0), creating incompatibilities:
- golangci-lint v1.64.8 was built with Go 1.24, but runs with Go 1.25.0
- The `covdata` tool has issues with the newer Go version
- The `stringer` tool is not installed in the environment

---

## Fix Steps

### Step 1: Disable GOTOOLCHAIN Auto-Download
In the workflow file (`.github/workflows/*.yml`), add `GOTOOLCHAIN: "local"` to prevent automatic toolchain downloads.

```yaml
- name: Set up Go
  uses: actions/setup-go@v5
  with:
    go-version: '1.24'
  env:
    GOTOOLCHAIN: "local"
```

Or set it as a separate step:
```yaml
- name: Disable Go toolchain auto-download
  run: echo "GOTOOLCHAIN=local" >> $GITHUB_ENV
```

### Step 2: Install Stringer Tool for Generate Job
Add a step to install the `stringer` tool before running `go generate`:

```yaml
- name: Install stringer
  run: go install golang.org/x/tools/cmd/stringer@latest
```

### Step 3: Update or Remove Coverage Profile (Optional)
The `covdata` errors occur when using `-coverprofile=coverage.out`. Consider either:
- Updating Go version to match (1.25)
- Removing coverage for now if not critical

---

## Expected Result

After these changes:
1. Lint job will use Go 1.24 consistently without version mismatch
2. Test jobs will run without covdata errors
3. Generate job will have access to stringer tool
