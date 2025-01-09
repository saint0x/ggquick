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
cp .env.example .env

# Add your tokens to .env
# - GitHub token (with repo scope): https://github.com/settings/tokens
# - OpenAI API key: https://platform.openai.com/account/api-keys

# Run the installer
chmod +x build.sh && ./build.sh
```

## Usage

Start the background server:
```bash
ggquick start
```

That's it! The server will:
1. Install git hooks in your repositories
2. Watch for new branch pushes
3. Automatically create PRs with AI-generated content
4. Follow repository contributing guidelines

Stop the server:
```bash
ggquick stop
```

### Debug Mode
```bash
ggquick start --debug  # Verbose output
```

## How it Works

1. When you push a new branch, git hooks notify the background server
2. The server analyzes your changes using AI
3. It fetches the repository's contributing guidelines
4. Creates a well-formatted PR following the guidelines

## Troubleshooting

If the automation isn't working:
1. Check your tokens: `echo $GITHUB_TOKEN && echo $OPENAI_API_KEY`
2. Verify the server is running: `ps aux | grep ggquick`
3. Check git hooks: `ls -la .git/hooks/`
4. Try restarting: `ggquick stop && ggquick start`

## License

MIT - Hack away! 🛠️ 