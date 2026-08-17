# Azure Blob Test Tool

`cmd/azureblob-test-tool` is a standalone CLI for creating and managing Azure Blob test objects outside the Everest portal. Use it when you need a controlled way to seed, inspect, modify, download, or clean up blobs while validating connector, policy, AI, and action workflows.

The same capabilities are also available in the portal under **Azure Blob Workbench** for users who do not want to run command lines.

The tool supports:

- Azure Storage connection string through `--connection-string` or `AZURE_STORAGE_CONNECTION_STRING`
- Full container URL with SAS through `--container-url` or `AZURE_BLOB_CONTAINER_URL`
- Account/service Blob URL with SAS through `--service-url` plus `--container`
- Container discovery through `--op list-containers` when using a connection string or account/service SAS URL
- JSON output for repeatable testing and screenshots
- `--dry-run` for write/delete previews
- guarded delete with `--confirm DELETE`

## Portal Workbench

Start the local Data Agent portal:

```powershell
go run ./cmd/policy-agent
```

Open:

```text
http://localhost:8080
```

Use **Azure Blob Workbench** from the left navigation.

Recommended UI flow:

1. Paste the Azure Storage connection string.
2. Select **Save Credential**.
3. Select **Discover Containers**.
4. Choose a returned container with **Use**.
5. Select an operation such as **List blobs**, **Create blob**, **View blob**, **Set metadata**, **Set tags**, **Download blob**, or **Delete blob**.

Expected UI output:

- Workbench status shows `ok`, `preview`, or `saved`.
- Container discovery populates the container dropdown.
- Blob listing populates selectable blob paths.
- View/create/update operations show object details, metadata, index tags, and content preview.
- Download blob enables **Download Result** in the browser.
- Dry-run is enabled by default on the UI; clear it before intentional create/update/tag/metadata/delete operations.
- Delete requires confirmation text `DELETE`; use dry-run first.
- The header theme toggle switches between day and night themes and persists the choice in the browser.

AI verb and HITL flow:

1. Select or enter a container and blob path.
2. Use a verb button such as **Recommend Tags**, **Complete Metadata**, **Prepare Archive**, **Prepare Quarantine**, or **Delegate Assessment**.
3. Review the editable **Action canvas** produced by AI.
4. Change metadata, index tags, or rationale directly in the canvas if needed.
5. Choose automation mode:
   - `HITL required` routes risky or destructive actions to approval.
   - `Preview only` keeps the recommendation non-mutating.
   - `Auto-apply safe` can apply safe metadata/index tag actions when dry-run is cleared.
6. Use **Apply Canvas To Form** to move the artifact into the operation fields.
7. Use **Preview / Apply Safe** for governed safe actions.
8. Use **Submit HITL** for restricted actions.

Expected AI output:

- Safe metadata/tag recommendations show `HITL: not required`.
- Destructive or movement actions show `HITL: required`.
- If an AI runtime is unavailable, the UI falls back to a local guardrail recommendation so the demo path remains usable.
- Ambient suggestions appear after selecting or viewing an object, without requiring a prompt.
- Delegated assessment reviews the selected container/prefix and returns prioritized recommended verbs.

DSPM content scan flow:

1. View a blob so the Workbench has object details and a content preview.
2. Choose scan tier:
   - `metadata`
   - `rules`
   - `model`
   - `deep`
3. Choose model profile:
   - `builtin-rules`
   - `local-bert-ner`
   - `domain-bert-dspm`
   - `cloud-language-service`
4. Select **Run DSPM Scan**.
5. Review classification, risk score, next tier, detected entities, and checks.
6. Select **Use DSPM Tags** to copy recommended metadata and index tags into the Workbench form.
7. Preview/apply safe tags or submit HITL when the scan requires approval.

Keyword pattern matching:

1. Choose a keyword pack such as `All built-in packs`, `Legal contracts`, or `Finance and HR`.
2. Optionally add custom keyword patterns:

```text
Project Everest|high|CUSTOM_ENTITY|phrase
acquisition target|critical|BUSINESS_SENSITIVE|phrase
customer-[0-9]{5}|medium|CUSTOM_ENTITY|regex
```

