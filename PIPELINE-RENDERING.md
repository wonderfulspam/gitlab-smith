# Pipeline Rendering with GitLab

## The Truth About GitLab Pipeline "Rendering"

GitLab doesn't have an API to "render" pipelines without executing them. To see what jobs would run, you must:

1. **Create a real pipeline** (triggers execution)
2. **Immediately cancel it** (stops jobs from running)
3. **Inspect the created structure**

## Method 1: Using Python Script

```bash
# Start local GitLab
./scripts/gitlab-local-setup.sh

# Wait for GitLab to start (~5 minutes)

# Render your pipeline
python3 scripts/pipeline-interceptor.py your-repo/.gitlab-ci.yml \
  --token <your-token>

# Compare two configs
python3 scripts/pipeline-interceptor.py old.yml --compare new.yml
```

## Method 2: Using gitlab-smith with Real API

```bash
# With local GitLab
gitlab-smith refactor --old old.yml --new new.yml \
  --full-test \
  --gitlab-url http://localhost:8080 \
  --gitlab-token <token>

# With GitLab.com (uses your CI minutes!)
gitlab-smith refactor --old old.yml --new new.yml \
  --full-test \
  --gitlab-url https://gitlab.com \
  --gitlab-token $GITLAB_TOKEN
```

## Method 3: Manual API Calls

```bash
# 1. Create project
PROJECT_ID=$(curl -X POST http://localhost:8080/api/v4/projects \
  -H "PRIVATE-TOKEN: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "test-pipeline"}' | jq -r .id)

# 2. Upload .gitlab-ci.yml
curl -X POST "http://localhost:8080/api/v4/projects/$PROJECT_ID/repository/files/.gitlab-ci.yml" \
  -H "PRIVATE-TOKEN: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "branch": "main",
    "content": "'$(base64 < .gitlab-ci.yml)'",
    "encoding": "base64",
    "commit_message": "Add CI"
  }'

# 3. Create pipeline
PIPELINE_ID=$(curl -X POST "http://localhost:8080/api/v4/projects/$PROJECT_ID/pipeline" \
  -H "PRIVATE-TOKEN: $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"ref": "main"}' | jq -r .id)

# 4. Immediately cancel to prevent execution
curl -X POST "http://localhost:8080/api/v4/projects/$PROJECT_ID/pipelines/$PIPELINE_ID/cancel" \
  -H "PRIVATE-TOKEN: $TOKEN"

# 5. Get the rendered structure
curl "http://localhost:8080/api/v4/projects/$PROJECT_ID/pipelines/$PIPELINE_ID/jobs" \
  -H "PRIVATE-TOKEN: $TOKEN" | jq
```

## What You Get

The pipeline creation shows you:
- Which jobs GitLab creates
- Job dependencies and ordering  
- Stage assignments
- Actual `needs` relationships
- Variables that would be set
- Rules evaluation results

## Important Notes

1. **This creates real pipelines** - even cancelled ones show in history
2. **Uses CI minutes** on GitLab.com (cancelled jobs still count)
3. **Requires project setup** - you need a project with your code
4. **Not instant** - GitLab takes time to process

## Why gitlab-smith Mostly Simulates

Because of these limitations, gitlab-smith primarily uses **local simulation**:
- Parses YAML locally
- Builds dependency graphs
- Simulates execution order
- No GitLab required for most features

The API backend is for when you need **real GitLab validation**.