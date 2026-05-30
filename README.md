# sqlite-gui

A lightweight web GUI for SQLite and PostgreSQL, shipped as a single self-contained binary.

## Installation

```bash
curl -fsSL https://raw.githubusercontent.com/GNITOAHC/sqlite-gui/main/install.sh | bash
```

Or download a pre-built binary from the [Releases](https://github.com/GNITOAHC/sqlite-gui/releases) page (`linux`, `darwin`, `windows` — `amd64`/`arm64`).

**Build from source** (requires Go 1.22+ and Bun):

```bash
git clone https://github.com/GNITOAHC/sqlite-gui.git
cd sqlite-gui && ./build.sh
```

## Usage

```bash
sqlite-gui                                      # opens sqlite-gui.db in the current directory
sqlite-gui -db myapp=file:./myapp.db            # named SQLite file
sqlite-gui -db prod=postgresql://user:pass@host/db  # PostgreSQL
sqlite-gui -db a=file:./a.db -db b=postgresql://... # multiple connections
```

Then open `http://localhost:3000` in your browser. Use `-port` to change the port.