3. Optionally add exclusions such as:

```text
sample,template,test
```

4. Run DSPM Scan.

Expected DSPM output:

- `metadata` tier evaluates path, file id, content type, size, metadata, and tags.
- `rules` tier detects common PII/secrets in the visible content sample.
- Keyword packs detect sensitive business terms in path, metadata, tags, and sampled content.
- Custom keywords produce `KEYWORD`, `CUSTOM_ENTITY`, or customer-selected entity types.
- `model` and `deep` tiers show model-profile readiness and recommend external/local model runtime integration when required.
- High-risk or restricted findings set `requires_hitl` to `true`.

The backend endpoint used by the UI is:

```text
POST /v1/tools/azureblob/workbench
```

## Build Check

```powershell
go test ./cmd/azureblob-test-tool
```

Expected output:

```text
?   	everest.local/data-agent-policy-resolver/cmd/azureblob-test-tool	[no test files]
```

## Authentication Options

Connection string:

```powershell
$env:AZURE_STORAGE_CONNECTION_STRING = "<storage-account-connection-string>"
go run ./cmd/azureblob-test-tool --op list-containers
go run ./cmd/azureblob-test-tool --op list --container "<container-name>"
```

Container SAS URL:

```powershell
$env:AZURE_BLOB_CONTAINER_URL = "https://<account>.blob.core.windows.net/<container>?<sas-token>"
go run ./cmd/azureblob-test-tool --op list
```

Service/account SAS URL:

```powershell
go run ./cmd/azureblob-test-tool --op list-containers --service-url "https://<account>.blob.core.windows.net?<sas-token>"
go run ./cmd/azureblob-test-tool --op list --service-url "https://<account>.blob.core.windows.net?<sas-token>" --container "<container-name>"
```

For private-network or air-gapped customer environments, run the tool from a host that has network access to the customer Azure Blob endpoint or private endpoint. The CLI does not require the Everest portal to be running.

## Common Commands

Discover containers from a connection string:

```powershell
$env:AZURE_STORAGE_CONNECTION_STRING = "<storage-account-connection-string>"
go run ./cmd/azureblob-test-tool --op list-containers
```

Discover containers with an account/service SAS URL:

```powershell
go run ./cmd/azureblob-test-tool --op list-containers --service-url "https://<account>.blob.core.windows.net?<sas-token>"
```

List blobs:

```powershell
go run ./cmd/azureblob-test-tool --op list --container "<container-name>" --prefix "everest-test/" --limit 25
```

Create a generated JSON test blob:

```powershell
go run ./cmd/azureblob-test-tool --op create --container "<container-name>" --blob "everest-test/generated.json" --metadata "owner=qa,purpose=everest" --tags "everest_test=true,case=generated"
```

Create a text blob:

```powershell
go run ./cmd/azureblob-test-tool --op create --container "<container-name>" --blob "everest-test/readme.txt" --content "Everest Azure Blob test object" --metadata "owner=qa" --tags "everest_test=true"
```

Upload a local file:

```powershell
go run ./cmd/azureblob-test-tool --op create --container "<container-name>" --blob "everest-test/files/sample.pdf" --file "C:\Temp\sample.pdf" --content-type "application/pdf" --tags "everest_test=true,classification=confidential"
```

View properties, metadata, tags, and a small content preview:

```powershell
go run ./cmd/azureblob-test-tool --op view --container "<container-name>" --blob "everest-test/readme.txt"
```

Update blob content:

```powershell
go run ./cmd/azureblob-test-tool --op update --container "<container-name>" --blob "everest-test/readme.txt" --content "Updated test content" --metadata "owner=qa,version=2" --tags "everest_test=true,state=updated"
```

Set metadata only:

```powershell
go run ./cmd/azureblob-test-tool --op set-metadata --container "<container-name>" --blob "everest-test/readme.txt" --metadata "owner=qa,reviewed=true"
```

Set blob index tags only:

```powershell
go run ./cmd/azureblob-test-tool --op set-tags --container "<container-name>" --blob "everest-test/readme.txt" --tags "everest_test=true,policy_candidate=true"
```

