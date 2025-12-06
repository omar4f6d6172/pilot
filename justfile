
# Default recipe: build everything and start the VM
default: run

# 1. Compile both binaries for Linux
build:
    @echo "🔨 Building binaries..."
    @mkdir -p bin
    GOOS=linux go build -o bin/pilot ./main.go
    GOOS=linux go build -o bin/user-rest-api ./test/user-rest-api.go

# 2. Boot ephemeral VM with binaries mounted in PATH
run: build
    @echo "🚀 Booting Pilot Environment..."
    @sudo systemd-nspawn \
        -D ./debian-nspawn \
        --ephemeral \
        --boot \
        --bind-ro=`pwd`/bin/pilot:/usr/local/bin/pilot \
        --bind-ro=`pwd`/bin/user-rest-api:/usr/local/bin/user-rest-api


# Open a persistent shell (changes are saved to ./debian-nspawn)
shell:
    @echo "WARNING: You are entering a PERSISTENT shell."
    @echo "Changes made here (apt install, etc) will remain forever."
    @sudo systemd-nspawn -D ./debian-nspawn /bin/bash
