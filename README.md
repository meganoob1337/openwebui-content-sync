# OpenWebUI Content Sync

A Kubernetes-native application that synchronizes content from various sources (GitHub repositories, Confluence spaces) to OpenWebUI knowledge bases using an adapter architecture.

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.

## Features

- **Multi-Source Support**: GitHub repositories and Confluence spaces
- **Adapter Architecture**: Pluggable adapters for different data sources
- **File Diffing**: Only syncs changed files based on content hashing
- **Persistent Storage**: Uses Kubernetes persistent volumes for local file storage
- **Scheduled Sync**: Configurable sync intervals using cron-like scheduling
- **OpenWebUI Integration**: Full integration with OpenWebUI file and knowledge APIs
- **Confluence Support**: Sync entire spaces or specific parent pages with sub-pages

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Data Sources  │    │   Content Sync   │    │   OpenWebUI     │
│   • GitHub      │───▶│   Application    │───▶│   Knowledge     │
│   • Confluence  │    │   (Adapters)     │    │   Base          │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌──────────────────┐
                       │   Local Storage  │
                       │   (PVC)          │
                       └──────────────────┘
```

## API Integration

The connector integrates with OpenWebUI using the following APIs:

- `POST /api/v1/files/` - Upload files to OpenWebUI
- `GET /api/v1/knowledge/` - List knowledge sources
- `POST /api/v1/knowledge/{id}/file/add` - Add file to knowledge
- `POST /api/v1/knowledge/{id}/file/remove` - Remove file from knowledge

## Quick Start

### Prerequisites

- Kubernetes cluster
- OpenWebUI instance running
- GitHub repository access

### 1. Configure Secrets

Update the secrets in `k8s/secrets.yaml`:

```bash
# Encode your OpenWebUI API key
echo -n "your-openwebui-api-key" | base64

# Encode your GitHub token
echo -n "your-github-token" | base64

# Encode your Confluence API key
echo -n "your-confluence-api-key" | base64
```

### 2. Update Configuration

Edit `k8s/configmap.yaml` to set:
- GitHub repositories to sync
- Confluence spaces or parent page IDs
- Knowledge IDs for file association
- Sync interval

### 3. Deploy to Kubernetes

```bash
# Apply all manifests
kubectl apply -f k8s/

# Check deployment status
kubectl get pods -l app=openwebui-content-sync
```

### 4. Local Development

```bash
# Build the application
go build -o connector .

# Run with configuration
./connector -config config.yaml
```

## Usage Examples

### GitHub Adapter

The GitHub adapter syncs files from GitHub repositories to OpenWebUI knowledge bases.

#### Basic GitHub Configuration

```yaml
github:
  enabled: true
  token: "ghp_your_github_token_here"
  repositories:
    - "microsoft/vscode"
    - "facebook/react"
    - "your-org/your-repo"
  knowledge_id: "your-knowledge-base-id"
```

#### GitHub Features

- **Repository Sync**: Syncs all files from specified repositories
- **File Filtering**: Automatically filters out binary files and common ignore patterns
- **Content Hashing**: Only syncs changed files based on SHA256 hashes
- **Branch Support**: Syncs from the default branch (usually `main` or `master`)

#### GitHub Example Output

```
INFO[0000] Syncing files from adapter: github
DEBU[0000] Fetching files from repository: microsoft/vscode
DEBU[0000] Found 1,234 files in repository microsoft/vscode
INFO[0001] Successfully synced file: README.md
INFO[0001] Successfully synced file: package.json
INFO[0001] Successfully synced file: src/index.js
```

### Confluence Adapter

The Confluence adapter syncs pages from Confluence spaces to OpenWebUI knowledge bases.

#### Option 1: Sync Entire Space

```yaml
confluence:
  enabled: true
  base_url: "https://your-domain.atlassian.net"
  username: "your-email@example.com"
  api_key: "your-confluence-api-key"
  spaces:
    - "DOCS"
    - "ENGINEERING"
  knowledge_id: "your-knowledge-base-id"
  page_limit: 100
  include_attachments: true
