import asyncio
import csv
import json

import httpx
from tqdm import tqdm


ENDPOINT = "http://localhost:4239/build"


async def send_request(client: httpx.AsyncClient, bar: tqdm, semaphore: asyncio.Semaphore, repo: str, ref: str) -> dict:
    async with semaphore:
        response = await client.post(ENDPOINT, json={
            "repo": repo,
            "ref": ref
        })

        data = {
            **response.json(),
            "repo": repo,
            "ref": ref,
        }

        bar.update()

        return data


async def main(csv_file: str) -> None:
    semaphore = asyncio.Semaphore(4)
    async with httpx.AsyncClient(timeout=180) as client:
        bar = tqdm("Build repos")
        tasks = []
        with open(csv_file, 'r') as file:
            reader = csv.DictReader(file)
            for row in reader:
                repo = row['repo']
                ref = row['ref']
                tasks.append(send_request(client, bar, semaphore, repo, ref))

        bar.total = len(tasks)
        results = await asyncio.gather(*tasks)

        with open("build_logs.json", "w") as f:
            json.dump(results, f)


if __name__ == "__main__":
    csv_file_path = "repos.csv"
    asyncio.run(main(csv_file_path))
