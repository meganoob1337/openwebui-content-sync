# OpenWebUI GitHub Connector - Architecture

## Overview

The OpenWebUI GitHub Connector is a Kubernetes-native application that synchronizes files from GitHub repositories to OpenWebUI knowledge bases using an adapter architecture pattern.

## Architecture Components

### 1. Adapter Layer
- **Interface**: `adapter.Adapter` defines the contract for data source adapters
- **GitHub Adapter**: Implements GitHub API integration for repository file fetching
- **Extensible**: Easy to add new adapters (GitLab, Bitbucket, etc.)

### 2. Sync Manager
- **File Diffing**: Uses SHA256 hashing to detect file changes
- **Local Storage**: Maintains files on persistent volumes
- **OpenWebUI Integration**: Handles file uploads and knowledge base association

### 3. Scheduler
- **Cron-based**: Uses robfig/cron for scheduled synchronization
- **Configurable**: Supports various interval patterns (1h, 2h, etc.)
- **Graceful Shutdown**: Properly handles termination signals

### 4. Configuration Management
- **YAML-based**: Primary configuration via YAML files
- **Environment Override**: Environment variables override file settings
- **Kubernetes Integration**: ConfigMaps and Secrets support

### 5. Health Monitoring
- **HTTP Endpoints**: `/health` and `/ready` for Kubernetes probes
- **Structured Logging**: JSON-formatted logs with configurable levels
- **Error Handling**: Comprehensive error handling and recovery

## Data Flow

```
GitHub Repository → GitHub Adapter → Sync Manager → OpenWebUI API
                                        ↓
                                   Local Storage (PVC)
```

### Detailed Flow:

1. **Scheduler Trigger**: Cron job triggers sync process
2. **Adapter Fetch**: GitHub adapter fetches repository files
3. **File Processing**: Files are filtered (text files only) and hashed
4. **Change Detection**: Compare hashes with previously synced files
5. **Local Storage**: Save files to persistent volume
6. **OpenWebUI Upload**: Upload new/changed files to OpenWebUI
7. **Knowledge Association**: Add files to specified knowledge base
8. **Index Update**: Update local file index for future comparisons

## API Integration

### OpenWebUI APIs Used:
- `POST /api/v1/files/` - Upload files
- `GET /api/v1/knowledge/` - List knowledge sources
- `POST /api/v1/knowledge/{id}/file/add` - Add file to knowledge
- `POST /api/v1/knowledge/{id}/file/remove` - Remove file from knowledge

### GitHub APIs Used:
- `GET /repos/{owner}/{repo}/contents` - Fetch repository contents
- File content retrieval via GitHub's content API

## File Processing

### Supported File Types:
- Markdown (`.md`)
- Text files (`.txt`)
- Code files (`.go`, `.py`, `.js`, `.ts`, etc.)
- Configuration files (`.yaml`, `.json`, `.env`)
- Documentation files (`.rst`, `.adoc`)
- And many more text-based formats

### File Filtering:
- Binary files are automatically excluded
- Large files are handled via GitHub's download URLs
- File size limits can be configured

## Storage Strategy

### Local Storage:
- **Persistent Volume**: Kubernetes PVC for data persistence
- **File Organization**: Files organized by source and path
- **Index Management**: JSON-based file index for change tracking

### File Index Structure:
```json
{
  "source:path": {
    "path": "file.md",
    "hash": "sha256_hash",
    "file_id": "openwebui_file_id",
    "source": "github",
    "synced_at": "2024-01-01T00:00:00Z",
    "modified": "2024-01-01T00:00:00Z"
  }
}
```

## Error Handling

### Retry Logic:
- Network failures are retried with exponential backoff
- GitHub API rate limits are respected
- OpenWebUI API failures are logged and retried

### Recovery:
- Application can recover from crashes
- File index is persisted and restored
- Partial syncs are resumed on restart

## Security Considerations

### Authentication:
- GitHub Personal Access Tokens
- OpenWebUI API Keys
- Kubernetes Secrets for credential management

### Network Security:
- HTTPS for all external API calls
- Configurable timeouts and retry limits
- No sensitive data in logs

## Monitoring and Observability

### Logging:
- Structured JSON logging
- Configurable log levels
- Request/response logging for debugging

### Health Checks:
- Liveness probe: `/health`
- Readiness probe: `/ready`
- Kubernetes-native health monitoring

### Metrics:
- Sync operation counts
- File processing statistics
- Error rates and types

## Scalability

### Horizontal Scaling:
- Stateless design allows multiple replicas
- File index can be shared via external storage
- Adapter instances can be distributed

### Vertical Scaling:
- Configurable resource limits
- Memory usage scales with repository size
- CPU usage scales with sync frequency

## Deployment

### Kubernetes Manifests:
- Deployment with health checks
- PersistentVolumeClaim for storage
- ConfigMap for configuration
- Secrets for credentials

### Docker:
- Multi-stage build for minimal image size
- Alpine Linux base image
- Non-root user for security

## Configuration

### Environment Variables:
- `OPENWEBUI_BASE_URL`: OpenWebUI instance URL
- `OPENWEBUI_API_KEY`: API authentication
- `GITHUB_TOKEN`: GitHub authentication
- `GITHUB_KNOWLEDGE_ID`: Target knowledge base
- `STORAGE_PATH`: Local storage path
- `LOG_LEVEL`: Logging verbosity

### Configuration File:
```yaml
log_level: info
schedule:
  interval: 1h
storage:
  path: /data
openwebui:
  base_url: "http://localhost:8080"
  api_key: ""
github:
  enabled: true
  token: ""
  repositories:
    - "owner/repo1"
    - "owner/repo2"
  knowledge_id: ""
```

## Future Enhancements

### Planned Features:
- Additional adapters (GitLab, Bitbucket)
- Webhook-based real-time sync
- File content transformation
- Advanced filtering rules
- Sync status dashboard
- Metrics and alerting

### Extensibility:
- Plugin architecture for custom adapters
- Custom file processors
- Integration with CI/CD pipelines
- Multi-tenant support
