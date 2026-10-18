# CLI fixture

Rebuilt by `make cli-fixture` in the Archivist repo. That replays this git history with a fixed author and dates, indexes the tree, and checks `archivist map` against `testdata/cli-fixture/expect`.

After it exists:

```
go run ./cmd/archivist --path testdata/cli-project map seal
```

| Query | What comes back |
| --- | --- |
| `seal` | `Seal`, then `sealNote` |
| `brimstone` | `Seal`, from the second comment line |
| `formatOunces` | `list.js` imports `./web/format.js` |
| `cedarwood` | `print_label`, from the second docstring line |
| `unique-token-alpha` | `weigh_crate` and the alpha commit |
| `unique-token-beta` | the beta commit |
| `wax stamp notes` | `Seal` and the wax-stamp commit |

`jar` matches the `pantry/jar` path, comments that say jar, and the jar commits.
