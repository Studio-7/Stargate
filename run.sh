tmux new -d -s display "X $DISPLAY -config dummy.conf"
sleep 2
tmux new -d -s dosbox "dosbox -conf $GAME/dosbox.conf"
./main
