FROM debian:12-slim AS builder

ARG DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y cmake g++ git

# If source if explicitly set, use it. Else, build a github git url.
ARG repo ref src
ADD --keep-git-dir ${src:-https://github.com/$repo.git#$ref} "/var/tournament/repo/"

RUN --network=none <<EOF
    REPO="/var/tournament/repo/"
    ARTIFACT="/var/tournament/artifact.out"

    indent() {
        sed 's/^/>  /'
        echo
    }

    cd "$REPO"

    # Support importing code from archives.
    case $src in
        http://*.tar | https://*.tar)
            tar xf *.tar
    esac
    
    echo 'Applying fixes...'
    find . -type f -name 'CMakeLists.txt' -exec sed -i '
        s/^cmake_minimum_required(.*$/cmake_minimum_required(VERSION 3.12)/
        /^set(CMAKE_CXX_COMPILER/d
        /^set(CMAKE_C_COMPILER/d
        /^FetchContent_Declare(/{s/^/return()\n/}
    ' {} +;

    if [ -d .git ]; then
        if ! git diff --exit-code > /dev/null ; then
            echo Below is the diff for the applied fixes.
            echo If they break your build, prepend spaces before the changed lines and apply fixes manually.
            git diff | indent
        else
            echo No fixes were applied. Nice.
            echo
        fi

        rm -rf .git
    fi

    BUILDDIR=$(mktemp -d --suffix=.build)

    echo Generating the buildsystem...
    cmake "$REPO" -B "$BUILDDIR" 2>&1 | indent

    echo Building the submission...
    cmake --build "$BUILDDIR" 2>&1 | indent

    echo Searching for the executable...
    MAIN_EXECUTABLE=$(find "$BUILDDIR" -type d -name 'CMakeFiles' -prune -o -type f -executable -print | head -n1)

    if [ -z "$MAIN_EXECUTABLE" ]; then
        echo No executable files found. Make sure that your build produces EXACTLY ONE executable file.
        exit 1
    fi

    echo Assuming that \""$MAIN_EXECUTABLE"\" is the main executable file.
    echo If that is not the case, make sure that your build produces EXACTLY ONE executable file.

    echo Exporting executable file...
    cp "$MAIN_EXECUTABLE" "$ARTIFACT"
EOF

FROM debian:12-slim

COPY --from=builder --chmod=755 /var/tournament/artifact.out /opt/tournament-submission

CMD ["/opt/tournament-submission"]
