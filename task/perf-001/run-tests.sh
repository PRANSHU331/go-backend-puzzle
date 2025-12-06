set -e
cd "$(dirname "$0")"
pytest -q task_tests.py
