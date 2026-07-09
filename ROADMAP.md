# Livery Stable — Roadmap

Private file-sharing service for musicians and music producers to share audio as works-in-progress or reference material. Files stream directly in the browser. Built with a Go backend and a lightweight JavaScript frontend.

---

## Open Design Decisions

These must be resolved before or during the relevant phase. Each has a recommendation.

### Frontend Framework: Svelte vs. React

**Recommendation: Svelte.**
Svelte compiles to vanilla JS with no runtime dependency, producing smaller bundles and simpler component code. For a media-heavy app where load time and playback responsiveness matter, this is a real advantage. React has a larger ecosystem and talent pool, which may matter more if this grows into a team project. If broad contributor access is a priority down the road, React is the safer long-term bet.

**Decide before:** Phase 3 (frontend player work begins).

---

### Object Storage: S3 vs. Cloudflare R2 vs. GCS ✓

**Decision: Cloudflare R2.** No egress fees, which matters for repeated audio streaming of large files. S3-compatible API — use the AWS Go SDK (`aws-sdk-go-v2`) pointed at the R2 endpoint. Presigned URLs and multipart upload are both supported.

**Follow-up to revisit:** `aws-sdk-go-v2` pulls in a lot of AWS-specific machinery (STS, SSO, SSOOIDC, a "signin" service) that a single S3-compatible bucket with static credentials never touches. `github.com/minio/minio-go/v7` is purpose-built for S3-compatible stores (R2, MinIO, DigitalOcean Spaces included), has a much smaller dependency footprint, and still handles multipart upload and presigned URLs. Worth migrating `storage/r2.go` to it once R2 credentials are provisioned and there's a live bucket to validate the switch against.

---

### Cloud Deployment Target

**Context:** The operator has deep AWS and Kubernetes experience but wants to evaluate cost and DX alternatives.

**Recommendation: Fly.io for MVP; DigitalOcean Kubernetes (DOKS) if Kubernetes is preferred.**

