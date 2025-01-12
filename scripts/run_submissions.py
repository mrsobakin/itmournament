import asyncio
import json
import os
from random import shuffle
import httpx
from tqdm import tqdm
from itertools import permutations

ENDPOINT = "http://localhost:4239/run_match"


async def send_request(client: httpx.AsyncClient, lock: asyncio.Lock, semaphore: asyncio.Semaphore, bar: tqdm, master_image_id: str, slave_image_id: str) -> None:
    async with semaphore:
        # try:
        response = await client.post(ENDPOINT, json={
            "master_image_id": master_image_id,
            "slave_image_id": slave_image_id
        })

        data = {
            **response.json(),
            "master_image_id": master_image_id,
            "slave_image_id": slave_image_id,
        }

        async with lock:
            with open("match_results.json", "a") as f:
                f.write(json.dumps(data) + "\n")

        bar.update()

        # except Exception as e:
        #     print(f"Error with master={master_image_id}, slave={slave_image_id}: {e}")


def load_existing_results(results_file: str) -> set:
    if not os.path.exists(results_file):
        return set()

    completed_pairs = set()
    with open(results_file, "r") as f:
        for line in f:
            try:
                result = json.loads(line)
                completed_pairs.add((result["master_image_id"], result["slave_image_id"]))
            except json.JSONDecodeError:
                pass

    return completed_pairs


def load_pending_pairs(build_logs_file: str, results_file: str) -> list:
    with open(build_logs_file, 'r') as f:
        build_logs = json.load(f)

    image_ids = [entry['image_id'] for entry in build_logs if 'image_id' in entry]
    all_pairs = list(permutations(image_ids, 2))

    completed_pairs = load_existing_results(results_file)
    pending_pairs = [pair for pair in all_pairs if pair not in completed_pairs]

    shuffle(pending_pairs)

    return pending_pairs


async def run_matches(pending_pairs: list) -> None:
    lock = asyncio.Lock()
    semaphore = asyncio.Semaphore(75)
    bar = tqdm(total=len(pending_pairs), desc="Running match submissions")

    async with httpx.AsyncClient(timeout=100000) as client:
        tasks = [
            send_request(client, lock, semaphore, bar, master, slave)
            for master, slave in pending_pairs
        ]

        await asyncio.gather(*tasks)


if __name__ == "__main__":
    build_logs_file_path = "build_logs.json"
    results_file_path = "match_results.json"

    if not os.path.exists(results_file_path):
        with open(results_file_path, "w") as f:
            pass

    pending_pairs = load_pending_pairs(build_logs_file_path, results_file_path)
    asyncio.run(run_matches(pending_pairs))

