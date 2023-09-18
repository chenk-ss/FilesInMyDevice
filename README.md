# FilesInMyDevice

run
```shell
./build_linux.sh
./filesInMyDevice -P xxx -PATH /root -URL http://ip:port -DOWNLOAD_URL http://ip:port
```

nginx:
```nginx
server {
  listen       your_DP;
  server_name  filesInMyDevice;

  location / {
        alias  /root/;
        add_header Content-Type text/plain;
  }
}
```