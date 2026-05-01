Set Theme { "background": "#ffffff", "foreground": "#333333", "black": "#000000", "white": "#ffffff" }
Set FontSize 16
Set Width 1920
Set Height 1080
Set TypingSpeed 0.05s

Hide
Type "export PS1='$ '; clear"
Enter
Sleep 200ms
Show

Type "./claws -t light"
Enter
Sleep 2s

Type ":"
Sleep 0.5s
Type "ec2/instances"
Enter
Sleep 2s
Require "Instance ID"

Screenshot docs/images/theme-light.png

Type "q"
Sleep 0.5s
