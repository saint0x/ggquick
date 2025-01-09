# ggquick

AI-powered PR generator that creates detailed, conversational pull requests from your Git changes.

## Features

- 🤖 Automatically generates PRs when you push any branch
- 🎯 Creates detailed, conversational PR descriptions that explain the changes thoroughly
- 📝 Follows repository contributing guidelines when generating PRs
- 🔄 Handles rate limiting for webhook events
- 🔒 Secure token management for GitHub and OpenAI
- 🌟 Supports all branch types (feature, fix, docs, etc.)
- 🔍 AI-powered analysis of code changes
- 📊 Smart PR title and description generation

## Installation

```bash
# Clone the repository
git clone https://github.com/saint0x/ggquick
cd ggquick

# Build and install
./build.sh
```

## Setup

1. Create a `.env` file in the project root:
```
GITHUB_TOKEN=your_github_token
OPENAI_API_KEY=your_openai_key
```

2. Start listening for changes:
```bash
ggquick start
```

## Usage

1. Create and switch to a new branch:
```bash
git checkout -b your-branch-name
```

2. Make your changes and commit them:
```bash
git add .
git commit -m "your commit message"
```

3. Push your branch:
```bash
git push origin your-branch-name
```

ggquick will automatically:
- Detect your push
- Analyze the changes
- Generate a detailed, conversational PR description
- Create a pull request on GitHub

## Commands

- `ggquick start` - Start listening for Git events
- `ggquick check` - Check server status
- `ggquick stop` - Stop the server

## Environment Variables

- `GITHUB_TOKEN` - Your GitHub personal access token
- `OPENAI_API_KEY` - Your OpenAI API key
- `DEBUG` - Enable debug logging (optional)

## Contributing

1. Fork the repository
2. Create your feature branch
3. Make your changes
4. Push to your branch
5. Create a Pull Request

## License

MIT License - see LICENSE for details 