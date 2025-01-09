# ggquick

AI-powered PR generator that creates detailed, conversational pull requests from your Git changes.

## Features

- 🤖 Automatically generates PRs when you push new branches
- 🎯 Creates detailed, conversational PR descriptions that explain the changes thoroughly
- 📝 Follows repository contributing guidelines when generating PRs
- 🔄 Handles rate limiting for webhook events
- 🔒 Secure token management for GitHub and OpenAI

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
git checkout -b feature/my-changes
```

2. Make your changes and commit them:
```bash
git add .
git commit -m "feat: add new feature"
```

3. Push your branch:
```bash
git push origin feature/my-changes
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