# ggquick

AI-powered PR generator that creates detailed, conversational pull requests from your Git changes.

## Features

- 🤖 Automatically generates PRs when you push new branches
- 🎯 Creates detailed, conversational PR descriptions that explain the changes thoroughly
- 📝 Follows repository contributing guidelines when generating PRs
- 🔄 Simple and reliable rate limiting
- 🚀 Fast and lightweight implementation

## Installation

1. Clone the repository:
```bash
git clone https://github.com/saint0x/ggquick
cd ggquick
```

2. Set up environment:
```bash
cp .env.example .env
```

3. Edit `.env` and add your credentials:
```env
GITHUB_TOKEN=your_github_token      # GitHub personal access token
OPENAI_API_KEY=your_openai_key     # OpenAI API key
DEBUG=true                         # Optional: Enable debug logging
```

4. Install ggquick:
```bash
./build.sh
```

## Usage

1. Configure a repository:
```bash
ggquick apply https://github.com/user/repo
```
This will:
- Validate your environment
- Configure the repository
- Set up the GitHub webhook
- Verify the connection

2. Start the service:
```bash
ggquick start
```
This will:
- Start the local server
- Connect to the remote server
- Begin processing Git events

3. Check server status:
```bash
ggquick check
```
This will:
- Verify local server health
- Check remote server connection
- Validate webhook configuration

## How it Works

1. When you push changes:
- ggquick detects the push event
- Analyzes your changes
- Generates a detailed PR description
- Creates a pull request automatically

2. The PR will include:
- A clear, descriptive title
- Detailed explanation of changes
- Impact analysis
- Any relevant context

## Commands

- `ggquick apply <repo-url>` - Configure a repository
- `ggquick start` - Start the service
- `ggquick check` - Check server status
- `ggquick stop` - Stop the service

## Environment Variables

- `GITHUB_TOKEN` - GitHub personal access token (required)
- `OPENAI_API_KEY` - OpenAI API key (required)
- `DEBUG` - Enable debug logging (optional)
- `PORT` - Custom port for local server (optional, default: 8080)

## Troubleshooting

1. Check server status:
```bash
ggquick check
```

2. View debug logs:
```bash
DEBUG=true ggquick start
```

3. Verify webhook:
```bash
ggquick check --webhook
```

---

Built for [Theo](https://x.com/t3dotgg) with some of the same design philosophy and opinionations as [ghquick-cli](https://github.com/saint0x/ghquick-cli)