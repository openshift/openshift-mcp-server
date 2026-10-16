#!/usr/bin/env bash
set -euo pipefail
umask 077

fail() { echo 'ERROR: invalid or unverified Istio bootstrap input' >&2; exit 1; }
[[ $# == 4 ]] || fail
version=$1
destination=$2
addons=$3
manifest=$4
[[ $version == 1.31.0 && -f $manifest && ! -L $manifest ]] || fail
case "$(uname -s)/$(uname -m)" in
    Linux/x86_64) platform=linux-amd64 ;;
    Linux/aarch64|Linux/arm64) platform=linux-arm64 ;;
    Linux/armv7l) platform=linux-armv7 ;;
    Darwin/x86_64) platform=osx-amd64 ;;
    Darwin/arm64) platform=osx-arm64 ;;
    *) fail ;;
esac
asset="istio-$version-$platform.tar.gz"
seen=' '
expected=''
entries=0
while IFS= read -r line || [[ -n $line ]]; do
    [[ $line =~ ^([a-f0-9]{64})[[:space:]]+([^[:space:]]+)$ ]] || fail
    digest=${BASH_REMATCH[1]}
    name=${BASH_REMATCH[2]}
    case $name in
        istio-1.31.0-linux-amd64.tar.gz|istio-1.31.0-linux-arm64.tar.gz|istio-1.31.0-linux-armv7.tar.gz|istio-1.31.0-osx-amd64.tar.gz|istio-1.31.0-osx-arm64.tar.gz) ;;
        *) fail ;;
    esac
    [[ $seen != *" $name "* ]] || fail
    seen+="$name "
    entries=$((entries + 1))
    [[ $name != "$asset" ]] || expected=$digest
done < "$manifest"
[[ $entries == 5 && -n $expected ]] || fail
safe_path() {
    local current=$1
    [[ $current == /* && $current != *$'\n'* && $current != *$'\r'* && $current != */ && $current != *'//'* && $current != */./* && $current != */../* && $current != */. && $current != */.. ]] || fail
    while [[ $current != / ]]; do
        [[ ! -L $current ]] || fail
        current=$(dirname -- "$current")
        [[ ! -e $current || -d $current ]] || fail
    done
}
safe_path "$destination"
safe_path "$addons"
[[ ! -e $destination || -f $destination ]] || fail
[[ ! -e $addons || -d $addons ]] || fail
for name in jaeger prometheus kiali; do
    safe_path "$addons/$name.yaml"
    [[ ! -e $addons/$name.yaml || -f $addons/$name.yaml ]] || fail
done
work=''
binary_temp=''
addon_temp=''
cleanup() {
    [[ -z $binary_temp ]] || rm -f -- "$binary_temp"
    [[ -z $addon_temp ]] || rm -rf -- "$addon_temp"
    [[ -z $work ]] || rm -rf -- "$work"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM
work=$(mktemp -d "${TMPDIR:-/tmp}/istio-install.XXXXXXXX")
[[ -d $work && ! -L $work && -O $work ]] || fail
chmod 0700 "$work"
curl --fail --location --silent --show-error --proto '=https' --tlsv1.2 --connect-timeout 10 --max-time 300 \
    "https://github.com/istio/istio/releases/download/$version/$asset" --output "$work/archive.tar.gz"
if command -v sha256sum >/dev/null 2>&1; then
    actual=$(sha256sum "$work/archive.tar.gz")
else
    actual=$(shasum -a 256 "$work/archive.tar.gz")
fi
[[ ${actual%% *} == "$expected" ]] || fail
prefix="istio-$version"
tar -tzf "$work/archive.tar.gz" > "$work/members"
while IFS= read -r member; do
    [[ $member == "$prefix/"* && $member =~ ^[A-Za-z0-9_./+-]+$ && $member != */../* && $member != */./* && $member != */.. ]] || fail
done < "$work/members"
members=(bin/istioctl samples/addons/jaeger.yaml samples/addons/prometheus.yaml samples/addons/kiali.yaml)
for member in "${members[@]}"; do
    [[ $(grep -Fxc -- "$prefix/$member" "$work/members") == 1 ]] || fail
    details=$(tar -tvzf "$work/archive.tar.gz" -- "$prefix/$member")
    [[ $details == -* && $details != *$'\n'* ]] || fail
    tar -xOzf "$work/archive.tar.gz" -- "$prefix/$member" > "$work/${member##*/}"
    [[ -s $work/${member##*/} ]] || fail
done
sed '/ tracing:/,/ identity:/ {
    s/ enabled: false/ enabled: true\
        in_cluster_url: "http:\/\/tracing.istio-system:16685\/jaeger"\
        use_grpc: true/
}' "$work/kiali.yaml" > "$work/kiali-transformed.yaml"
mv -- "$work/kiali-transformed.yaml" "$work/kiali.yaml"
parent=$(dirname -- "$destination")
mkdir -p -- "$parent" "$addons"
binary_temp=$(mktemp "$parent/.istioctl.XXXXXXXX")
addon_temp=$(mktemp -d "$addons/.istio-addons.XXXXXXXX")
cp -- "$work/istioctl" "$binary_temp"
chmod 0755 "$binary_temp"
for name in jaeger prometheus kiali; do
    cp -- "$work/$name.yaml" "$addon_temp/$name.yaml"
    chmod 0644 "$addon_temp/$name.yaml"
done
mv -- "$binary_temp" "$destination"
binary_temp=''
for name in jaeger prometheus kiali; do
    mv -- "$addon_temp/$name.yaml" "$addons/$name.yaml"
done
