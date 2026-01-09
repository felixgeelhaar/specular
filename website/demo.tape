# Specular CLI Demo Recording
# VHS tape file - https://github.com/charmbracelet/vhs
# Styled to match WezTerm Ghostty-like appearance

Output public/demo.gif

Set Shell "zsh"
Set FontFamily "JetBrainsMono Nerd Font"
Set FontSize 16
Set Width 900
Set Height 500
Set Padding 24
Set Framerate 30
Set TypingSpeed 40ms
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

# Initialize project
Type "./specular init --yes --no-detect"
Sleep 300ms
Enter
Sleep 5s

# Run doctor
Type "./specular doctor"
Sleep 300ms
Enter
Sleep 5s

Sleep 1s
