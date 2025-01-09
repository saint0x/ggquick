# ggquick 🚀

AI-powered GitHub PR automation. Automatically creates well-formatted pull requests when you push new branches.

## Features

- 🤖 Automatic PR creation on branch push
- 🎯 AI-generated titles and descriptions
- 📝 Follows repository contributing guidelines
- 🔄 Zero-config git hook integration

## Quick Start

```bash
# Clone and setup
git clone https://github.com/saint0x/ggquick.git
cd ggquick

# Add your GitHub token to the environment
export GITHUB_TOKEN=your_github_token
export GITHUB_REPOSITORY=username/repo

# Run the installer
chmod +x build.sh && ./build.sh
```

## Usage

Start the server:
```bash
ggquick start
```

Check server status:
```bash
ggquick check
```

Stop the server:
```bash
ggquick stop
```

Enable debug logging:
```bash
ggquick start --debug
```

## Environment Variables

- `GITHUB_TOKEN`: Required. GitHub personal access token with repo scope
- `GITHUB_REPOSITORY`: Required. Repository to create PRs in (e.g., username/repo)
- `GGQUICK_PORT`: Optional. Server port (default: 8080)

## How it Works

1. When you push a new branch, git hooks notify the background server
2. The server analyzes your changes using AI
3. It fetches the repository's contributing guidelines
4. Creates a well-formatted PR following the guidelines

## Troubleshooting

1. Check server status:
   ```bash
   ggquick check
   ```

2. If server isn't running:
   ```bash
   ggquick start --debug  # Start with verbose logging
   ```

3. If server is stuck:
   ```bash
   ggquick stop
   ggquick start
   ```

## License

MIT - Hack away! 🛠️ 