# ADR-003: Config Storage & Rollback Strategy

**Status:** Approved  
**Date:** 2026-03-09  
**Decision Maker:** Architect  
**Implementation Feature:** #32 (Config Builder — Save & History)

---

## Context

Feature #5 (Configuration Builder) requires saving Alertmanager configuration changes with the ability to:

1. **View history** of past configurations
2. **Rollback** to previous versions
3. **Diff** current vs. previous versions
4. **Atomic writes** (no partial updates on failure)

**Current Architecture:**
- Alertmanager config is stored as YAML file on disk
- Currently loaded on startup, updated via external means (git, manual edits)
- No built-in version control in Alertmanager itself

**Requirements:**
- Support two save modes: **Disk** (local file) and **Git** (GitOps)
- Maintain last N versions (suggested: last 20 versions)
- Enable atomic writes (no data loss on failure)
- Support both immediate save and scheduled save (optional)
- Graceful degradation if backup mechanism fails

**Constraints:**
- No external database (stateless architecture)
- Must work in containerized environment
- Must not require additional infrastructure
- Keep backup files small (YAML is already compact)

---

## Options Considered

### Option A: Dual-Mode Strategy — Git + Disk Backup (Recommended)

**Description:**

**Mode 1: Git-based (default for GitOps workflows)**
- Save to git repository
- Each save = one commit with message
- History via `git log --patch`
- Rollback = `git revert` or `git checkout` previous version
- Push to remote (optional webhook trigger)

**Mode 2: Disk-based (fallback for file-based deployments)**
- Save to main config file
- Keep rotating backup files: `alertmanager.yml.bak.1`, `.bak.2`, ..., `.bak.N`
- History stored locally as timestamped snapshots
- Rollback = restore from backup file
- No remote synchronization

**Hybrid:**
- User can choose at save time (radio button in UI)
- Git preferred if repo configured
- Disk fallback always available

---

### Option A-1: Git Implementation

**When user clicks "Save" with Git mode:**

```
1. Read current config from UI (JSON)
2. Convert to YAML
3. Write temp file
4. `git add alertmanager.yml`
5. `git commit -m "Config updated: Added routing for team-alerts"`
6. Optional: `git push origin main` (webhook-triggered deployment)
7. Return commit hash to UI
```

**Rollback:**
```
1. `git log --oneline` → fetch last N commits
2. User selects commit to restore
3. `git checkout <hash> -- alertmanager.yml`
4. Restart Alertmanager (or reload config via endpoint)
5. Display confirmation
```

**Diff:**
```
1. `git diff HEAD~1 HEAD -- alertmanager.yml` → structured diff
2. Parse YAML before/after, return line-by-line changes
3. Display in UI with syntax highlighting
```

**Pros (Git mode):**
- ✅ Full audit trail (git log shows who, when, why)
- ✅ Distributed backups (push to remote)
- ✅ Integrates with GitOps workflows
- ✅ No size constraints (git compresses efficiently)
- ✅ Works across deployments

**Cons (Git mode):**
- ❌ Requires git repository initialized
- ❌ Requires ssh keys or HTTPS credentials for push
- ❌ More complex setup

---

### Option A-2: Disk Backup Implementation

**When user clicks "Save" with Disk mode:**

```
1. Read current config from UI (JSON)
2. Convert to YAML
3. Read current config from disk (if exists)
4. If current ≠ new:
   a. Rotate backups: .bak.1 → .bak.2, etc.
   b. Move current → .bak.1
   c. Write new config to main file
   d. Write metadata (timestamp, user, diff summary) to .metadata.json
5. Return success with backup info
```

**Rotation logic:**
```
ls -1 alertmanager.yml.bak.* | sort -V | tail -1
# alertmanager.yml.bak.20

# If N=20 reached, delete oldest
rm alertmanager.yml.bak.20

# Shift all: .bak.1 → .bak.2, .bak.2 → .bak.3, etc.
for i in {19..1}; do mv alertmanager.yml.bak.$i alertmanager.yml.bak.$((i+1)); done
```

**History endpoint:**
```
GET /api/config/history?limit=10
→ [
  { version: 1, timestamp: "2026-03-09T10:00:00Z", user: "alice", summary: "Added slack receiver" },
  { version: 2, timestamp: "2026-03-09T10:15:00Z", user: "bob", summary: "Updated routing" },
  ...
]
```

