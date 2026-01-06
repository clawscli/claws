# Demo recording for claws - run via: task demo:record:gif
# Requires LocalStack running with demo data

Output docs/images/demo.gif

Set FontSize 16
Set Width 1920
Set Height 1080
Set TypingSpeed 0.08

Sleep 0.5s
Type "./claws -p prod,dev -r us-east-1,us-west-2"
Sleep 1.5s
Enter

Sleep 2.5s

Type "s"
Sleep 2s

Type ":"
Sleep 1.5s
Type "vpc/subnets"
Sleep 1.5s
Enter
Sleep 2s

Type "/web"
Sleep 1s
Enter
Sleep 2s

Type "m"
Sleep 1s
Type "G"
Sleep 1s
Type "d"
Sleep 3s
Escape
Sleep 1.5s

Type ":"
Sleep 1.5s
Type "ec2/instances"
Sleep 1.5s
Enter
Sleep 2s

Type "/running"
Sleep 1s
Enter
Sleep 2s

Type "a"
Sleep 2s
Type "S"
Sleep 1.5s
Type "y"
Sleep 2s
Escape
Sleep 1s

Ctrl+r
Sleep 2s

Type "a"
Sleep 2s
Type "D"
Sleep 2s
Type "i-"
Sleep 0.5s
Type "abc"
Sleep 1.5s
Escape
Sleep 1.5s

Type "q"
Sleep 0.5s
