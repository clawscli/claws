Output docs/images/test-diff.gif

Set Shell "bash"
Set FontSize 16
Set Width 800
Set Height 400
Set Padding 10

Type "./claws"
Enter
Sleep 2s

# Navigate to ec2 instances (has demo data)
Type ":ec2/instances"
Enter
Sleep 2s

# Open command mode and type diff with space
Type ":diff "
Sleep 500ms

Screenshot docs/images/test-diff-before-tab.png

# Press Tab to complete
Tab
Sleep 500ms

Screenshot docs/images/test-diff-after-tab.png

# Press Tab again
Tab
Sleep 500ms

Screenshot docs/images/test-diff-after-tab2.png

Type "q"
Sleep 500ms