**Rollback:**
```
1. POST /api/config/rollback/{version}
2. Verify version exists: alertmanager.yml.bak.{version}
3. Rotate current to .bak.1
4. Copy .bak.{version} → alertmanager.yml
5. Update .metadata.json (rollback marker)
6. Return success
```

**Diff:**
```
1. GET /api/config/diff/{version1}/{version2}
2. Read both .bak files (or main + .bak)
3. Parse YAML, compute structured diff
4. Return line-by-line changes
```

**Pros (Disk mode):**
- ✅ No external system required
- ✅ Simple backup mechanism
- ✅ Works in air-gapped environments
- ✅ Fast local access
- ✅ No authentication needed

**Cons (Disk mode):**
- ❌ Limited to local backups (no remote sync)
- ❌ Backup files consume disk space (mitigated by limiting N)
- ❌ No distributed backup
- ❌ No audit trail (unless metadata.json enhanced)

---

### Option B: Database-backed Storage (Not Recommended)

**Description:** Store all versions in PostgreSQL with timestamps and metadata.

**Cons:**
- ❌ Requires external database (stateless principle violated)
- ❌ Adds dependency
- ❌ Overkill for config versioning
- ❌ Conflicts with "zero external state" goal

**Not Recommended**

---

### Option C: External Version Control (S3, etc.)

**Description:** Store backups in S3 or similar object storage.

**Pros:**
- ✅ Distributed, reliable backups
- ✅ Multi-region support (if using S3 replication)

**Cons:**
- ❌ Requires AWS credentials/setup
- ❌ Network dependency
- ❌ Cost
- ❌ Over-engineered for MVP

**Not Recommended for MVP** (can be added post-Phase 1)

---

## Decision

**✅ APPROVED: Dual-Mode Strategy (Git + Disk) — Option A**

**Rationale:**

1. **Flexibility:**
   - Git for modern GitOps workflows (preferred for production)
   - Disk for simple file-based deployments (fallback)
   - User chooses at save time

2. **No external dependencies:**
   - Git is already present (standard DevOps practice)
   - Disk is always available
   - No additional infrastructure

3. **Auditability:**
   - Git mode: Full commit history with author + message
   - Disk mode: Metadata file with timestamp + user

4. **Rollback capability:**
   - Both modes support easy rollback
   - UI presents list of previous versions
   - One-click restore

5. **Aligns with AlertLens goals:**
   - Configuration as code (Git mode)
   - Simple file-based (Disk mode)
   - Stateless (no database)

6. **Post-MVP extensibility:**
   - Can add S3 backups later
   - Can add database for advanced features
   - Dual-mode foundation remains stable

---

## Implementation Details

### Backend Implementation

**File:** `internal/api/handlers/config.go`

**PUT /api/config endpoint:**

