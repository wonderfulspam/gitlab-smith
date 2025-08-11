# Continue Implementation

Automatically checks the implementation state and works on the next priority item, then updates the state.

## Usage

```bash
/continue
```

## Implementation

This command will:
1. Read the current implementation-state.json 
2. Identify the next priority item from the "next_steps" list
3. Work on that specific enhancement/fix
4. Run tests to validate the changes
5. Update the implementation-state.json with progress

Each run will make incremental progress on the highest priority incomplete item.

## Command

```bash
#!/bin/bash

# Check if we're in the right directory
if [ ! -f "implementation-state.json" ]; then
    echo "Error: implementation-state.json not found. Run from project root."
    exit 1
fi

# Let Claude handle the implementation continuation
claude --prompt "Check implementation-state.json and work on the next priority item from next_steps. Focus on the highest priority incomplete task from the critical_fixes_needed list. After completing work, run tests with 'go test ./...' and update implementation-state.json with the progress made and the next steps."
```
