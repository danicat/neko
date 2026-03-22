#!/usr/bin/env bash

# Read JSON input from stdin
input=$(cat)

# Extract tool name using jq
tool_name=$(echo "$input" | jq -r '.tool_name')

# Block the built-in replace tool
if [[ "$tool_name" == "replace" ]]; then
    echo "Hook: Blocked built-in 'replace' tool" >&2
    cat <<EOF
{
  "decision": "deny",
  "reason": "Optimization Hook: The built-in 'replace' tool lacks language server integration and structural awareness. You MUST use the 'mcp_neko_edit_file' tool instead to edit file contents within this project to maintain the semantic context pipeline.",
  "systemMessage": "🛑 Blocked built-in replace"
}
EOF
    exit 0
fi

# Default: Allow the tool to proceed
echo '{"decision": "allow"}'
exit 0