```

#### Option 2: Sync Specific Parent Page and Sub-pages

```yaml
confluence:
  enabled: true
  base_url: "https://your-domain.atlassian.net"
  username: "your-email@example.com"
  api_key: "your-confluence-api-key"
  spaces: []  # Not needed when using parent_page_ids
  parent_page_ids:  # Specific parent page IDs to process sub-pages
    - "1234567890"
    - "0987654321"  # Add multiple parent page IDs as needed
  knowledge_id: "your-knowledge-base-id"
  page_limit: 0  # No limit
  include_attachments: false
```

#### Confluence Features

- **Space Sync**: Sync all pages from specified Confluence spaces
- **Parent Page Sync**: Sync specific parent pages and all their sub-pages
- **Multiple Parent Pages**: Support for multiple parent page IDs in a single configuration
- **Mixed Configuration**: Can sync both entire spaces and specific parent pages simultaneously
- **HTML to Text**: Converts Confluence HTML content to plain text
- **Filename Sanitization**: Converts page titles to safe filenames (e.g., "Call Summary Best Practices" → `call_summary_best_practices.txt`)
- **Content Formatting**: Includes webui links and page content in uploaded files

#### Confluence Example Output

```
INFO[0000] Syncing files from adapter: confluence
DEBU[0000] Using PARENT PAGE mode - Processing 2 parent pages
DEBU[0000] Fetching files from Confluence parent page: 1234567890
DEBU[0000] Parent page: PoV Guide (Space: 2088140816)
DEBU[0000] Found 4 pages under parent page PoV Guide
DEBU[0000] Fetching files from Confluence parent page: 0987654321
DEBU[0000] Parent page: API Documentation (Space: 2088140816)
DEBU[0000] Found 3 pages under parent page API Documentation
INFO[0001] Successfully synced file: call_summary_best_practices.txt
INFO[0001] Successfully synced file: enabling_features_using_admin_apiconsole.txt
INFO[0001] Successfully synced file: api_endpoints_reference.txt
INFO[0001] Successfully synced file: authentication_guide.txt
```

#### Finding Confluence Page IDs

To find a Confluence page ID:

1. Open the page in your browser
2. Look at the URL: `https://your-domain.atlassian.net/wiki/spaces/SPACEKEY/pages/1234567890/Page+Title`
3. The page ID is `1234567890`

#### Confluence Content Format

Each uploaded file contains:
```
/spaces/SPACEKEY/pages/1234567890/Page+Title

[Page content converted from HTML to plain text]
```

### Multi-Adapter Configuration

You can run both GitHub and Confluence adapters simultaneously:

```yaml
github:
  enabled: true
  token: "your-github-token"
  repositories:
    - "your-org/docs"
  knowledge_id: "docs-knowledge-base"

confluence:
  enabled: true
  base_url: "https://your-domain.atlassian.net"
  username: "your-email@example.com"
  api_key: "your-confluence-api-key"
  parent_page_ids:
    - "1234567890"
    - "0987654321"
  knowledge_id: "confluence-knowledge-base"
```

## Configuration

### Environment Variables

- `OPENWEBUI_BASE_URL`: OpenWebUI instance URL
- `OPENWEBUI_API_KEY`: OpenWebUI API key
- `GITHUB_TOKEN`: GitHub personal access token
- `GITHUB_KNOWLEDGE_ID`: OpenWebUI knowledge ID for GitHub files
- `CONFLUENCE_API_KEY`: Confluence API key
- `CONFLUENCE_BASE_URL`: Confluence instance URL (optional, can be set in config)
- `CONFLUENCE_USERNAME`: Confluence username (optional, can be set in config)
- `CONFLUENCE_KNOWLEDGE_ID`: OpenWebUI knowledge ID for Confluence files
- `STORAGE_PATH`: Local storage path (default: /data)
- `LOG_LEVEL`: Log level (debug, info, warn, error)

