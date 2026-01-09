# Specular CLI Demo Recording
# VHS tape file - https://github.com/charmbracelet/vhs

Output public/demo.gif

Set Shell "zsh"
Set FontFamily "JetBrains Mono"
Set FontSize 14
Set Width 900
Set Height 550
Set Theme "Builtin Dark"
Set Padding 20
Set Framerate 24
Set TypingSpeed 50ms

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
