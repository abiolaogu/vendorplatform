#!/usr/bin/env python3
"""
Repository Assessor Agent
Analyzes a repository to determine:
1. If project goals have been achieved
2. If PRD exists
3. What important documentation is missing
4. Overall completion status
"""

import os
import json
import anthropic
from pathlib import Path


class RepoAssessor:
    def __init__(self, github_token, anthropic_api_key):
        self.github_token = github_token
        self.anthropic_api_key = anthropic_api_key
        self.client = anthropic.Anthropic(api_key=anthropic_api_key)

    def scan_repo_structure(self):
        """Generate a tree structure of the repository."""
        repo_root = Path.cwd()

        # Ignore common directories
        ignore_dirs = {'.git', 'node_modules', '__pycache__', '.venv', 'venv',
                      'dist', 'build', '.pytest_cache', 'coverage'}

        structure = []

        def walk_dir(path, prefix="", max_depth=4, current_depth=0):
            if current_depth > max_depth:
                return

            try:
                items = sorted(path.iterdir(), key=lambda x: (not x.is_dir(), x.name))
                for item in items:
                    if item.name in ignore_dirs or item.name.startswith('.'):
                        continue

                    rel_path = item.relative_to(repo_root)
                    if item.is_dir():
                        structure.append(f"{prefix}📁 {item.name}/")
                        walk_dir(item, prefix + "  ", max_depth, current_depth + 1)
                    else:
                        structure.append(f"{prefix}📄 {item.name}")
            except PermissionError:
                pass

        walk_dir(repo_root)
        return "\n".join(structure)

    def collect_key_files(self):
        """Collect contents of key documentation and configuration files."""
        key_files = [
            "README.md",
            "CLAUDE.md",
            "PRD.md",
            "docs/PRD.md",
            "PRODUCT_REQUIREMENTS.md",
            "package.json",
            "requirements.txt",
            "pubspec.yaml",
            "Cargo.toml",
            "go.mod",
            "pom.xml",
            "build.gradle",
            ".github/workflows/*.yml"
        ]

        collected = {}
        repo_root = Path.cwd()

        for pattern in key_files:
            if "*" in pattern:
                # Handle glob patterns
                from glob import glob
                matches = glob(pattern, recursive=True)
                for match in matches[:5]:  # Limit to first 5 workflow files
                    path = Path(match)
                    if path.exists() and path.is_file():
                        try:
                            content = path.read_text(encoding='utf-8')
                            collected[str(path)] = content[:2000]  # First 2000 chars
                        except Exception:
                            pass
            else:
                path = repo_root / pattern
                if path.exists() and path.is_file():
                    try:
                        content = path.read_text(encoding='utf-8')
                        collected[pattern] = content[:3000]  # First 3000 chars
                    except Exception:
                        pass

        return collected

    def assess_completion_status(self):
        """Use Claude to assess repository completion status."""
        structure = self.scan_repo_structure()
        key_files = self.collect_key_files()

        # Build context
        context = f"""# Repository Structure
{structure}

# Key Files Content
"""
        for filename, content in key_files.items():
            context += f"\n## {filename}\n```\n{content}\n```\n"

        # Prompt Claude for assessment
        prompt = """You are a senior software architect analyzing a repository.

Your task: Analyze this repository and determine:
1. **Goals Achieved**: Are the project's stated goals/requirements completed? (YES/NO/PARTIAL)
2. **PRD Exists**: Is there a Product Requirements Document? (YES/NO)
3. **Missing Docs**: What critical documentation is missing? (list)
4. **Completion Percentage**: Estimate completion (0-100%)
5. **Next Steps**: What should be done next? (IMPROVE/CONTINUE_DEVELOPMENT)

Repository Information:
{REPO_INFO}

Respond ONLY with this JSON format:
{
  "goals_achieved": "YES|NO|PARTIAL",
  "prd_exists": true|false,
  "missing_docs": ["doc1", "doc2", ...],
  "completion_percentage": 0-100,
  "next_steps": "IMPROVE|CONTINUE_DEVELOPMENT",
  "reasoning": "Brief explanation"
}
"""

        prompt = prompt.replace("{REPO_INFO}", context)

        try:
            response = self.client.messages.create(
                model="claude-3-5-sonnet-20241022",
                max_tokens=2000,
                messages=[{
                    "role": "user",
                    "content": prompt
                }]
            )

            response_text = response.content[0].text

            # Extract JSON from response
            import re
            json_match = re.search(r'\{[^}]+\}', response_text, re.DOTALL)
            if json_match:
                assessment = json.loads(json_match.group(0))
                return assessment
            else:
                # Fallback
                return {
                    "goals_achieved": "PARTIAL",
                    "prd_exists": False,
                    "missing_docs": ["PRD.md", "API_DOCS.md", "DEPLOYMENT.md"],
                    "completion_percentage": 50,
                    "next_steps": "CONTINUE_DEVELOPMENT",
                    "reasoning": "Unable to parse repository properly"
                }
        except Exception as e:
            print(f"Error during assessment: {e}")
            return {
                "goals_achieved": "PARTIAL",
                "prd_exists": False,
                "missing_docs": ["PRD.md"],
                "completion_percentage": 50,
                "next_steps": "CONTINUE_DEVELOPMENT",
                "reasoning": f"Error: {str(e)}"
            }

    def run(self):
        """Run the assessment and return results."""
        print("🔍 Starting repository assessment...")

        assessment = self.assess_completion_status()

        print("\n📊 Assessment Results:")
        print(f"  Goals Achieved: {assessment['goals_achieved']}")
        print(f"  PRD Exists: {assessment['prd_exists']}")
        print(f"  Completion: {assessment['completion_percentage']}%")
        print(f"  Next Steps: {assessment['next_steps']}")
        print(f"  Reasoning: {assessment['reasoning']}")

        # Save to file for next agent
        output_path = Path(".factory-assessment.json")
        output_path.write_text(json.dumps(assessment, indent=2))

        print(f"\n✅ Assessment saved to {output_path}")

        return assessment


if __name__ == "__main__":
    github_token = os.getenv("GITHUB_TOKEN")
    anthropic_api_key = os.getenv("ANTHROPIC_API_KEY")

    if not anthropic_api_key:
        print("❌ Error: ANTHROPIC_API_KEY environment variable not set")
        exit(1)

    assessor = RepoAssessor(github_token, anthropic_api_key)
    result = assessor.run()

    # Exit with status code based on next steps
    if result['next_steps'] == 'IMPROVE':
        exit(0)  # Ready for improvement
    else:
        exit(10)  # Needs continued development
