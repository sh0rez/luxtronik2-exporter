#!/bin/sh

IMAGE="index.docker.io/shorez/luxtronik2-exporter"

if [ "$BUILDKITE_BRANCH" = "master" ] ; then
    DEST=$IMAGE:latest
    echo ":dash: master branch, releasing as $DEST"
elif [[ ! -z "$BUILDKITE_PULL_REQUEST" && "$BUILDKITE_PULL_REQUEST" != "false" ]] ; then
    DEST="$IMAGE:pr-$BUILDKITE_PULL_REQUEST"
    echo ":git: pull-request, pushing as $DEST"
else
    DEST=$IMAGE
    NO_PUSH=1
    echo ":x: regular commit, build only"
fi

set -xe
docker build -t $DEST ${BUILDKITE_GIT_ROOT}
if [ -z "$NO_PUSH" ]; then
    docker push $DEST
fi
