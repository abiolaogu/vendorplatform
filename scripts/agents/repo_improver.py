def create_github_issue(self, title, body, labels=None):
        """Create a GitHub issue with @claude auto-trigger."""
        if not self.github_token or not self.github_repo:
            print(f"⚠️ Cannot create issue: {title}")
            return

        url = f"https://api.github.com/repos/{self.github_repo}/issues"
        headers = {
            "Authorization": f"token {self.github_token}",
            "Accept": "application/vnd.github.v3+json"
        }

        # FORCE THE TAG AT THE VERY TOP
        # Using '\n\n' ensures GitHub's listener picks up the string clearly.
        formatted_body = f"@claude\n\n{body}"

        data = {
            "title": title,
            "body": formatted_body,
            "labels": labels or []
        }

        try:
            response = requests.post(url, headers=headers, json=data)
            if response.status_code == 201:
                issue_url = response.json()['html_url']
                print(f"✅ Created issue with @claude trigger: {issue_url}")
            else:
                print(f"⚠️ Failed: {response.status_code} - {response.text}")
        except Exception as e:
            print(f"Error: {e}")

    def run(self):
        """Run improvement and ensure @claude is tagged."""
        print("🔧 Starting repository improvement...")
        assessment = self.load_assessment()
        
        # Determine the content based on mode
        if assessment['next_steps'] == 'IMPROVE':
            gaps = self.detect_architectural_gaps()
            if not gaps:
                # If no gaps found, create a general 'optimization' issue to keep Claude active
                self.create_github_issue(
                    title="🔧 Routine Architecture Optimization",
                    body="No critical gaps found. Claude, please perform a general code quality review.",
                    labels=["improvement"]
                )
            for gap in gaps:
                self.create_github_issue(
                    title=f"🔧 Architecture Gap: {gap['title']}",
                    body=f"Type: {gap['issue_type']}\n\n{gap['description']}",
                    labels=["architecture", "improvement"]
                )
        else:
            # This is likely where your script is currently hitting
            print("🚧 Mode: CONTINUE DEVELOPMENT")
            self.create_github_issue(
                title=f"🚧 Progress: {assessment['completion_percentage']}% Complete",
                body=f"Project is in development mode. @claude, please review the PRD and suggest next implementation steps.",
                labels=["development"]
            )
