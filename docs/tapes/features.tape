# Generate feature screenshots - run via: task demo:record:features
# Requires LocalStack running with demo data

Set FontSize 16
Set Width 1920
Set Height 1080
Set TypingSpeed 0.05

Type "./claws"
Enter
Sleep 2s
Require "Services"

Type ":"
Sleep 0.5s
Type "ec2/instances"
Enter
Sleep 2s
Require "Instance ID"
Screenshot docs/images/resource-browser.png

Type "d"
Sleep 2s
Require "Detail"
Screenshot docs/images/detail-view.png
Escape
Sleep 1s

Type "a"
Sleep 1.5s
Require "Stop"
Screenshot docs/images/actions-menu.png
Escape
Sleep 1s

Type "R"
Sleep 1s
Require "Region"
Screenshot docs/images/multi-region.png
Type "/ap-northeast-1"
Sleep 1s
Enter
Sleep 0.5s
Type " "
Sleep 0.5s
Enter
Sleep 3s
Screenshot docs/images/multi-region-data.png

Type "P"
Sleep 1s
Screenshot docs/images/multi-profile.png
Type "/dev"
Sleep 1s
Enter
Sleep 0.5s
Type " "
Sleep 0.5s
Enter
Sleep 1s

Type "R"
Sleep 0.5s
Type "/us-west-2"
Sleep 1s
Enter
Sleep 0.5s
Type " "
Sleep 0.5s
Enter
Sleep 3s
Screenshot docs/images/multi-account-region.png

Type "q"
Sleep 0.5s
