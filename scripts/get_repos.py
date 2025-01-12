import csv
import os
import time

import requests
from tqdm import tqdm

GITHUB_TOKEN = os.environ.get("GIT_AUTH_TOKEN")
ORGANIZATION = "is-itmo-c-24"

GITHUB_API_URL = "https://api.github.com"
HEADERS = {
    "Authorization": f"Bearer {GITHUB_TOKEN}",
    "Accept": "application/vnd.github+json"
}


def make_github_request(endpoint, params=None):
    url = f"{GITHUB_API_URL}/{endpoint.lstrip('/')}"
    while True:
        response = requests.get(url, headers=HEADERS, params=params)
        if response.status_code == 403:
            remaining = int(response.headers.get("X-RateLimit-Remaining", 1))
            if remaining == 0:
                reset_time = int(response.headers.get("X-RateLimit-Reset", time.time()))
                sleep_time = reset_time - int(time.time())
                time.sleep(max(sleep_time, 0))
            continue
        return response


def get_repositories(org):
    repositories = []
    page = 1

    bar = tqdm(desc="Getting all repositories")
    while True:
        bar.update()
        response = make_github_request(f"/orgs/{org}/repos", params={"per_page": 100, "page": page})
        data = response.json()
        if not data:
            break
        repositories.extend(data)
        page += 1
    bar.close()

    return repositories


def get_branch_ref(repo_name, branch_name):
    response = make_github_request(f"/repos/{ORGANIZATION}/{repo_name}/branches/{branch_name}")
    branch_data = response.json()

    if response.status_code == 404:
        return None

    return branch_data.get("commit", {}).get("sha")


def main():
    repositories = get_repositories(ORGANIZATION)

    repositories = list(filter(lambda r: r["name"].startswith("labwork5-"), repositories))

    repos_with_branch = []
    for repo in tqdm(repositories, desc="Filtering repositories"):
        repo_name = repo["name"]

        branch_ref = get_branch_ref(repo_name, "tournament")
        if branch_ref:
            repos_with_branch.append({"repo": f"{ORGANIZATION}/{repo_name}", "ref": branch_ref})

    with open("repos.csv", mode="w", newline="", encoding="utf-8") as file:
        writer = csv.DictWriter(file, fieldnames=["repo", "ref"])
        writer.writeheader()
        writer.writerows(repos_with_branch)


if __name__ == "__main__":
    main()
