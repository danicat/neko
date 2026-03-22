#!/usr/bin/env bash

# Read JSON input from stdin
input=$(cat)

# Extract tool name using jq
tool_name=$(echo "$input" | jq -r '.tool_name')

# Block the built-in write_file tool
if [[ "$tool_name" == "write_file" ]]; then
    echo "Hook: Blocked built-in 'write_file' tool" >&2
    cat <<EOF
{
  "decision": "deny",
  "reason": "Optimization Hook: The built-in 'write_file' tool lacks language server integration and structural awareness. You MUST use the 'mcp_neko_create_file' tool instead to create or overwrite file contents within this project to maintain the semantic context pipeline.",
  "systemMessage": "🛑 Blocked built-in write_file"
}
EOF
    exit 0
fi

# Default: Allow the tool to proceed
echo '{"decision": "allow"}'
exit 0
