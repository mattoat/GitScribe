# GitScribe

GitScribe is a tool that helps you generate commit messages and pull request descriptions using LLMs.

## Features

- Generate commit messages based on staged changes
- Generate pull request descriptions based on commit history
- Create pull requests directly from the command line

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
   The build script will offer installation options:
   - Install for current user only (in ~/bin)
   - Skip installation (run from the current directory)

3. The build script will create a binary named `gs`. You can either:
   - Run it from the project directory with `./gs`
   - Install it globally (the build script will prompt you)

## Usage

### Generate a commit message

```
gs
```

This will analyze your staged changes and generate a commit message.

### Generate a pull request description

```
gs -pr
```

This will analyze the commits in your branch and generate a pull request description.

### Additional options

- `-target <branch>`: Specify the target branch for the PR (default: master)
- `-skip-create`: Generate the PR message but don't create the PR on GitHub
- `-config <path>`: Specify a custom path to the configuration file

## Configuration

GitScribe looks for its configuration file in the following locations (in order of priority):

1. Custom path specified with the `-config` flag
2. `.gitscribe_config.json` in the current working directory
3. `~/.gitscribe/.gitscribe_config.json`
5. In the same directory as the executable

The configuration file allows you to customize:

- Commit message template
- Pull request template
- LLM settings (model, temperature, etc.)

## License

[MIT License](LICENSE)