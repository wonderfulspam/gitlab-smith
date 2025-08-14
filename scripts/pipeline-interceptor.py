#!/usr/bin/env python3
"""
Pipeline Interceptor - Creates and analyzes GitLab pipelines without running jobs
"""
import requests
import json
import time
import sys
import yaml
from pathlib import Path

class GitLabPipelineInterceptor:
    def __init__(self, gitlab_url="http://localhost:8080", token=None):
        self.gitlab_url = gitlab_url.rstrip('/')
        self.api_url = f"{self.gitlab_url}/api/v4"
        self.token = token
        self.headers = {"PRIVATE-TOKEN": token} if token else {}
        
    def get_or_create_token(self):
        """Get token via API using username/password"""
        if self.token:
            return self.token
            
        # Try to authenticate and get a token
        session = requests.Session()
        
        # Get CSRF token
        signin_page = session.get(f"{self.gitlab_url}/users/sign_in")
        csrf_token = signin_page.text.split('name="authenticity_token" value="')[1].split('"')[0]
        
        # Login
        login_data = {
            "user[login]": "root",
            "user[password]": "password123",
            "authenticity_token": csrf_token
        }
        session.post(f"{self.gitlab_url}/users/sign_in", data=login_data)
        
        # Create personal access token
        token_page = session.get(f"{self.gitlab_url}/-/profile/personal_access_tokens")
        csrf_token = token_page.text.split('name="authenticity_token" value="')[1].split('"')[0]
        
        token_data = {
            "personal_access_token[name]": f"pipeline-interceptor-{int(time.time())}",
            "personal_access_token[scopes][]": ["api", "read_api", "write_repository"],
            "authenticity_token": csrf_token
        }
        response = session.post(f"{self.gitlab_url}/-/profile/personal_access_tokens", data=token_data)
        
        # Extract token from response
        if "Your new personal access token" in response.text:
            self.token = response.text.split('id="created-personal-access-token" value="')[1].split('"')[0]
            self.headers = {"PRIVATE-TOKEN": self.token}
            print(f"✅ Created token: {self.token}")
            return self.token
        else:
            print("❌ Could not create token automatically. Please create one manually.")
            return None

    def create_test_project(self):
        """Create a test project for pipeline rendering"""
        project_data = {
            "name": f"pipeline-test-{int(time.time())}",
            "visibility": "private",
            "initialize_with_readme": True
        }
        
        response = requests.post(
            f"{self.api_url}/projects",
            headers=self.headers,
            json=project_data
        )
        
        if response.status_code == 201:
            project = response.json()
            print(f"✅ Created project: {project['name']} (ID: {project['id']})")
            return project
        else:
            print(f"❌ Failed to create project: {response.text}")
            return None

    def upload_ci_config(self, project_id, ci_yaml_path):
        """Upload .gitlab-ci.yml to project"""
        with open(ci_yaml_path, 'r') as f:
            content = f.read()
        
        # Create or update .gitlab-ci.yml
        file_data = {
            "branch": "main",
            "content": content,
            "commit_message": "Update CI configuration"
        }
        
        # Try to update first
        response = requests.put(
            f"{self.api_url}/projects/{project_id}/repository/files/.gitlab-ci.yml",
            headers=self.headers,
            json=file_data
        )
        
        if response.status_code == 400:  # File doesn't exist, create it
            file_data["author_email"] = "test@example.com"
            file_data["author_name"] = "Test"
            response = requests.post(
                f"{self.api_url}/projects/{project_id}/repository/files/.gitlab-ci.yml",
                headers=self.headers,
                json=file_data
            )
        
        if response.status_code in [200, 201]:
            print("✅ Uploaded CI configuration")
            return True
        else:
            print(f"❌ Failed to upload CI config: {response.text}")
            return False

    def create_pipeline(self, project_id, ref="main", variables=None):
        """Create a pipeline and immediately cancel it to prevent job execution"""
        # Create pipeline
        pipeline_data = {"ref": ref}
        if variables:
            pipeline_data["variables"] = [{"key": k, "value": v} for k, v in variables.items()]
        
        response = requests.post(
            f"{self.api_url}/projects/{project_id}/pipeline",
            headers=self.headers,
            json=pipeline_data
        )
        
        if response.status_code == 201:
            pipeline = response.json()
            print(f"✅ Created pipeline {pipeline['id']} with status: {pipeline['status']}")
            
            # Immediately cancel to prevent actual execution
            cancel_response = requests.post(
                f"{self.api_url}/projects/{project_id}/pipelines/{pipeline['id']}/cancel",
                headers=self.headers
            )
            
            if cancel_response.status_code == 200:
                print("✅ Cancelled pipeline to prevent job execution")
            
            return pipeline
        else:
            print(f"❌ Failed to create pipeline: {response.text}")
            return None

    def get_pipeline_details(self, project_id, pipeline_id):
        """Get detailed pipeline information"""
        # Get pipeline
        response = requests.get(
            f"{self.api_url}/projects/{project_id}/pipelines/{pipeline_id}",
            headers=self.headers
        )
        pipeline = response.json() if response.status_code == 200 else None
        
        # Get jobs
        response = requests.get(
            f"{self.api_url}/projects/{project_id}/pipelines/{pipeline_id}/jobs",
            headers=self.headers
        )
        jobs = response.json() if response.status_code == 200 else []
        
        # Get pipeline variables
        response = requests.get(
            f"{self.api_url}/projects/{project_id}/pipelines/{pipeline_id}/variables",
            headers=self.headers
        )
        variables = response.json() if response.status_code == 200 else []
        
        return {
            "pipeline": pipeline,
            "jobs": jobs,
            "variables": variables,
            "job_graph": self.build_job_graph(jobs)
        }

    def build_job_graph(self, jobs):
        """Build a dependency graph from jobs"""
        graph = {}
        for job in jobs:
            graph[job['name']] = {
                "stage": job['stage'],
                "status": job['status'],
                "dependencies": job.get('dependencies', []),
                "needs": [n['name'] for n in job.get('needs', [])],
                "when": job.get('when', 'on_success'),
                "allow_failure": job.get('allow_failure', False)
            }
        return graph

    def render_pipeline(self, ci_yaml_path, variables=None):
        """Complete pipeline rendering workflow"""
        print(f"\n🔍 Rendering pipeline for: {ci_yaml_path}")
        
        # Ensure we have a token
        if not self.token:
            self.get_or_create_token()
        
        # Create test project
        project = self.create_test_project()
        if not project:
            return None
        
        # Upload CI config
        if not self.upload_ci_config(project['id'], ci_yaml_path):
            return None
        
        # Create pipeline
        pipeline = self.create_pipeline(project['id'], variables=variables)
        if not pipeline:
            return None
        
        # Wait a bit for GitLab to process
        time.sleep(2)
        
        # Get full details
        details = self.get_pipeline_details(project['id'], pipeline['id'])
        
        # Clean up project (optional)
        # requests.delete(f"{self.api_url}/projects/{project['id']}", headers=self.headers)
        
        return details

    def compare_pipelines(self, yaml1_path, yaml2_path):
        """Compare two pipeline configurations"""
        print("\n📊 Comparing pipeline configurations...")
        
        result1 = self.render_pipeline(yaml1_path)
        result2 = self.render_pipeline(yaml2_path)
        
        if not result1 or not result2:
            return None
        
        comparison = {
            "jobs_added": [],
            "jobs_removed": [],
            "jobs_modified": [],
            "stage_changes": [],
            "dependency_changes": []
        }
        
        jobs1 = {j['name']: j for j in result1['jobs']}
        jobs2 = {j['name']: j for j in result2['jobs']}
        
        # Find differences
        comparison['jobs_removed'] = list(set(jobs1.keys()) - set(jobs2.keys()))
        comparison['jobs_added'] = list(set(jobs2.keys()) - set(jobs1.keys()))
        
        for job_name in set(jobs1.keys()) & set(jobs2.keys()):
            j1, j2 = jobs1[job_name], jobs2[job_name]
            if j1['stage'] != j2['stage']:
                comparison['stage_changes'].append(f"{job_name}: {j1['stage']} -> {j2['stage']}")
            
            deps1 = set(j1.get('dependencies', []))
            deps2 = set(j2.get('dependencies', []))
            if deps1 != deps2:
                comparison['dependency_changes'].append({
                    "job": job_name,
                    "removed": list(deps1 - deps2),
                    "added": list(deps2 - deps1)
                })
        
        return {
            "pipeline1": result1,
            "pipeline2": result2,
            "comparison": comparison
        }

if __name__ == "__main__":
    import argparse
    
    parser = argparse.ArgumentParser(description="GitLab Pipeline Interceptor")
    parser.add_argument("ci_yaml", help="Path to .gitlab-ci.yml")
    parser.add_argument("--compare", help="Path to second .gitlab-ci.yml for comparison")
    parser.add_argument("--gitlab-url", default="http://localhost:8080", help="GitLab URL")
    parser.add_argument("--token", help="GitLab personal access token")
    parser.add_argument("--var", action="append", help="Pipeline variables (KEY=VALUE)")
    
    args = parser.parse_args()
    
    # Parse variables
    variables = {}
    if args.var:
        for var in args.var:
            if '=' in var:
                k, v = var.split('=', 1)
                variables[k] = v
    
    interceptor = GitLabPipelineInterceptor(args.gitlab_url, args.token)
    
    if args.compare:
        result = interceptor.compare_pipelines(args.ci_yaml, args.compare)
    else:
        result = interceptor.render_pipeline(args.ci_yaml, variables)
    
    if result:
        print("\n📋 Pipeline Rendering Result:")
        print(json.dumps(result, indent=2, default=str))