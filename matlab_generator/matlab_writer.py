#!/usr/bin/env python
import sys
from scipy.io import savemat
import json
import argparse
import os

def main():
    # Initialize parser
    parser = argparse.ArgumentParser()

    # Adding optional argument
    parser.add_argument("-p", "--Path", help = "File Path")
    args = parser.parse_args()
    _, tail = os.path.split(args.Path)

    try:
        file_name = tail[0: len(tail) - 5]
        input_data = sys.stdin.read()
        
        data = {"data": json.loads(input_data)}

        # Attempt to save the data as .mat
        savemat(f"./{file_name}.mat", data, long_field_names=True)
        print("MATLAB file created successfully.")

    except json.JSONDecodeError as e:
        print("Error decoding JSON input:", e)
    except Exception as e:
        print("An error occurred:", e)


if __name__ == '__main__':
    main()