```go
type UpdateConfigRequest struct {
    Config    ConfigResponse `json:"config"`
    SaveMode  string         `json:"save_mode"` // "git" | "disk"
    GitOptions *GitOptions   `json:"git_options,omitempty"`
    Comment   string         `json:"comment"`  // Commit message or rollback reason
}

type GitOptions struct {
    Branch  string `json:"branch"` // e.g., "main", "config-updates"
    Push    bool   `json:"push"`   // Push to remote after commit
    Author  string `json:"author"` // User info
}

func (h *Handler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
    var req UpdateConfigRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        writeJSONError(w, 400, "invalid request")
        return
    }

    // Validate config
    if err := validateConfig(req.Config); err != nil {
        writeJSONError(w, 400, fmt.Sprintf("invalid config: %v", err))
        return
    }

    // Choose save strategy
    var result ConfigSaveResult
    if req.SaveMode == "git" {
        result, err = h.saveConfigToGit(r.Context(), req)
    } else {
        result, err = h.saveConfigToDisk(r.Context(), req)
    }

    if err != nil {
        writeJSONError(w, 500, fmt.Sprintf("save failed: %v", err))
        return
    }

    writeJSON(w, result)
}

// Git-based save
func (h *Handler) saveConfigToGit(ctx context.Context, req UpdateConfigRequest) (ConfigSaveResult, error) {
    // 1. Convert config JSON to YAML
    yamlBytes, err := toYAML(req.Config)
    if err != nil {
        return ConfigSaveResult{}, err
    }

    // 2. Write to temp file, validate
    tmpFile := filepath.Join(h.configDir, ".alertmanager.yml.tmp")
    if err := os.WriteFile(tmpFile, yamlBytes, 0o644); err != nil {
        return ConfigSaveResult{}, err
    }

    // 3. Validate YAML with official parser
    var cfg alertmanagerConfig.Config
    if err := yaml.UnmarshalStrict(yamlBytes, &cfg); err != nil {
        os.Remove(tmpFile) // cleanup
        return ConfigSaveResult{}, fmt.Errorf("invalid YAML: %w", err)
    }

    // 4. Git operations
    configFile := filepath.Join(h.configDir, "alertmanager.yml")
    
    // Backup current config
    if err := os.Rename(configFile, configFile+".bak.tmp"); err != nil {
        os.Remove(tmpFile)
        return ConfigSaveResult{}, err
    }

    // Move temp file to config file
    if err := os.Rename(tmpFile, configFile); err != nil {
        os.Rename(configFile+".bak.tmp", configFile) // rollback
        return ConfigSaveResult{}, err
    }

    // Git add & commit
    branch := req.GitOptions.Branch
    if branch == "" {
        branch = "main"
    }

    cmd := exec.CommandContext(ctx, "git", "-C", h.configDir, "add", "alertmanager.yml")
    if err := cmd.Run(); err != nil {
        os.Rename(configFile+".bak.tmp", configFile) // rollback
        return ConfigSaveResult{}, fmt.Errorf("git add failed: %w", err)
    }

    author := req.GitOptions.Author
    if author == "" {
        author = "alertlens-bot"
    }

    commitMsg := req.Comment
    if commitMsg == "" {
        commitMsg = "Update alertmanager configuration"
    }

    cmd = exec.CommandContext(ctx, "git", "-C", h.configDir,
        "-c", fmt.Sprintf("user.name=%s", author),
        "-c", "user.email=alertlens@example.com",
        "commit", "-m", commitMsg)
    if err := cmd.Run(); err != nil {
        os.Rename(configFile+".bak.tmp", configFile) // rollback
        return ConfigSaveResult{}, fmt.Errorf("git commit failed: %w", err)
    }

    // Get commit hash
    cmd = exec.CommandContext(ctx, "git", "-C", h.configDir, "rev-parse", "HEAD")
    var out bytes.Buffer
    cmd.Stdout = &out
    if err := cmd.Run(); err != nil {
        return ConfigSaveResult{}, err
    }
    commitHash := strings.TrimSpace(out.String())

    // Optional: git push
    if req.GitOptions.Push {
        cmd = exec.CommandContext(ctx, "git", "-C", h.configDir, "push", "origin", branch)
        if err := cmd.Run(); err != nil {
            // Don't fail save if push fails, but log warning
            log.Printf("git push failed (config still saved locally): %v", err)
        }
    }

    // Cleanup backup
    os.Remove(configFile + ".bak.tmp")

    return ConfigSaveResult{
        SaveMode:   "git",
        CommitHash: commitHash,
        Timestamp:  time.Now(),
    }, nil
}

// Disk-based save with rotation
func (h *Handler) saveConfigToDisk(ctx context.Context, req UpdateConfigRequest) (ConfigSaveResult, error) {
    // 1. Convert config JSON to YAML
    yamlBytes, err := toYAML(req.Config)
    if err != nil {
        return ConfigSaveResult{}, err
    }

    // 2. Validate YAML
    var cfg alertmanagerConfig.Config
    if err := yaml.UnmarshalStrict(yamlBytes, &cfg); err != nil {
        return ConfigSaveResult{}, fmt.Errorf("invalid YAML: %w", err)
    }

    configFile := filepath.Join(h.configDir, "alertmanager.yml")
    backupDir := filepath.Join(h.configDir, "backups")

    // Create backup dir if needed
    os.MkdirAll(backupDir, 0o755)

    // 3. Atomic write: write to temp, then rename
    tmpFile := configFile + ".tmp"
    if err := os.WriteFile(tmpFile, yamlBytes, 0o644); err != nil {
        return ConfigSaveResult{}, fmt.Errorf("write temp failed: %w", err)
    }

    // 4. If current config exists, rotate backups
    var backupVersion int
    if _, err := os.Stat(configFile); err == nil {
        // Rotate existing backups
        backupVersion, err = h.rotateBackups(backupDir)
        if err != nil {
            os.Remove(tmpFile)
            return ConfigSaveResult{}, fmt.Errorf("rotate backups failed: %w", err)
        }

        // Move current to .bak.1
        currentBackup := filepath.Join(backupDir, fmt.Sprintf("alertmanager.yml.bak.1"))
        if err := os.Rename(configFile, currentBackup); err != nil {
            os.Remove(tmpFile)
            return ConfigSaveResult{}, fmt.Errorf("backup current failed: %w", err)
        }
    }

    // 5. Move temp file to main location
    if err := os.Rename(tmpFile, configFile); err != nil {
        return ConfigSaveResult{}, fmt.Errorf("move config failed: %w", err)
    }

    // 6. Write metadata (for audit trail)
    metadata := ConfigMetadata{
        Version:   backupVersion,
        Timestamp: time.Now(),
        User:      req.GitOptions.Author, // Can reuse author field
        Comment:   req.Comment,
    }
    metadataFile := filepath.Join(backupDir, "metadata.json")
    if metaJSON, err := json.MarshalIndent([]ConfigMetadata{metadata}, "", "  "); err == nil {
        os.WriteFile(metadataFile, metaJSON, 0o644)
    }

    return ConfigSaveResult{
        SaveMode:       "disk",
        BackupVersion:  backupVersion,
        Timestamp:      time.Now(),
    }, nil
}

func (h *Handler) rotateBackups(backupDir string) (int, error) {
    maxBackups := 20
    
    // Find all backup files
    entries, err := filepath.Glob(filepath.Join(backupDir, "alertmanager.yml.bak.*"))
    if err != nil {
        return 0, err
    }

    // Sort and keep track of versions
    sort.Strings(entries)
    
    // If we have max backups, delete oldest
    if len(entries) >= maxBackups {
        os.Remove(entries[0])
    }

    // Rotate: .bak.1 → .bak.2, etc.
    for i := len(entries) - 1; i >= 0; i-- {
        oldPath := entries[i]
        newPath := strings.TrimSuffix(oldPath, fmt.Sprintf(".%d", i)) + fmt.Sprintf(".%d", i+1)
        os.Rename(oldPath, newPath)
    }

    return len(entries) + 1, nil
}
```

