#!/usr/bin/env python3

import json
import mimetypes
import os
import re
import sys
import urllib.error
import urllib.parse
import urllib.request

SKIP_ASSETS = {"flick-deb-staging"}


def die(msg):
    print(f"error: {msg}", file=sys.stderr)
    sys.exit(1)


def env(name, default=None, required=False):
    value = os.environ.get(name, "").strip()
    if not value:
        if required:
            die(f"missing required environment variable {name}")
        return default
    return value


def github_repo_from_mirror(url):
    repo = re.sub(r"^.*github\.com[:/]", "", url)
    return re.sub(r"\.git$", "", repo).strip("/")


def request(method, url, headers, data=None, raw=None):
    if raw is not None:
        body = raw
    elif data is not None:
        body = json.dumps(data).encode()
        headers = {**headers, "Content-Type": "application/json"}
    else:
        body = None
    req = urllib.request.Request(url, data=body, method=method, headers=headers)
    try:
        resp = urllib.request.urlopen(req)
        payload = resp.read()
        return resp.status, json.loads(payload) if payload else None
    except urllib.error.HTTPError as exc:
        payload = exc.read()
        try:
            payload = json.loads(payload)
        except Exception:
            payload = payload.decode(errors="replace")
        return exc.code, payload


def gitlab_notes(base, project_id, token, tag):
    url = f"{base.rstrip('/')}/api/v4/projects/{project_id}/releases/{urllib.parse.quote(tag, safe='')}"
    status, payload = request("GET", url, {"PRIVATE-TOKEN": token})
    if status == 200 and isinstance(payload, dict):
        notes = (payload.get("description") or "").strip()
        if notes:
            print("Using GitLab release notes")
            return notes
    print(f"No GitLab release notes for {tag} (status {status}), using fallback")
    return None


class GitHub:
    def __init__(self, repo, token):
        self.repo = repo
        self.headers = {
            "Authorization": f"Bearer {token}",
            "Accept": "application/vnd.github+json",
            "X-GitHub-Api-Version": "2022-11-28",
        }

    def get_release(self, tag):
        status, payload = request(
            "GET",
            f"https://api.github.com/repos/{self.repo}/releases/tags/{urllib.parse.quote(tag, safe='')}",
            self.headers,
        )
        return payload if status == 200 else None

    def upsert_release(self, tag, name, body):
        existing = self.get_release(tag)
        data = {"tag_name": tag, "name": name, "body": body, "draft": False, "prerelease": False}
        if existing:
            status, payload = request(
                "PATCH",
                f"https://api.github.com/repos/{self.repo}/releases/{existing['id']}",
                self.headers,
                data,
            )
            action = "updated"
        else:
            status, payload = request(
                "POST",
                f"https://api.github.com/repos/{self.repo}/releases",
                self.headers,
                {**data, "generate_release_notes": False},
            )
            action = "created"
        if status not in (200, 201):
            die(f"GitHub {action} release {status}: {payload}")
        print(f"GitHub release {action}: {payload['html_url']}")
        return payload

    def delete_asset(self, asset_id):
        request(
            "DELETE",
            f"https://api.github.com/repos/{self.repo}/releases/assets/{asset_id}",
            self.headers,
        )

    def upload_asset(self, release, path):
        name = os.path.basename(path)
        for asset in release.get("assets", []):
            if asset["name"] == name:
                self.delete_asset(asset["id"])  # replace on re-run
        with open(path, "rb") as handle:
            blob = handle.read()
        content_type = mimetypes.guess_type(name)[0] or "application/octet-stream"
        upload_url = re.sub(r"\{.*\}$", "", release["upload_url"])
        url = f"{upload_url}?name={urllib.parse.quote(name)}"
        headers = {**self.headers, "Content-Type": content_type}
        status, payload = request("POST", url, headers, raw=blob)
        if status not in (200, 201):
            die(f"GitHub upload {name} {status}: {payload}")
        print(f"  uploaded {name} ({len(blob)} bytes)")


def main():
    tag = env("VERSION", required=True)
    repo = github_repo_from_mirror(env("MIRROR_URL", required=True))
    github = GitHub(repo, env("GITHUB_TOKEN", required=True))

    notes = None
    gl_token = env("GITLAB_TOKEN")
    if gl_token:
        notes = gitlab_notes(env("GITLAB_URL", required=True), env("GITLAB_PROJECT_ID", required=True), gl_token, tag)
    if not notes:
        notes = env("RELEASE_NOTES_FALLBACK") or f"Release {tag}."

    release = github.upsert_release(tag, tag, notes)

    asset_dir = env("ASSET_DIR", default="build/bin")
    if not os.path.isdir(asset_dir):
        die(f"asset directory {asset_dir} not found")
    uploaded = 0
    for name in sorted(os.listdir(asset_dir)):
        path = os.path.join(asset_dir, name)
        if not os.path.isfile(path) or name in SKIP_ASSETS:
            continue
        github.upload_asset(release, path)
        uploaded += 1
    print(f"Done: {uploaded} asset(s) attached to {tag}")

if __name__ == "__main__":
    main()
