#!/bin/bash

echo "Building gitscribe..."
go build -o gs
echo "Build complete. You can now run gitscribe with the 'gs' command."

# Make the binary executable
chmod +x gs

# Installation options
echo "Installation options:"
echo "1. Install for current user only (~/bin)"
echo "2. Skip installation"
read -p "Choose an option (1-2): " -n 1 -r INSTALL_OPTION
echo

case $INSTALL_OPTION in
    1)
        # Install for current user
        mkdir -p ~/bin
        cp gs ~/bin/
        echo "gitscribe installed in ~/bin/"
        
        # Create config directory and copy default config
        mkdir -p ~/.gitscribe
        if [ ! -f ~/.gitscribe/.gitscribe_config.json ]; then
            cp .gitscribe_config.json ~/.gitscribe/ 2>/dev/null || cp config.json ~/.gitscribe/.gitscribe_config.json
            echo "Default config copied to ~/.gitscribe/.gitscribe_config.json"
        else
            echo "Config file already exists at ~/.gitscribe/.gitscribe_config.json"
        fi
        
        # Add to PATH if not already there
        if [[ ":$PATH:" != *":$HOME/bin:"* ]]; then
            echo 'export PATH="$HOME/bin:$PATH"' >> ~/.bashrc
            echo "Added ~/bin to your PATH in ~/.bashrc"
            echo "Run 'source ~/.bashrc' to update your current session"
            
            # Also add to .zshrc if it exists
            if [ -f ~/.zshrc ]; then
                echo 'export PATH="$HOME/bin:$PATH"' >> ~/.zshrc
                echo "Added ~/bin to your PATH in ~/.zshrc"
                echo "Run 'source ~/.zshrc' to update your current session"
            fi
        fi
        
        echo ""
        echo "Installation complete! You can now run gitscribe with the 'gs' command."
        echo "If 'gs' is not found, you may need to restart your terminal or run:"
        echo "  export PATH=\"\$HOME/bin:\$PATH\""
        ;;
    2)
        echo "Installation skipped. You can run gitscribe from the current directory with './gs'."
        ;;
    *)
        echo "Invalid option. Installation skipped."
        ;;
esac 