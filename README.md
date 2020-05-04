Build the container:
```
$ docker build -t mfs .
```

Run the container:
```
$ docker run -d --rm --device /dev/fuse --privileged mfs
```