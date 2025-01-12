import sys
from pathlib import Path
import pandas as pd

input_path = Path(sys.argv[1])
output_path = input_path.with_suffix('.csv')

df = pd.read_json(input_path)
df.to_csv(output_path.open(mode='w'), encoding='utf-8', index=False, header=True)
