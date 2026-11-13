#!/usr/bin/env bash
set -euo pipefail
umask 077

fail() { echo 'ERROR: invalid candidate artifact or binding' >&2; exit 1; }
[[ ${CANDIDATE_SHA:-} =~ ^[a-f0-9]{40}$ && ${GITHUB_RUN_ID:-} =~ ^[1-9][0-9]{0,15}$ && ${GITHUB_RUN_ATTEMPT:-} =~ ^[1-9][0-9]{0,5}$ ]] || fail
[[ ${SOURCE_REPOSITORY:-} =~ ^[A-Za-z0-9_.-]{1,100}/[A-Za-z0-9_.-]{1,100}$ ]] || fail
[[ ${ARTIFACT_DIRECTORY:-} == /* && $ARTIFACT_DIRECTORY != *$'\n'* && $ARTIFACT_DIRECTORY != *$'\r'* && ! -L $ARTIFACT_DIRECTORY ]] || fail
case ${1:-} in
build)
    [[ ${CANDIDATE_DIRECTORY:-} == /* && -d $CANDIDATE_DIRECTORY && ! -L $CANDIDATE_DIRECTORY && ! -e $ARTIFACT_DIRECTORY ]] || fail
    [[ $(git -C "$CANDIDATE_DIRECTORY" rev-parse HEAD) == "$CANDIDATE_SHA" ]] || fail
    mkdir -m 0700 -- "$ARTIFACT_DIRECTORY"
    cd -- "$CANDIDATE_DIRECTORY"
    package=github.com/containers/kubernetes-mcp-server
    flags="-s -w -X $package/pkg/version.CommitHash=$CANDIDATE_SHA -X $package/pkg/version.Version=$CANDIDATE_SHA -X $package/pkg/version.BuildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ') -X $package/pkg/version.BinaryName=kubernetes-mcp-server -X $package/pkg/version.WebsiteURL=https://github.com/containers/kubernetes-mcp-server"
    GOENV=off GOWORK=off GOTOOLCHAIN=local GOFLAGS='' go build -ldflags "$flags" -o "$ARTIFACT_DIRECTORY/kubernetes-mcp-server" ./cmd/kubernetes-mcp-server
    ;;
verify) ;;
*) fail ;;
esac
[[ -d $ARTIFACT_DIRECTORY && -O $ARTIFACT_DIRECTORY ]] || fail
directory=$(cd -- "$ARTIFACT_DIRECTORY" && pwd -P)
[[ $directory == "$ARTIFACT_DIRECTORY" ]] || fail
binary="$directory/kubernetes-mcp-server"
[[ -f $binary && ! -L $binary && -O $binary && -s $binary ]] || fail
size=$(wc -c < "$binary" | tr -d ' ')
(( size <= 268435456 )) || fail
(( size >= 64 )) || fail
IFS= read -r -N 4 magic < "$binary"
[[ $magic == $'\x7fELF' ]] || fail
digest=$(sha256sum < "$binary"); digest=${digest%% *}
if [[ $1 == build ]]; then
    jq -n --arg candidate "$CANDIDATE_SHA" --arg repository "$SOURCE_REPOSITORY" --arg sha256 "$digest" --argjson size "$size" --argjson run "$GITHUB_RUN_ID" --argjson attempt "$GITHUB_RUN_ATTEMPT" \
        '{version:1,candidate:$candidate,repository:$repository,sha256:$sha256,size:$size,run_id:$run,run_attempt:$attempt,job:"candidate-build"}' > "$directory/metadata.json"
fi
metadata="$directory/metadata.json"
[[ -f $metadata && ! -L $metadata && -O $metadata && $(wc -c < "$metadata") -le 2048 ]] || fail
shopt -s nullglob dotglob
members=("$directory"/*)
(( ${#members[@]} == 2 )) || fail
jq -e --arg candidate "$CANDIDATE_SHA" --arg repository "$SOURCE_REPOSITORY" --arg digest "$digest" --argjson size "$size" --argjson run "$GITHUB_RUN_ID" --argjson attempt "$GITHUB_RUN_ATTEMPT" '
    keys == ["candidate","job","repository","run_attempt","run_id","sha256","size","version"] and
    .version == 1 and .candidate == $candidate and .repository == $repository and .sha256 == $digest and .size == $size and .run_id == $run and .run_attempt == $attempt and .job == "candidate-build"
    ' "$metadata" >/dev/null || fail
if [[ $1 == verify ]]; then
    [[ ${TRUSTED_DIRECTORY:-} == /* && -d $TRUSTED_DIRECTORY && ! -L $TRUSTED_DIRECTORY ]] || fail
    trusted=$(cd -- "$TRUSTED_DIRECTORY" && pwd -P)
    [[ $trusted == "$TRUSTED_DIRECTORY" && -O $trusted && ! -e $trusted/kubernetes-mcp-server && ! -L $trusted/kubernetes-mcp-server ]] || fail
    install -m 0755 -- "$binary" "$trusted/kubernetes-mcp-server"
fi
