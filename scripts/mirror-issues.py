#!/usr/bin/env python3

import json
import os
import re
import sys
import urllib.error
import urllib.request

MARKER = "flick-mirror:gitlab-issue:{}"
MARKER_RE = re.compile(r"flick-mirror:gitlab-issue:(\d+)")


def die(msg):
    print(f"error: {msg}", file=sys.stderr)
    sys.exit(1)


def env(name):
    value = os.environ.get(name, "").strip()
    if not value:
        die(f"missing required environment variable {name}")
    return value


def github_repo_from_mirror(url):
    repo = re.sub(r"^.*github\.com[:/]", "", url)
    return re.sub(r"\.git$", "", repo).strip("/")


def request(method, url, headers, data=None):
    body = json.dumps(data).encode() if data is not None else None
    req = urllib.request.Request(url, data=body, method=method, headers=headers)
    try:
        resp = urllib.request.urlopen(req)
        raw = resp.read()
        return resp.status, dict(resp.headers), json.loads(raw) if raw else None
    except urllib.error.HTTPError as exc:
        raw = exc.read()
        try:
            payload = json.loads(raw)
        except Exception:
            payload = raw.decode(errors="replace")
        return exc.code, dict(exc.headers), payload


class GitLab:
    def __init__(self, base, project_id, token):
        self.base = base.rstrip("/")
        self.project_id = project_id
        self.headers = {"PRIVATE-TOKEN": token, "Accept": "application/json"}

    def issues(self):
        out, page = [], 1
        while True:
            url = (
                f"{self.base}/api/v4/projects/{self.project_id}/issues"
                f"?state=all&scope=all&per_page=100&page={page}"
            )
            status, _, payload = request("GET", url, self.headers)
            if status != 200:
                die(f"GitLab issues {status}: {payload}")
            if not payload:
                break
            out.extend(payload)
            if len(payload) < 100:
                break
            page += 1
        return out


class GitHub:
    def __init__(self, repo, token):
        self.repo = repo
        self.headers = {
            "Authorization": f"Bearer {token}",
            "Accept": "application/vnd.github+json",
            "X-GitHub-Api-Version": "2022-11-28",
        }

    def issues(self):
        out = []
        url = f"https://api.github.com/repos/{self.repo}/issues?state=all&per_page=100"
        while url:
            status, headers, payload = request("GET", url, self.headers)
            if status != 200:
                die(f"GitHub issues {status}: {payload}")
            out.extend(payload)
            url = None
            for part in headers.get("Link", "").split(","):
                if 'rel="next"' in part:
                    url = part[part.find("<") + 1 : part.find(">")]
        return out

    def create_issue(self, data):
        status, _, payload = request(
            "POST", f"https://api.github.com/repos/{self.repo}/issues", self.headers, data
        )
        if status not in (200, 201):
            die(f"GitHub create issue {status}: {payload}")
        return payload

    def update_issue(self, number, data):
        status, _, payload = request(
            "PATCH",
            f"https://api.github.com/repos/{self.repo}/issues/{number}",
            self.headers,
            data,
        )
        if status != 200:
            die(f"GitHub update issue #{number} {status}: {payload}")
        return payload


def build_body(issue):
    description = (issue.get("description") or "").strip() or "_No description provided._"
    author = (issue.get("author") or {}).get("name") or "unknown"
    iid = issue["iid"]
    return (
        f"{description}\n\n"
        "---\n"
        f"*Mirrored from GitLab issue (opened by {author}).*\n"
        f"<!-- {MARKER.format(iid)} -->\n"
    )


def main():
    publish_label = os.environ.get("MIRROR_ISSUE_LABEL", "").strip() or "public"
    gitlab = GitLab(env("GITLAB_URL"), env("GITLAB_PROJECT_ID"), env("GITLAB_TOKEN"))
    github = GitHub(github_repo_from_mirror(env("MIRROR_URL")), env("GITHUB_TOKEN"))

    all_issues = gitlab.issues()
    public_issues = [i for i in all_issues if publish_label in i.get("labels", [])]
    print(
        f"Found {len(all_issues)} GitLab issue(s), "
        f"{len(public_issues)} labelled '{publish_label}'"
    )

    index = {}
    for issue in github.issues():
        if "pull_request" in issue:  # the issues endpoint also returns PRs
            continue
        match = MARKER_RE.search(issue.get("body") or "")
        if match:
            index[int(match.group(1))] = issue

    created = updated = unchanged = retracted = 0
    public_iids = set()
    for issue in public_issues:
        iid = issue["iid"]
        public_iids.add(iid)
        title = issue["title"]
        body = build_body(issue)
        # The publish label is an internal marker; don't carry it to GitHub.
        labels = sorted(l for l in issue.get("labels", []) if l != publish_label)
        state = "closed" if issue.get("state") == "closed" else "open"

        existing = index.get(iid)
        if existing is None:
            payload = github.create_issue({"title": title, "body": body, "labels": labels})
            number = payload["number"]
            if state == "closed":
                github.update_issue(number, {"state": "closed"})
            created += 1
            print(f"  + created GitHub #{number} from GitLab #{iid}: {title}")
            continue

        number = existing["number"]
        current_labels = sorted(label["name"] for label in existing.get("labels", []))
        changed = (
            existing.get("title") != title
            or (existing.get("body") or "") != body
            or existing.get("state") != state
            or current_labels != labels
        )
        if changed:
            github.update_issue(
                number, {"title": title, "body": body, "state": state, "labels": labels}
            )
            updated += 1
            print(f"  ~ updated GitHub #{number} from GitLab #{iid}: {title}")
        else:
            unchanged += 1

    for iid, existing in index.items():
        if iid not in public_iids and existing.get("state") == "open":
            github.update_issue(existing["number"], {"state": "closed"})
            retracted += 1
            print(f"  - closed GitHub #{existing['number']} (GitLab #{iid} no longer public)")

    print(
        f"Done: {created} created, {updated} updated, "
        f"{unchanged} unchanged, {retracted} retracted"
    )

if __name__ == "__main__":
    main()
