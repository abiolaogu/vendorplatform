#!/usr/bin/env python3
"""
Repository Improver Agent - vendorplatform Edition
- Calculates progress dynamically
- Injects codebase context into @claude issues
"""

import os
import json
import anthropic
import requests
from pathlib import Path

class RepoImprover:
    def __init__(self, github_token, anthropic_api_key, github_repo):
        self.github_token = github_token
        self.anthropic_api_key = anthropic_api_key
        self.github_repo = github_repo
        self.client = anthropic.Anthropic(api_key=anthropic_api_key)
        self.assessment_file = Path(".factory-assessment.json")

    def load_assessment(self):
        if not self.assessment_file.exists():
            return {
                "goals_achieved": "PARTIAL",
                "prd_exists": False,
                "missing_docs": [],
                "completion_percentage": 50,
                "next_steps": "IMPROVE"
            }
        return json.loads(self.assessment_file.read_text())

    def save_assessment(self, assessment):
        self.assessment_file.write_text(json.dumps(assessment, indent=2))
        print(f"📈 Updated local assessment: {assessment['completion_percentage']}%")

    def scan_codebase(self):
        """Gather code context to help Claude understand the project."""
        repo_root = Path.cwd()
        code_files = []
        extensions = {'.py', '.js', '.ts', '.jsx', '.tsx', '.dart', '.go'}
        ignore_dirs = {'.git', 'node_modules', 'dist', 'build', '.venv'}

        for path in repo_root.rglob('*'):
            if path.is_file() and path.suffix in extensions:
                if any(ignored in path.parts for ignored in ignore_dirs):
                    continue
                try:
                    content = path.read_text(encoding='utf-8')
                    code_files.append(f"--- File: {path.relative_to(repo_root)} ---\n{content[:1000]}")
                except:
                    pass
            if len(code_files) >= 5: break # Limit context size
        return "\n\n".join(code_files)

    def generate_prd(self):
        assessment = self.load_assessment()
        if assessment.get('prd_exists'): return False
        readme_path = Path("README.md")
        readme_content = readme_path.read_text(encoding='utf-8') if readme_path.exists() else ""
        try:
            response = self.client.messages.create(
                model="claude-3-5-sonnet-20241022",
                max_tokens=4000,
                messages=[{"role": "user", "content": f"Generate PRD: {readme_content}"}]
            )
            prd_path = Path("docs/PRD.md")
            prd_path.parent.mkdir(exist_ok=True)
            prd_path.write_text(response.content[0].text)
            return True
        except: return False

    def create_github_issue(self, title, body, labels=None):
        url = f"https://api.github.com/repos/{self.github_repo}/issues"
        headers = {"Authorization": f"token {self.github_token}", "Accept": "application/vnd.github.v3+json"}
        
        # We add 'implement' to tell Claude to write code, not just talk
        formatted_body = f"@claude implement\n\n{body}"
        data = {"title": title, "body": formatted_body, "labels": labels or []}
        requests.post(url, headers=headers, json=data)

    def run(self):
        print("🔧 Starting vendorplatform improvement...")
        assessment = self.load_assessment()
        initial_percentage = assessment.get('completion_percentage', 50)
        work_done_bonus = 0
        
        # Get codebase snapshot
        context = self.scan_codebase()

        if self.generate_prd():
            work_done_bonus += 15
            assessment['prd_exists'] = True

        if assessment['next_steps'] == 'IMPROVE':
            # Handle Improvement Mode
            work_done_bonus += 5
            issue_body = f"## Context\n{context}\n\n## Task\nFix architectural gaps found in the context above."
            self.create_github_issue(title="🔧 Architecture Improvement", body=issue_body, labels=["improvement"])
        else:
            # Handle Development Mode
            work_done_bonus += 5
            issue_body = f"## Project Context\n{context}\n\n## Current Status: {initial_percentage}%\nClaude, analyze the code and implement the next logical feature for vendorplatform."
            self.create_github_issue(
                title=f"🚧 Continue Development: {initial_percentage + work_done_bonus}% Complete",
                body=issue_body,
                labels=["development"]
            )

        assessment['completion_percentage'] = min(initial_percentage + work_done_bonus, 100)
        self.save_assessment(assessment)

if __name__ == "__main__":
    improver = RepoImprover(os.getenv("GITHUB_TOKEN"), os.getenv("ANTHROPIC_API_KEY"), os.getenv("GITHUB_REPOSITORY"))
    improver.run()