**GET /api/config/history endpoint:**

```go
type ConfigVersion struct {
    Version     int       `json:"version"`
    Timestamp   time.Time `json:"timestamp"`
    User        string    `json:"user"`
    Comment     string    `json:"comment"`
    SaveMode    string    `json:"save_mode"` // "git" | "disk"
    CommitHash  string    `json:"commit_hash,omitempty"`
}

func (h *Handler) GetConfigHistory(w http.ResponseWriter, r *http.Request) {
    // Implementation depends on save mode:
    // - Git: run `git log --oneline | head -N`
    // - Disk: read from backups/metadata.json
}
```

**POST /api/config/rollback/{version} endpoint:**

```go
func (h *Handler) RollbackConfig(w http.ResponseWriter, r *http.Request) {
    version := chi.URLParam(r, "version")
    
    // Determine save mode and rollback accordingly
    // - Git: git checkout <version> -- alertmanager.yml
    // - Disk: restore from .bak.{version}
}
```

---

### Frontend Implementation

**File:** `web/src/routes/config/review/+page.svelte`

```svelte
<script>
  import YAMLPreview from '../../../components/YAMLPreview.svelte';
  import DiffViewer from '../../../components/DiffViewer.svelte';
  import SaveModeSelector from '../../../components/SaveModeSelector.svelte';

  let step = 1; // 1: preview, 2: diff, 3: save-mode, 4: confirm

  let formData = {};
  let saveMode = 'disk'; // or 'git'
  let comment = '';
  let gitBranch = 'main';
  let gitPush = false;

  async function handleSave() {
    const payload = {
      config: formData,
      save_mode: saveMode,
      comment: comment
    };

    if (saveMode === 'git') {
      payload.git_options = {
        branch: gitBranch,
        push: gitPush,
        author: currentUser.name
      };
    }

    const response = await fetch('/api/config', {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });

    if (response.ok) {
      // Show success message, redirect to dashboard
    } else {
      // Show error
    }
  }
</script>

<div class="config-review">
  {#if step === 1}
    <h2>Preview Configuration</h2>
    <YAMLPreview config={formData} />
    <button on:click={() => step = 2}>Next</button>

  {:else if step === 2}
    <h2>Review Changes</h2>
    <DiffViewer oldConfig={currentConfig} newConfig={formData} />
    <button on:click={() => step = 1}>Back</button>
    <button on:click={() => step = 3}>Next</button>

  {:else if step === 3}
    <h2>Save Mode</h2>
    <SaveModeSelector bind:mode={saveMode} bind:branch={gitBranch} bind:push={gitPush} />
    <textarea placeholder="Commit message or save reason" bind:value={comment} />
    <button on:click={() => step = 2}>Back</button>
    <button on:click={() => step = 4}>Review & Save</button>

  {:else if step === 4}
    <h2>Confirm Save</h2>
    <p>Save mode: <strong>{saveMode}</strong></p>
    <p>Comment: <strong>{comment}</strong></p>
    <button on:click={handleSave}>Confirm</button>
    <button on:click={() => step = 3}>Back</button>
  {/if}
</div>
```

