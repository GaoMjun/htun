FROM scratch

COPY main/htun_linux_x64 /bin/htun
COPY htun.cer .
COPY htun.key .

EXPOSE 19999

ENTRYPOINT ["htun", "client", "-l", ":19999", "-ca", "./htun.cer", "-pk", "./htun.key"]
