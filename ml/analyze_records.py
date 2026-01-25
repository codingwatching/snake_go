import os
import glob
from collections import Counter

def analyze_records(records_dir):
    files = glob.glob(os.path.join(records_dir, "*.jsonl"))
    if not files:
        print("No .jsonl files found in", records_dir)
        return

    lengths = []
    for f in files:
        with open(f, 'r') as file:
            count = sum(1 for line in file)
            lengths.append(count)

    lengths.sort()
    
    total_files = len(lengths)
    total_steps = sum(lengths)
    min_len = min(lengths)
    max_len = max(lengths)
    mean_len = total_steps / total_files
    median_len = lengths[total_files // 2]

    print(f"📊 --- Game Records Distribution Analysis ---")
    print(f"Total Files: {total_files}")
    print(f"Total Steps: {total_steps}")
    print(f"Min Length:  {min_len}")
    print(f"Max Length:  {max_len}")
    print(f"Mean Length: {mean_len:.2f}")
    print(f"Median:      {median_len}")
    print("-" * 40)

    # Binning the data
    bins = [
        (0, 50),
        (51, 100),
        (101, 200),
        (201, 500),
        (501, 1000),
        (1001, float('inf'))
    ]
    
    bin_counts = {b: 0 for b in bins}
    for l in lengths:
        for b in bins:
            if b[0] <= l <= b[1]:
                bin_counts[b] += 1
                break

    print("Length Ranges (Steps) | File Count | Percentage")
    print("-" * 40)
    for b in bins:
        label = f"{b[0]}-{b[1]}" if b[1] != float('inf') else f"{b[0]}+"
        count = bin_counts[b]
        pct = (count / total_files) * 100
        bar = "█" * int(pct / 2) # Visualization
        print(f"{label:<20} | {count:>10} | {pct:>6.1f}%  {bar}")

if __name__ == "__main__":
    analyze_records(os.path.abspath("../records"))
