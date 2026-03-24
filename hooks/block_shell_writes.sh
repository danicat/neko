#!/usr/bin/env bash

# Read JSON input from stdin
input=$(cat)

# Extract tool name using jq
tool_name=$(echo "$input" | jq -r '.tool_name')

# Block raw shell writes
if [[ "$tool_name" == "run_shell_command" ]]; then
  command=$(echo "$input" | jq -r '.tool_input.command // ""')

  # List of common commands or patterns used to write file contents
  write_patterns=("> " ">> " "tee " "sed -i " "awk -i ")
  # Additional commands that can be used to bypass via scripting languages
  script_patterns=()
  
  blocked=false
  matched_cmd=""

  # Check standard write patterns across the entire command
  for pattern in "${write_patterns[@]}"; do
    if [[ "$command" == *"$pattern"* ]]; then
        blocked=true
        matched_cmd="$pattern"
        break
    fi
  done
  
  # Check for pipeline bypasses using scripting languages
  if [[ "$blocked" == false ]]; then
    IFS='|' read -ra parts <<< "$command"
    if [ ${#parts[@]} -gt 1 ]; then
      last_part="${parts[${#parts[@]}-1]}"
      last_part=$(echo "$last_part" | xargs)
      for sp in "${script_patterns[@]}"; do
        if [[ "$last_part" == "$sp"* ]]; then
           blocked=true
           matched_cmd="pipe to $sp"
           break
        fi
      done
    fi
  fi

  # Explicitly check for echo to a file redirect if not caught above
  if [[ "$blocked" == false ]] && [[ "$command" == *"echo "* ]] && [[ "$command" == *">"* ]]; then
      blocked=true
      matched_cmd="echo ... > "
  fi

  # Exceptions (e.g., if we want to allow standard output redirection to /dev/null)
  if [[ "$command" == *"> /dev/null"* ]] || [[ "$command" == *">/dev/null"* ]]; then
      blocked=false
  fi

  if [[ "$blocked" == true ]]; then
    # Log to stderr for debugging
    echo "Hook: Blocked shell command '$command' attempting to use '$matched_cmd' to write/edit files" >&2

    # Return a JSON denial instructing the agent to use mcp_neko_create_file or edit_file
    cat <<EOF
{
  "decision": "deny",
  "reason": "Optimization Hook: Using standard shell commands (like '$matched_cmd') to write or edit files is inefficient and breaks semantic awareness. You MUST use the 'mcp_neko_create_file' or 'mcp_neko_edit_file' tools instead to modify file contents within this project.",
  "systemMessage": "🛑 Blocked raw file write/edit via shell"
}
EOF
    exit 0
  fi
fi

# Default: Allow the tool to proceed
echo '{"decision": "allow"}'
exit 0
