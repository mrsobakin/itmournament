import json
import os
import time

import requests
from tqdm import tqdm

GITHUB_TOKEN = os.environ.get("GIT_AUTH_TOKEN")


def post_or_comment_github_issue(repo, title, body, username="mrsobakin"):
    """
    Creates a new GitHub issue or comments on an existing one by the user.

    Parameters:
    - repo: str - The target repository in the format "owner/repo".
    - title: str - Title of the issue to create or check.
    - body: str - Body of the new issue (if created).
    - username: str - Username of the issue creator to check for (default: "mrsobakin").

    Returns:
    - dict: Response JSON from GitHub API.
    """
    headers = {
        "Authorization": f"Bearer {GITHUB_TOKEN}",
        "Accept": "application/vnd.github+json",
    }

    # Step 1: Search for existing issues with the title
    search_url = f"https://api.github.com/repos/{repo}/issues"
    params = {"state": "open", "per_page": 100}  # Adjust per_page if needed
    response = requests.get(search_url, headers=headers, params=params)

    if response.status_code != 200:
        raise Exception(f"Failed to fetch issues: {response.status_code}, {response.text}")

    issues = response.json()
    for issue in issues:
        if issue["title"] == title and issue["user"]["login"] == username:
            # Issue exists, post a comment
            comment_url = issue["comments_url"]
            comment_response = requests.post(
                comment_url, headers=headers, json={"body": body}
            )
            if comment_response.status_code != 201:
                raise Exception(f"Failed to comment on issue: {comment_response.status_code}, {comment_response.text}")
            return {"action": "commented", "issue_url": issue["html_url"], "response": comment_response.json()}

    # Step 2: No matching issue found, create a new issue
    create_issue_url = f"https://api.github.com/repos/{repo}/issues"
    issue_data = {"title": title, "body": body}
    create_response = requests.post(create_issue_url, headers=headers, json=issue_data)

    if create_response.status_code != 201:
        raise Exception(f"Failed to create issue: {create_response.status_code}, {create_response.text}")
    return {"action": "created", "issue_url": create_response.json()["html_url"], "response": create_response.json()}


if __name__ == "__main__":
    with open("build_logs.json") as f:
        data = json.load(f)

    for d in tqdm(data):
        user = d["repo"].split("/")[1].split("-", 1)[1]
        ref = d["ref"]
        logs = d["logs"]

        if "image_id" in d:
            msg = f"Hi again @{user}.\n\nThis is an update on build status of your submission. As of commit `{ref}`, it builds successfully. Good job 👍.\n\nThis is a last call to make any changes to your submission. Any changes made after `23:59:59 MSK` will be ignored.\n\nGood luck!"
        else:
            msg = f"Hi again @{user}.\n\nThis is an update on build status of your submission. As of commit `{ref}`, it fails to build. Simplified build log:\n\n```\n{logs}```\n\nThis is a last call to fix your submission or make any other changes to it. Any changes made after `23:59:59 MSK` will be ignored.\n\nGood luck!"

        post_or_comment_github_issue(d["repo"], "Battleship tournament", msg, "mrsobakin")
        time.sleep(7)
