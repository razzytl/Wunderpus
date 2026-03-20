# Verification Before Completion

This documentation codifies Wunderpus’ policy for verification before making any completion claim. It ensures that claims of success are backed by fresh, verifiable evidence.

The Iron Law
- NO COMPLETION CLAIMS WITHOUT FRESH VERIFICATION EVIDENCE

The Gate Function
1. IDENTIFY: Determine which command proves the claim.
2. RUN: Execute the full command (fresh, complete).
3. READ: Read the full output, check exit code, and count failures.
4. VERIFY: Does the output confirm the claim? If not, state the actual status with evidence.
5. ONLY THEN: Make the claim.

Common Failures
| Claim | Requires | Not Sufficient |
|-------|----------|----------------|
| Tests pass | Test command exit code 0 | Previous run reporting only that tests should pass |
| Linter clean | Linter exit code 0 | Extrapolated or partial results |
| Build succeeds | Build command exit code 0 | Linting or logs alone do not prove compilation |
| Bug fixed | Tests reproducing the original symptom now pass | Code changes without end-to-end confirmation |
| Regression test works | Red–Green verification cycle completed | Passing tests without red phase |
| Agent completed | VCS diff shows changes | Solely trusting agent output |
| Requirements met | Plan re-read and checklist verification | Merely claiming coverage without verification |

Red Flags
- Do not claim success if verification has not been performed.
- Avoid language indicating certainty without evidence (e.g., "should pass", "looks good").
- Do not proceed to commit/push/PR without verification results.
- Do not rely on partial verification results to claim completion.

Rationalization Prevention
- Do not substitute confidence for evidence.
- Do not skip verification to save time; verification is the gate to completion.
- Ensure that verification is repeatable and documented.

Key Patterns
- Tests: Implement and run tests; ensure green on new/affected areas.
- Regression tests (TDD red-green): Write red test first, ensure it fails; implement fix; run tests to ensure green.
- Build: Ensure compilation succeeds with the verification command.
- Agent delegation: Verify changes via VCS before accepting agent-supplied success signals.

When To Apply
- Before any success/completion claim or positive state communication.
- Before committing, creating a PR, or moving to the next task.
- Before delegating work to agents or tools.

Bottom Line
- Do not claim completion without running verification.
- Read outputs, verify, and then claim the result.

Getting Started: Example Verification Commands
- Go projects (tests and build):
  - Run tests: `go test ./...`
  - Build: `go build ./...`
- Linting (if configured):
  - `golangci-lint run` (or your repository's lint script)
- Documentation-specific checks (optional):
  - `markdownlint` or your docs tool of choice

Notes
- The exact commands depend on the language and tooling of your project.
- Prefer running commands from a clean state (fresh build/test) to avoid stale results.

This page is intended to be a practical reference for engineers and contributors working in Wunderpus. For workflow guidance, see the Gate Function and Red Flags sections above.
