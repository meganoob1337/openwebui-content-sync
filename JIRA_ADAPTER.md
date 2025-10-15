# Jira Adapter

The Jira adapter allows you to sync content from Atlassian Jira projects into OpenWebUI knowledge bases. This adapter uses the Jira REST API to fetch issues and comments from specified Jira projects and uploads them to OpenWebUI.

## API Compatibility

This adapter uses Jira REST API v3, which provides:
- Modern cursor-based pagination
- Improved performance and reliability
- Better support for large projects
- Enhanced metadata and content structure

## Features

- **Issue Content Sync**: Fetches all issues from specified Jira projects using Jira API v3
- **HTML to Markdown Conversion**: Converts Jira's HTML content to markdown format
- **Comment Support**: downloads and syncs issue comments
- **Multi-Project Support**: Can sync from multiple Jira projects
- **Cursor-based Pagination**: Uses modern cursor-based pagination for efficient data retrieval

## Configuration

### YAML Configuration

Add the following to your `config.yaml`:

```yaml
jira:
  enabled: true
  base_url: "https://your-domain.atlassian.net"
  username: "your-email@example.com"
  api_key: "your-jira-api-key"
  project_mappings:
    - project_key: "PROJ"
      knowledge_id: "your-knowledge-base-id"
    - project_key: "ANOTHER"
      knowledge_id: "another-knowledge-base-id"
  page_limit: 100
```

### Environment Variables

Only the API key can be configured via environment variable (for security):

```bash
JIRA_API_KEY="your-jira-api-key"
```

All other configuration should be done in the `config.yaml` file.

### Kubernetes Configuration

#### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: connector-config
data:
  config.yaml: |
    jira:
      enabled: true
      base_url: "https://your-domain.atlassian.net"
      username: "your-email@example.com"
      project_mappings:
        - project_key: "PROJ"
          knowledge_id: "your-knowledge-base-id"
        - project_key: "ANOTHER"
          knowledge_id: "another-knowledge-base-id"
      page_limit: 100
```

## Authentication

The Jira adapter uses Basic Authentication with your Jira username and API key. To get an API key:

1. Go to [Atlassian Account Settings](https://id.atlassian.com/manage-profile/security/api-tokens)
2. Click "Create API token"
3. Give it a label and copy the generated token
4. Use your email address as the username and the token as the API key

## Configuration Parameters

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `enabled` | boolean | No | `false` | Enable the Jira adapter |
| `base_url` | string | Yes | - | Your Jira instance URL (e.g., `https://your-domain.atlassian.net`) |
| `username` | string | Yes | - | Your Jira username (usually your email) |
| `api_key` | string | Yes | - | Your Jira API key |
| `project_mappings` | array | Yes | - | List of Jira project keys and their corresponding OpenWebUI knowledge base IDs |
| `page_limit` | integer | No | `100` | Maximum number of issues to fetch per project |

## File Processing

### Issue Content

- Jira issues are converted from HTML to markdown format
- Issues are saved as `.md` files with sanitized filenames
- File paths follow the pattern: `{issue-id}.md`

### Issue Metadata

Each issue file includes:
- Issue key
- Reporter name
- Issue type
- Status
- Resolution status

### Comments

- Comments are fetched and included in the markdown file
- Each comment includes the author's display name and timestamp
- Comments are formatted in markdown

## Error Handling

- **Authentication Errors**: Invalid credentials will cause the adapter to fail initialization
- **API Errors**: HTTP errors from Jira API are logged and may cause individual issue processing to fail
- **File Processing Errors**: Individual file processing errors are logged but don't stop the overall sync
- **Network Errors**: Connection timeouts and network issues are handled gracefully

## Limitations

1. **API Rate Limits**: Jira has API rate limits that may affect sync performance
2. **Large Projects**: Very large projects with many issues may take significant time to sync
3. **HTML Conversion**: The HTML to markdown conversion is basic and may not preserve all formatting
4. **Comment Limitations**: Comments are limited to the Jira API's available fields

## Troubleshooting

### Common Issues

1. **Authentication Failed**
   - Verify your username and API key are correct
   - Ensure your API key has the necessary permissions

2. **Project Not Found**
   - Check that the project key is correct
   - Verify you have access to the project

3. **No Content Synced**
   - Check that the project contains issues
   - Verify the `page_limit` setting is appropriate
   - Check logs for API errors

### Debug Mode

Enable debug logging to see detailed information about the sync process:

```yaml
log_level: debug
```

## Example Usage

### Basic Configuration

```yaml
jira:
  enabled: true
  base_url: "https://mycompany.atlassian.net"
  username: "john.doe@mycompany.com"
  api_key: "ATATT3xFfGF0..."
  project_mappings:
    - project_key: "DOCS"
      knowledge_id: "fbc18bc4-72c1-40f0-84b1-52055368c583"
    - project_key: "PROJ"
      knowledge_id: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
  page_limit: 100
```

### Advanced Configuration

```yaml
jira:
  enabled: true
  base_url: "https://mycompany.atlassian.net"
  username: "john.doe@mycompany.com"
  api_key: "ATATT3xFfGF0..."
  project_mappings:
    - project_key: "DOCS"
      knowledge_id: "fbc18bc4-72c1-40f0-84b1-52055368c583"
    - project_key: "PROJ"
      knowledge_id: "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
    - project_key: "OPS"
      knowledge_id: "98765432-10fe-dcba-0987-6543210fedcb"
  page_limit: 500
```

This configuration will sync up to 500 issues from each of the three specified projects.