---

### Atomic Write Safety

**Pattern used:**
1. Write to temp file
2. Validate content
3. Rename temp → main (atomic on POSIX systems)
4. On failure, cleanup temp file
5. No partial updates possible

---

## Testing Strategy

### Unit Tests

```go
// Test disk backup rotation
func TestRotateBackups(t *testing.T) {
    // Create 20 backups, verify oldest is deleted
}

// Test atomic write
func TestAtomicWrite(t *testing.T) {
    // Write, then fail during validation
    // Verify original config untouched
}

// Test Git commit
func TestGitCommit(t *testing.T) {
    // Mock git commands
    // Verify commit message + author
}
```

### Integration Tests

```go
// Full flow: save → history → rollback → restore
func TestSaveAndRollback(t *testing.T) {
    // 1. Save v1
    // 2. Save v2
    // 3. Fetch history (should have 2 versions)
    // 4. Rollback to v1
    // 5. Verify config matches v1
}
```

---

## Security Considerations

1. **File permissions:** Config files readable only by alertlens process (`0o644` for main, `0o755` for dir)
2. **Git credentials:** Use SSH keys or HTTPS tokens from environment
3. **Backup files:** Treated same as main config (same permissions)
4. **Atomic writes:** Prevent partial updates from being read
5. **Validation:** Always validate config before write (prevents malformed YAML)

---

## Dependencies & Coordination

- **Depends on:** None (core feature)
- **Enables:** Config Builder (#5) complete feature
- **Integrates with:** RBAC (#24) for audit trail

---

## Success Criteria

- [ ] Git save mode: commit + optional push works
- [ ] Disk save mode: backup rotation works
- [ ] History endpoint returns last N versions
- [ ] Rollback restores previous config
- [ ] Diff endpoint shows changes
- [ ] Atomic writes prevent partial updates
- [ ] Metadata stored (timestamp, user, comment)
- [ ] Tests pass (≥80% coverage)
- [ ] Security review passed

---

## Timeline

- **Duration:** 2 days (part of Feature #5, sub-task #32)
  - Day 1: Implement Git + Disk save backends
  - Day 2: History, rollback, diff endpoints + tests

---

## Related ADRs

- ADR-001: Routing Tree Visualization
- ADR-002: Form Framework Selection
- ADR-004: Real-time Update Strategy

---

## Approval Sign-off

- **Architect:** ✅ Approved 2026-03-09
- **Developer:** ⬜ To confirm on implementation
- **Security:** ✅ File-based backups acceptable
- **DevOps:** ✅ Git integration aligns with GitOps practices

---

## Notes

1. **Disk mode is always available:** Even if Git not initialized, system can save locally
2. **Git history is preserved:** Even if switching to disk mode, git history remains
3. **Rollback is reversible:** Each rollback is itself a new commit (if Git mode)
4. **Metadata.json is JSON Lines compatible:** Easy to parse and extend

---

**End of ADR-003**
