# Architecture Decisions

## Decision: RSI Embedding Backend
**Date:** 2026-03-23
**Status:** Accepted

**Context:** The Recursive Self-Improvement (RSI) engine requires vector embeddings of all functions in the Go codebase. We need to decide whether to use a local embedding model (e.g., `ollama nomic-embed-text`) or a cloud API (`text-embedding-3-small`).

**Decision:** We choose **API (`text-embedding-3-small`)** as the primary embedding backend.

**Rationale:**
1. **Developer Experience:** Does not require developers to install, run, and maintain an Ollama instance alongside the application.
2. **Quality & Cost:** `text-embedding-3-small` is extremely cheap and offers very high quality semantic representations for code out-of-the-box.
3. **Speed:** For a typical Go repository, making a few API calls is fast enough and doesn't monopolize local CPU/GPU resources that might impact the application's runtime or the developer's machine.

Local embeddings remain a viable future option for fully air-gapped deployments, but the API path is the default for MVP.