- **Fly.io** — lowest cost at MVP scale (~$15–25/month for app + Postgres), excellent DX, built-in managed Postgres, no Kubernetes overhead. `fly deploy` replaces the entire infra/ops workflow. Right choice for getting the product running quickly. Less control and harder to migrate off if you outgrow it.
- **DigitalOcean Kubernetes (DOKS)** — free control plane (vs. ~$72/month for AWS EKS), cheaper egress ($0.01/GB vs. AWS's $0.09/GB — significant for audio streaming), standard Kubernetes manifests that are portable elsewhere. Good choice if you want to stay sharp on Kubernetes or anticipate needing it sooner. DigitalOcean Spaces is S3-compatible and keeps everything on one platform.
- **AWS EKS** — most ecosystem depth and managed service integration, but hardest to justify at this scale: ~$120+/month floor before any application work. Reserve for when the user base and operational complexity warrant it.
- **AWS ECS (Fargate)** — no K8s control plane fee, simpler than EKS, but AWS-proprietary (manifests don't transfer). A reasonable middle ground if already invested in the AWS ecosystem.

The architecture should be kept portable regardless of the initial choice: standard Docker containers, environment-based config, no hard dependencies on platform-specific services.

**Decide before:** Phase 5 (production deployment).

---

### Authentication Strategy: Self-hosted vs. Managed ✓

**Decision: Self-hosted JWT auth.** Email/password registration and login, bcrypt password hashing, JWTs issued as `HttpOnly` cookies, sessions stored in the DB for true logout via revocation. If the app grows into a consumer product, revisit managed auth (e.g., Clerk, Auth0) or OAuth2 (Google login).

---

### Streaming Strategy: Presigned URL Redirect vs. Server-side Proxy

Two approaches for serving audio to clients:

- **Presigned URL redirect:** The server generates a time-limited signed URL pointing directly at the object store, then redirects the client. The audio data never touches the server. Simpler, no server bandwidth cost, but the URL is briefly visible to the client.
- **Server-side proxy:** The server fetches from object storage and streams to the client using HTTP range requests (206 Partial Content). More control, better for access revocation mid-stream, higher server resource usage.

**Recommendation: Presigned URL redirect for MVP.** Simpler and sufficient for a trusted user base. Revisit proxy approach if fine-grained access control mid-stream becomes a requirement.

**Decide before:** Phase 3 (audio streaming).

---

## File Constraints

- **Supported formats:** MP3, AAC, WAV, AIFF
- **Maximum file size:** 2 GB
  *(accommodates a 45-minute stereo WAV at 24-bit/96kHz, which is ~1.5 GB)*
- Enforce limit at upload time on both client and server.

---

## Phases

### Phase 0 — Skeleton ✓

Basic Go HTTP server with graceful shutdown. No application logic yet.

---

### Phase 1 — Data Layer & Authentication ✓

**Goal:** Persistent storage and authenticated user sessions.

- PostgreSQL via Docker Compose (dev on `:5432`, test on `:5433`)
- `golang-migrate` with SQL migrations embedded in the binary; runs automatically on startup
- Schema: `users` (id, email, password_hash, role, created_at), `sessions` (id, user_id, expires_at, revoked_at)
- Register, login, logout, and `/auth/me` endpoints
- bcrypt password hashing; JWTs (HS256) issued as `HttpOnly` cookies
- `requireAuth` middleware validates signature, expiry, and DB session revocation
- `DATABASE_URL` and `JWT_SECRET` required as environment variables
- Integration tests against a real database; no mocks

---

### Phase 2 — File Upload & Storage

**Goal:** Authenticated users can upload audio files to object storage.

**Backend only.** No frontend exists yet (framework decision is deferred to Phase 3), so
this phase is scoped to curl/Postman-testable HTTP endpoints — no upload UI or client-side
progress indicator yet.

- Storage built against a `Store` interface: a local-filesystem implementation for dev/CI,
  and a Cloudflare R2 implementation (S3-compatible, via `aws-sdk-go-v2`) selected once R2
  credentials are supplied via env vars
- Multipart upload support for large files (required for > 5 MB reliably, essential for 2 GB files), handled transparently by the S3 upload manager — no custom chunked/resumable protocol needed since there's no client yet
- Server-side validation: file format (hand-rolled magic-byte sniffing), file size limit
- Schema: `files` table (id, owner_id, filename, format, size_bytes, duration_seconds, storage_key, uploaded_at, deleted_at)
- List files endpoint (authenticated, scoped to owner)
- Delete file endpoint (soft delete; storage object removed asynchronously via an in-process worker)
- Unit tests for upload validation logic

---

### Phase 3 — Audio Streaming & Browser Player

**Goal:** Users can listen to their files in the browser.

- Audio serving endpoint: validates auth, resolves storage key, returns presigned URL (or proxies with range request support — see design decision above)
- Correct `Content-Type` and `Accept-Ranges` headers
- Frontend: audio player component
  - Play / pause / seek
  - Volume control
  - Elapsed time and total duration display
  - Loading/buffering state
- Waveform visualization using [wavesurfer.js](https://wavesurfer.xyz/) — **include in MVP** (acceptable complexity, high UX value for musicians)
  - Pre-generate waveform peak data on upload (store as JSON in DB) to avoid client-side decode latency
- Keyboard shortcuts: spacebar to play/pause

---

### Phase 4 — Sharing

**Goal:** Authenticated users can share a file with anyone via a private link, without requiring the recipient to have an account.

- Schema: `share_tokens` table (id, file_id, created_by, token [UUID], expires_at, revoked_at)
- Create share link endpoint (authenticated): generates a UUID token, stores with 7-day expiry
- Public share endpoint `/s/{token}`: validates token (exists, not expired, not revoked), serves audio using the same streaming strategy as Phase 3
- Share links page: list active shares for a file, show expiry, allow revocation
- Share links auto-expire; expired tokens return 410 Gone
- No account required for recipients — they get audio playback only (no upload, no listing)
- The public player UI should be minimal: play/pause/seek, file name, expiry notice

---

### Phase 5 — User Management & Admin

**Goal:** The admin can manage the user base for the private tool.

- Invite-only registration: admin generates an invite code/link; new users can only register with a valid invite
- Admin UI: list users, deactivate accounts, view storage usage per user
- User settings: change email, change password
- Basic storage quota per user (configurable, not enforced automatically in MVP — just visible)

---

### Phase 6 — Production Deployment

**Goal:** The app runs reliably in a cloud environment.

- Resolve deployment target (see design decision above)
- Dockerfile is already present; ensure it's production-ready (non-root user, minimal base image)
- Managed PostgreSQL provisioning
- Object storage bucket setup with appropriate access policies
- Environment-based configuration (no secrets in code)
- TLS termination (via reverse proxy or platform)
- Basic health check endpoint for uptime monitoring
- CI: build, lint, and test pipeline (already partially in place)
- CD: automated deploy on merge to `main`

---

## Post-MVP Backlog

These are worth designing for from the start (avoid data model decisions that block them), but are not in scope for MVP.

| Feature | Notes |
|---|---|
| Playlists | Users create ordered playlists of files; share a playlist with one link |
| Timestamp links | Link to a specific playback position (e.g., `/s/{token}?t=1m2s`) |
| Timestamped comments | Leave a comment at a specific point in the audio (à la SoundCloud) |
| File versioning | Upload a new version of a file, preserving the old one; share links can target a specific version |
| OAuth2 login | "Sign in with Google" for consumer launch |
| Invite by email | Send a share link directly to an email address from within the app |
| Consumer onboarding | Self-serve registration, pricing/plans, billing integration |
| Mobile-responsive UI | Ensure the player works well on phones |
| Notifications | Email notification when a shared file is accessed |
| Analytics | Play counts, share link open tracking |
| Download option | Allow recipients to optionally download the file (owner controls this per share) |
