#!/usr/bin/env python3
"""
Convert historical CSV data to the required format for backtesting
"""

import csv
import sys
import os
from datetime import datetime

def convert_csv(input_file, output_file):
    """Convert CSV from the GitHub source to required format"""

    try:
        with open(input_file, 'r') as infile, open(output_file, 'w', newline='') as outfile:
            reader = csv.reader(infile)
            writer = csv.writer(outfile)

            # Write the required header
            writer.writerow(['timestamp', 'open', 'high', 'low', 'close', 'volume'])

            # Skip header in input
            header = next(reader)
            print(f"Input header: {header}")

            count = 0
            for row in reader:
                if len(row) < 7:  # Need at least 7 columns
                    continue

                # Extract data (ignore Trades column)
                timestamp_str = row[0]  # Time UTC
                open_price = row[1]     # Open
                high_price = row[2]     # High
                low_price = row[3]      # Low
                close_price = row[4]    # Close
                volume = row[5]        # Volume

                # Convert timestamp format from "2020-01-01T00:00:00.000Z" to "2020-01-01 00:00:00"
                try:
                    # Parse the timestamp and convert to required format
                    dt = datetime.strptime(timestamp_str, "%Y-%m-%dT%H:%M:%S.%fZ")
                    formatted_timestamp = dt.strftime("%Y-%m-%d %H:%M:%S")

                    # Write the converted row
                    writer.writerow([formatted_timestamp, open_price, high_price, low_price, close_price, volume])
                    count += 1

                except ValueError as e:
                    print(f"Error parsing timestamp '{timestamp_str}': {e}")
                    continue

            print(f"✅ Converted {count} data points")
            return count

    except FileNotFoundError:
        print(f"❌ Input file not found: {input_file}")
        return 0
    except Exception as e:
        print(f"❌ Error converting CSV: {e}")
        return 0

if __name__ == "__main__":
    input_file = "./data/temp_raw.csv"
    output_file = "./data/BTCUSDT.csv"

    print("🔄 Converting historical data...")
    count = convert_csv(input_file, output_file)

    if count > 0:
        print(f"✅ Conversion successful! {count} rows written to {output_file}")

        # Show sample of converted data
        print("\n📋 Sample converted data:")
        with open(output_file, 'r') as f:
            for i, line in enumerate(f):
                if i < 6:  # Show first 6 lines (header + 5 data rows)
                    print(line.strip())
                else:
                    break
    else:
        print("❌ Conversion failed")