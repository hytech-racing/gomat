# gomat

![go construction](https://miro.medium.com/v2/resize:fit:400/1*rAglkfLL1fv8JccizJ3W-Q.png)

Raw MCAP to MATLAB converter written in Go.

It takes all the raw values from an MCAP and writes a MATLAB file with them

To run, enter the devshell:
```
nix develop
```
Once in the devshell, run:
```
src absolute/path/to/mcap/file
```

The output file will be in the root directory.

if you get:

```
<date> could not get mcap mesages
```

then you can try recovering the mcap file with (dont include the `<>` brackets for the names):


```
mcap recover <name-of-faulty-mcap>.mcap -o <new-recovered-name>.mcap
```
