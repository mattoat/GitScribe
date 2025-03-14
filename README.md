# GitScribe

GitScribe is a tool that helps you generate commit messages and pull request descriptions using LLMs.

## Features

- Generate commit messages based on staged changes
- Generate pull request descriptions based on commit history
- Create pull requests directly from the command line
- Amend commits with AI-generated messages
- Dry run mode to preview generated messages
- Configurable logging levels for troubleshooting

## Installation

1. Clone the repository:
   ```
   git clone https://github.com/mattoat/gitscribe.git
   cd gitscribe
   ```

2. Build and install:

   ### Option 1: Using the build script (recommended)
   ```
   ./build.sh
   ```
   The build script will:
   - Build the binary named `gs`
   - Offer installation options:
     - Install for current user only (in ~/bin)
     - Skip installation (run from the current directory)
   - Create a ~/.gitscribe directory and copy the default configuration file there
   - Add ~/bin to your PATH in ~/.bashrc if needed

3. The build script will create a binary named `gs`. You can either:
   - Run it from the project directory with `./gs`
   - Install it globally (the build script will prompt you)

## Usage

### Generate a commit message

```
gs
```

This will analyze your staged changes and generate a commit message.

### Amend the last commit with a new message

```
gs -amend
```

This will analyze both the changes in your last commit and any new staged changes, generate a new commit message that encompasses all changes, and then amend the commit. This is useful when you want to:

1. Fix a commit message
2. Add additional changes to the last commit
3. Do both of the above at once

### Generate a pull request description

```
gs -pr
```

This will analyze the commits in your branch and generate a pull request description.

### Additional options

- `-target <branch>`: Specify the target branch for the PR (default: master)
- `-skip-create`: Generate the PR message but don't create the PR on GitHub
- `-config <path>`: Specify a custom path to the configuration file
- `-dry-run`: Generate message but don't commit or create PR
- `-amend`: Amend the last commit with a new AI-generated message (includes both last commit and any staged changes)
- `-log-level <level>`: Set logging level (debug, info, warn, error, none)

## Configuration

GitScribe looks for its configuration file in the following locations (in order of priority):

1. Custom path specified with the `-config` flag
2. `.gitscribe_config.json` in the current working directory
3. `~/.gitscribe/.gitscribe_config.json`
4. In the same directory as the executable

The configuration file allows you to customize:

- Commit message template
- Pull request template
- LLM settings (model, temperature, max tokens, etc.)
- Whether to enable interactive questions for PR generation

## License

[MIT License](LICENSE)