Download a blob:

```powershell
go run ./cmd/azureblob-test-tool --op download --container "<container-name>" --blob "everest-test/readme.txt" --out "C:\Temp\everest-readme.txt"
```

Preview delete:

```powershell
go run ./cmd/azureblob-test-tool --op delete --container "<container-name>" --blob "everest-test/readme.txt" --dry-run
```

Delete after confirmation:

```powershell
go run ./cmd/azureblob-test-tool --op delete --container "<container-name>" --blob "everest-test/readme.txt" --confirm DELETE
```

## Expected Output Examples

List containers:

```json
{
  "operation": "list-containers",
  "status": "ok",
  "auth_mode": "connection_string",
  "count": 2,
  "containers": [
    {
      "name": "customer-demo",
      "last_modified": "2026-06-13T01:25:10Z",
      "etag": "\"0x8DD2A1B2C3D4E5F\"",
      "lease_status": "unlocked",
      "lease_state": "available"
    },
    {
      "name": "policy-test",
      "last_modified": "2026-06-13T01:31:40Z",
      "etag": "\"0x8DD2A1B2C3D4E60\"",
      "lease_status": "unlocked",
      "lease_state": "available"
    }
  ]
}
```

Create success:

```json
{
  "operation": "create",
  "status": "ok",
  "auth_mode": "connection_string",
  "container": "customer-demo",
  "blob": "everest-test/readme.txt",
  "message": "Blob content uploaded.",
  "details": {
    "name": "everest-test/readme.txt",
    "size_bytes": 30,
    "content_type": "text/plain; charset=utf-8",
    "metadata": {
      "owner": "qa"
    },
    "tags": {
      "everest_test": "true"
    }
  },
  "applied": {
    "uploaded_bytes": 30,
    "source": "inline",
    "content_type": "text/plain; charset=utf-8",
    "metadata": {
      "owner": "qa"
    },
    "tags": {
      "everest_test": "true"
    },
    "overwrite": false
  }
}
```

View success:

```json
{
  "operation": "view",
  "status": "ok",
  "auth_mode": "connection_string",
  "container": "customer-demo",
  "blob": "everest-test/readme.txt",
  "details": {
    "name": "everest-test/readme.txt",
    "size_bytes": 30,
    "content_type": "text/plain; charset=utf-8",
    "metadata": {
      "owner": "qa"
    },
    "tags": {
      "everest_test": "true"
    },
    "preview": {
      "bytes": 30,
      "truncated": false,
      "text": "Everest Azure Blob test object"
    }
  }
}
```

Dry-run delete:

```json
{
  "operation": "delete",
  "status": "preview",
  "container": "customer-demo",
  "blob": "everest-test/readme.txt",
  "dry_run": true,
  "message": "No Azure Blob changes were made.",
  "applied": {
    "required_confirm": "DELETE",
    "would_delete_snapshots": true
  }
}
```

## Testing Scenarios

Use these object patterns to validate customer scenarios:

- Normal documents: `everest-test/public/readme.txt`
- Confidential or regulated content: `everest-test/confidential/payroll.csv`
- Large files: upload with `--file` and verify list/view size fields
- Metadata-driven workflows: use `--metadata "owner=finance,retention=7y"`
- Tag-driven workflows: use `--tags "classification=restricted,policy_candidate=true"`
- Connector discovery: create several blobs under one prefix, then scan the same prefix from the portal
- HITL action validation: create a high-risk blob, scan it, approve or reject the pending action in the portal, then use `--op view` to verify metadata/tags changed

## Safety Notes

- `list-containers` works with connection strings and account/service SAS URLs; a container SAS URL only grants access to one known container, so it cannot discover sibling containers.
- `create` fails if the blob exists unless `--overwrite` is provided.
- `update` replaces blob content.
- `set-metadata` replaces the blob metadata set with the provided metadata.
- `set-tags` replaces the blob index tag set with the provided tags.
- `delete` requires `--confirm DELETE`; use `--dry-run` first when testing.
- SAS credentials must include permissions for the operation being tested. For example, tag operations require tag permissions.
