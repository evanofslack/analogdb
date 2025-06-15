.PHONY: test
test :
	uv run pytest .

.PHONY: install
install :
	uv pip install -e .

.PHONY: run
run :
	uv run dagster dev

.PHONY: source
source :
	source .venv/bin/activate

.PHONY: spacy
spacy :
	uv run -- spacy download en_core_web_lg
