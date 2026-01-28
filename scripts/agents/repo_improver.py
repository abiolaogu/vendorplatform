#!/usr/bin/env python3
"""
Repository Improver Agent
Based on assessment results, this agent:
1. Fixes architectural gaps (XP/DDD/TDD compliance)
2. Detects and fixes bugs
3. Generates missing PRD
4. Generates missing documentation
5. Creates GitHub issues with proposed improvements
"""

import os
import json
import anthropic
from pathlib import Path
from datetime import datetime


class RepoImprover:
    def __init__(self, github_token, anthropic_api_key, github_repo):
        self.github_token = github_token
        self.anthropic_api_key = anthropic_api_key
        self.github_repo = github_repo
        self.client = anthropic.Anthropic(api_key=anthropic_api_key)

    def load_assessment(self):
        """Load the assessment from repo_assessor."""
        assessment_path = Path(".factory-assessment.json")
        if not assessment_path.exists():
            print("⚠️ Warning: No assessment file found. Running in blind mode.")
            return {
                "goals_achieved": "PARTIAL",
                "prd_exists": False,
                "missing_docs": [],
                "completion_percentage": 50,
                "next_steps": "IMPROVE"
            }

        return json.loads(assessment_path.read_text())

    def scan_codebase(self):
        """Scan codebase for analysis."""
        repo_root = Path.cwd()
        code_files = []

        # Find code files
        extensions = {'.py', '.js', '.ts', '.jsx', '.tsx', '.go', '.rs', '.java', '.dart'}
        ignore_dirs = {'.git', 'node_modules', '__pycache__', '.venv', 'venv', 'dist', 'build'}

        for path in repo_root.rglob('*'):
            if path.is_file() and path.suffix in extensions:
                # Skip if in ignored directory
                if any(ignored in path.parts for ignored in ignore_dirs):
                    continue
                try:
                    content = path.read_text(encoding='utf-8')
                    code_files.append({
                        'path': str(path.relative_to(repo_root)),
                        'content': content[:1500]  # First 1500 chars
                    })
                    if len(code_files) >= 10:  # Limit to 10 files for analysis
                        break
                except Exception:
                    pass

        return code_files

    def detect_architectural_gaps(self):
        """Use Claude to detect architectural issues."""
        code_files = self.scan_codebase()

        if not code_files:
            return []

        code_summary = "\n\n".join([
            f"## {f['path']}\n```\n{f['content']}\n```"
            for f in code_files
        ])

        prompt = f"""You are a Lead Software Architect performing a Holy Trinity (XP/DDD/TDD) compliance audit.

Analyze this codebase sample and identify ONE critical architectural gap or code smell:
- Missing tests (TDD violation)
- Business logic in controllers/views (DDD violation)
- Hardcoded secrets or configuration
- Missing error handling
- Performance issues
- Security vulnerabilities

Code Sample:
{code_summary}

Respond ONLY with this JSON format:
{{
  "issue_found": true|false,
  "issue_type": "MISSING_TESTS|LOGIC_IN_CONTROLLERS|HARDCODED_SECRETS|ERROR_HANDLING|PERFORMANCE|SECURITY|OTHER",
  "title": "Brief title (max 60 chars)",
  "description": "Detailed description with file references",
  "severity": "CRITICAL|HIGH|MEDIUM|LOW",
  "fix_strategy": "Step-by-step fix approach"
}}
"""

        try:
            response = self.client.messages.create(
                model="claude-3-5-sonnet-20241022",
                max_tokens=2000,
                messages=[{"role": "user", "content": prompt}]
            )

            response_text = response.content[0].text

            # Extract JSON
            import re
            json_match = re.search(r'\{[^}]+\}', response_text, re.DOTALL)
            if json_match:
                gap = json.loads(json_match.group(0))
                if gap.get('issue_found'):
                    return [gap]

            return []
        except Exception as e:
            print(f"Error detecting gaps: {e}")
            return []

    def generate_prd(self):
        """Generate a PRD based on repository analysis."""
        assessment = self.load_assessment()

        if assessment.get('prd_exists'):
            print("✅ PRD already exists, skipping generation")
            return None

        # Analyze README and code to generate PRD
        readme_path = Path("README.md")
        readme_content = ""
        if readme_path.exists():
            readme_content = readme_path.read_text(encoding='utf-8')

        prompt = f"""You are a Product Manager. Generate a comprehensive Product Requirements Document (PRD) for this project.

README Content:
{readme_content}

Generate a PRD with these sections:
1. Product Vision
2. Goals & Objectives
3. User Stories
4. Functional Requirements
5. Non-Functional Requirements
6. Technical Architecture
7. Success Metrics

Format as Markdown. Be concise but comprehensive.
"""

        try:
            response = self.client.messages.create(
                model="claude-3-5-sonnet-20241022",
                max_tokens=4000,
                messages=[{"role": "user", "content": prompt}]
            )

            prd_content = response.content[0].text

            # Save PRD
            prd_path = Path("docs/PRD.md")
            prd_path.parent.mkdir(exist_ok=True)
            prd_path.write_text(prd_content)

            print(f"✅ Generated PRD at {prd_path}")
            return str(prd_path)
        except Exception as e:
            print(f"Error generating PRD: {e}")
            return None

    def generate_missing_docs(self):
        """Generate missing documentation."""
        assessment = self.load_assessment()
        missing_docs = assessment.get('missing_docs', [])

        generated = []

        for doc_name in missing_docs:
            if doc_name == "PRD.md":
                continue  # Handled by generate_prd()

            doc_path = Path(f"docs/{doc_name}")

            # Generate appropriate content based on doc type
            if "API" in doc_name.upper():
                content = self._generate_api_docs()
            elif "DEPLOYMENT" in doc_name.upper():
                content = self._generate_deployment_docs()
            elif "CONTRIBUTING" in doc_name.upper():
                content = self._generate_contributing_docs()
            else:
                content = f"# {doc_name}\n\nDocumentation placeholder - to be completed.\n"

            if content:
                doc_path.parent.mkdir(exist_ok=True)
                doc_path.write_text(content)
                generated.append(str(doc_path))
                print(f"✅ Generated {doc_path}")

        return generated

    def _generate_api_docs(self):
        """Generate API documentation."""
        return """# API Documentation

## Overview
This document describes the API endpoints and their usage.

## Endpoints

### GET /api/health
Health check endpoint.

**Response:**
```json
{
  "status": "ok",
  "timestamp": "2026-01-28T10:00:00Z"
}
```

## Authentication
Describe authentication mechanism here.

## Rate Limiting
Describe rate limiting policy here.
"""

    def _generate_deployment_docs(self):
        """Generate deployment documentation."""
        return """# Deployment Guide

## Prerequisites
- List required software/services
- Environment variables
- Access credentials

## Deployment Steps

### Development Environment
```bash
# Installation steps
npm install  # or pip install -r requirements.txt
npm run dev
```

### Production Environment
```bash
# Build steps
npm run build
npm start
```

## Environment Variables
- `PORT` - Server port (default: 3000)
- `DATABASE_URL` - Database connection string
- Add other variables...

## Troubleshooting
Common deployment issues and solutions.
"""

    def _generate_contributing_docs(self):
        """Generate contributing guidelines."""
        return """# Contributing Guidelines

## Getting Started
1. Fork the repository
2. Clone your fork
3. Create a feature branch
4. Make your changes
5. Submit a pull request

## Code Standards
- Follow existing code style
- Write tests for new features
- Update documentation

## Pull Request Process
1. Ensure all tests pass
2. Update README if needed
3. Request review from maintainers
"""

    def create_github_issue(self, title, body, labels=None):
        """Create a GitHub issue."""
        import requests

        if not self.github_token or not self.github_repo:
            print(f"⚠️ Cannot create issue: {title}")
            return

        url = f"https://api.github.com/repos/{self.github_repo}/issues"
        headers = {
            "Authorization": f"token {self.github_token}",
            "Accept": "application/vnd.github.v3+json"
        }

        data = {
            "title": title,
            "body": body,
            "labels": labels or []
        }

        try:
            response = requests.post(url, headers=headers, json=data)
            if response.status_code == 201:
                issue_url = response.json()['html_url']
                print(f"✅ Created issue: {issue_url}")
            else:
                print(f"⚠️ Failed to create issue: {response.status_code}")
        except Exception as e:
            print(f"Error creating issue: {e}")

    def run(self):
        """Run the improvement process."""
        print("🔧 Starting repository improvement...")

        assessment = self.load_assessment()

        if assessment['next_steps'] == 'IMPROVE':
            print("📈 Mode: IMPROVEMENT (Goals achieved, optimizing)")

            # Detect architectural gaps
            gaps = self.detect_architectural_gaps()
            for gap in gaps:
                self.create_github_issue(
                    title=f"🔧 Architecture Gap: {gap['title']}",
                    body=f"""## Issue Type
{gap['issue_type']}

## Description
{gap['description']}

## Severity
{gap['severity']}

## Fix Strategy
{gap['fix_strategy']}

---
_Generated by Factory Repository Improver_
""",
                    labels=["architecture", "improvement"]
                )

            # Generate PRD if missing
            self.generate_prd()

            # Generate missing docs
            self.generate_missing_docs()

        else:
            print("🚧 Mode: CONTINUE DEVELOPMENT (Goals not achieved)")

            # Generate PRD first (critical for development)
            self.generate_prd()

            # Generate missing docs
            self.generate_missing_docs()

            # Create development tracking issue
            self.create_github_issue(
                title=f"🚧 Continue Development: {assessment['completion_percentage']}% Complete",
                body=f"""## Project Status
**Completion:** {assessment['completion_percentage']}%
**Goals Achieved:** {assessment['goals_achieved']}

## Reasoning
{assessment['reasoning']}

## Next Actions
1. Review generated PRD (docs/PRD.md)
2. Continue implementing core features
3. Add tests for existing functionality
4. Update documentation as features are completed

---
_Generated by Factory Repository Improver_
""",
                labels=["development", "in-progress"]
            )

        print("\n✅ Repository improvement complete!")


if __name__ == "__main__":
    github_token = os.getenv("GITHUB_TOKEN")
    anthropic_api_key = os.getenv("ANTHROPIC_API_KEY")
    github_repo = os.getenv("GITHUB_REPOSITORY")

    if not anthropic_api_key:
        print("❌ Error: ANTHROPIC_API_KEY environment variable not set")
        exit(1)

    improver = RepoImprover(github_token, anthropic_api_key, github_repo)
    improver.run()
