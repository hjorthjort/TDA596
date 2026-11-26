# Lab 2 · MapReduce

Work in the supplied Go module at `Lab2/6.5840/src`. The three files for the basic assignment are `mr/rpc.go`, `mr/coordinator.go`, and `mr/worker.go`. The course supplies the main programs, plugins, and test script. [Lab handout](instructions.pdf)

Use `git submodule update --init --recursive` to pull the latest changes in the 6.5840 submodule.

## Run the starter

Open two terminals at the repository root. In each, run `cd Lab2/6.5840/src/main`. If your shell cannot find Go on this Mac, run `export PATH="$HOME/sdk/go1.27.3/bin:$PATH"` in each terminal.

In the first terminal, build the plugin and start the coordinator.

```sh
./coordinator.sh
```

In the second terminal, start a worker.

```sh
worker.sh
```

`Lab2/6.5840` is a Git submodule. Use `git -C Lab2/6.5840 status` from the repository root to see your source edits. The parent project's `archive.sh Lab2` omits submodule contents. If you need a ZIP of the committed lab code, run `git -C Lab2/6.5840 archive --format=zip HEAD > lab2-source.zip` from the repository root after committing the work in the submodule.
