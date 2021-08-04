# minitime-reader

Small application to parse Go build output w/ `minitime`:

```sh
$ cat<<'EOF' > minitime
#!/bin/sh
command time --format $'%C -> %es\\n' \"$@\"
EOF

$ go build -a -x -work -toolexec=minitime . |& tee /tmp/build-log
...

$ minitime-reader < /tmp/build-log
Longest execution sorted by time:
15m15.47s  | cgo -objdir $WORK/b057/ -importpath github.com/diamondburned/gotk4/pkg/gtk/v4...
3m48.79s   | cgo -objdir $WORK/b055/ -importpath github.com/diamondburned/gotk4/pkg/gio/v2...
...
```