### Configuration File

```yaml
log_level: info

schedule:
  interval: 1h  # Sync interval

storage:
  path: /data

openwebui:
  base_url: "http://localhost:8080"
  api_key: ""

# GitHub adapter configuration
github:
  enabled: true
  token: ""  # Set via GITHUB_TOKEN environment variable
  repositories:
    - "owner/repo1"
    - "owner/repo2"
  knowledge_id: ""  # Set via GITHUB_KNOWLEDGE_ID environment variable

# Confluence adapter configuration
confluence:
  enabled: false
  base_url: "https://your-domain.atlassian.net"
  username: "your-email@example.com"
  api_key: ""  # Set via CONFLUENCE_API_KEY environment variable
  spaces:
    - "SPACEKEY1"
    - "SPACEKEY2"
  parent_page_ids: []  # Optional: specific parent page IDs to process sub-pages only
  knowledge_id: ""  # Set via CONFLUENCE_KNOWLEDGE_ID environment variable
  page_limit: 100  # Maximum pages to fetch per space (0 = no limit)
  include_attachments: true  # Whether to download and sync page attachments
```

## Adapter Architecture

The application uses an adapter pattern to support multiple data sources:

```go
type Adapter interface {
    Name() string
    FetchFiles(ctx context.Context) ([]*File, error)
    GetLastSync() time.Time
    SetLastSync(t time.Time)
}
```

### Current Adapters

- **GitHub Adapter**: Syncs files from GitHub repositories
  - Supports multiple repositories
  - File filtering and content hashing
  - Branch-based syncing
- **Confluence Adapter**: Syncs pages from Confluence spaces
  - Space-based syncing (all pages in space)
  - Parent page syncing (specific page and sub-pages)
  - HTML to text conversion
  - Filename sanitization
- **Extensible**: Easy to add new adapters (GitLab, Bitbucket, Notion, etc.)

## File Sync Process

1. **Fetch**: Adapters fetch files from data sources
2. **Hash**: Calculate SHA256 hash of file content
3. **Compare**: Compare with previously synced files
4. **Upload**: Upload new/changed files to OpenWebUI
5. **Associate**: Add files to knowledge base
6. **Index**: Update local file index

## Monitoring

The application provides structured logging and health checks:

```bash
# View logs
kubectl logs -l app=openwebui-content-sync

# Check health
kubectl exec -it <pod-name> -- ps aux | grep connector
```

## Troubleshooting

### Common Issues

1. **Authentication Errors**: Verify API keys and tokens
   - GitHub: Check `GITHUB_TOKEN` environment variable
   - Confluence: Check `CONFLUENCE_API_KEY` and credentials
2. **Network Issues**: Check OpenWebUI connectivity
3. **Storage Issues**: Verify PVC is mounted correctly
4. **Sync Failures**: Check adapter configuration
   - GitHub: Verify repository names and access permissions
   - Confluence: Verify space keys or parent page IDs
5. **Confluence-Specific Issues**:
   - **Empty Results**: Check if parent page ID exists and has sub-pages
   - **Permission Errors**: Verify Confluence API key has read access to spaces
   - **Page Not Found**: Ensure page IDs are correct (check URLs)

### Debug Mode

Enable debug logging:

```yaml
log_level: debug
```

## Development

### Adding New Adapters

1. Implement the `Adapter` interface
2. Add configuration options
3. Register in main application
4. Add tests

### Building

```bash
# Build for local development
go build -o connector .

# Build Docker image
docker build -t openwebui-content-sync .

# Build multi-architecture image
make docker-build-multi

# Build for specific architecture
make docker-build-amd64
make docker-build-arm64
```

## License

This project is licensed under the GNU General Public License v3.0 - see the [LICENSE](LICENSE) file for details.
