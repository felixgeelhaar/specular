# Specular CLI Demo Recording
# VHS tape file - https://github.com/charmbracelet/vhs
# Styled to match WezTerm Ghostty-like appearance

Output public/demo.gif

Set Shell "zsh"
Set FontFamily "JetBrainsMono Nerd Font"
Set FontSize 15
Set Width 1000
Set Height 600
Set Padding 20
Set Framerate 30
Set TypingSpeed 35ms
Set CursorBlink false

# Ghostty-inspired dark theme colors
Set Theme { "name": "Ghostty Dark", "black": "#000000", "red": "#c75646", "green": "#8eb33b", "yellow": "#d0b03c", "blue": "#72b3cc", "magenta": "#c8a0d1", "cyan": "#218693", "white": "#b0b0b0", "brightBlack": "#5d5d5d", "brightRed": "#e09690", "brightGreen": "#cdee69", "brightYellow": "#ffe377", "brightBlue": "#9cd9f0", "brightMagenta": "#fbb1f9", "brightCyan": "#77dfd8", "brightWhite": "#f7f7f7", "background": "#0c0c0c", "foreground": "#cccccc", "selectionBackground": "#b4d5ff", "cursorColor": "#ffffff" }

# Setup working directory and copy binary
Hide
Type "cd /tmp && rm -rf specular-demo && mkdir specular-demo && cd specular-demo"
Enter
Sleep 500ms
Type "cp /Users/felixgeelhaar/Developer/projects/specular/cli/specular ./specular"
Enter
Sleep 500ms
Type "clear"
Enter
Sleep 300ms
Show

# 1. Initialize project
Type "./specular init --yes --no-detect"
Sleep 200ms
Enter
Sleep 4s

# 2. Show the generated spec file
Type "cat specular.yaml | head -25"
Sleep 200ms
Enter
Sleep 3s

# 3. Run doctor to check system health
Type "./specular doctor"
Sleep 200ms
Enter
Sleep 4s

# 4. Validate policies
Type "./specular policy validate"
Sleep 200ms
Enter
Sleep 3s

# 5. Check for drift
Type "./specular drift check"
Sleep 200ms
Enter
Sleep 3s

Sleep 1s
