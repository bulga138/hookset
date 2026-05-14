# From lefthook

Migrate your lefthook configuration to hookset.

## How it works

1. hookset reads `lefthook.yml` or `lefthook.yaml`
2. Converts commands with their glob patterns
3. Generates `.hookset.toml` entries

## Limitations

- **Parallelism settings are not migrated** — all hooks run sequentially in hookset v1
- Complex pipeline configurations may need manual adjustment

## Example

Before (`lefthook.yml`):
```yaml
pre-commit:
  commands:
    lint:
      glob: "*.go"
      run: go fmt {files}
    test:
      run: go test ./...
```

After (`hookset migrate --from lefthook --yes`):
```toml
[[hooks]]
name = "lefthook-pre-commit-lint"
event = "pre-commit"
match = ["*.go"]
command = "go fmt"

[[hooks]]
name = "lefthook-pre-commit-test"
event = "pre-commit"
command = "go test ./..."
```

## Notes

- The `{files}` placeholder is expanded at runtime by lefthook; for hookset, commands run directly on matched files
- You can edit the generated manifest to customize behavior