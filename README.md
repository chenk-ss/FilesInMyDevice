# FilesInMyDevice

run
```shell
./build_linux.sh
./filesInMyDevice -P xxx -DP xxx -PATH /root -DOMAIN http://ip
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