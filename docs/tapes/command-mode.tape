# Command Mode Suggestion & Completion Test
# Tests command mode navigation, suggestions, tab completion, and enter behavior

Set FontSize 16
Set Width 1200
Set Height 600
Set TypingSpeed 0.08

Type "./claws"
Enter
Sleep 2s
Require "Services"

# --- Suggestion Tests ---

# ec → ec2 | ecr | ecs
Type ":"
Sleep 300ms
Type "ec"
Sleep 500ms
Require "ec2"
Require "ecr"
Require "ecs"
Screenshot docs/images/cmd-suggest-ec.png

# ec2 → ec2 (resolved)
Type "2"
Sleep 500ms
Screenshot docs/images/cmd-suggest-ec2.png

# ec2/ → ec2 | ec2/capacity-reservations | ...
Type "/"
Sleep 500ms
Require "ec2/capacity"
Screenshot docs/images/cmd-suggest-ec2-slash.png

# ec2/c → ec2/capacity-reservations
Type "c"
Sleep 500ms
Require "capacity-reservations"
Screenshot docs/images/cmd-suggest-ec2-c.png

Escape
Sleep 300ms

# --- Enter Navigation Tests ---

# ec Enter → ec2 service
Type ":ec"
Sleep 300ms
Enter
Sleep 2s
Require "ec2"

# ec2/ Enter → ec2 service
Type ":ec2/"
Sleep 300ms
Enter
Sleep 2s
Require "ec2"

# ec2/c Enter → ec2/capacity-reservations
Type ":ec2/c"
Sleep 300ms
Enter
Sleep 2s
Require "Capacity"

# --- Tab Completion Test ---

# ec2/c Tab → completes to ec2/capacity-reservations
Type ":"
Sleep 300ms
Type "ec2/c"
Sleep 300ms
Tab
Sleep 500ms
Require "ec2/capacity-reservations"
Screenshot docs/images/cmd-tab-completion.png
Escape
Sleep 300ms

Type "q"
Sleep 500ms
