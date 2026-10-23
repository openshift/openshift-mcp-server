#!/usr/bin/env bash
set -euo pipefail

task_dir=$(cd "$(dirname "$0")" && pwd)
temp_dir=$(mktemp -d)
trap 'rm -rf "$temp_dir"' EXIT
mkdir -p "$temp_dir/bin" "$temp_dir/task/artifacts"
cp "$task_dir/setup.sh" "$temp_dir/task/"
cp "$task_dir/artifacts/"*.yaml "$temp_dir/task/artifacts/"
cat > "$temp_dir/bin/kubectl" <<'MOCK'
#!/usr/bin/env bash
set -euo pipefail
[[ $KUBECONFIG == /dev/null ]]
[[ " $* " == *' --request-timeout=30s '* ]]
printf '%q ' "$@" >> "$FIXTURE_ARGS"
printf '\n' >> "$FIXTURE_ARGS"
operation=
case "$1 $2" in
    'get storageclasses') operation=classes ;;
    'delete namespace')
        [[ $3 == resize-pv ]]
        operation=delete
        if [[ " $* " == *' --wait=false '* ]]; then operation=cleanup; fi
        ;;
    'create namespace') [[ $3 == resize-pv ]]; operation=create ;;
    'apply -f')
        if [[ $3 == artifacts/storage-pod.yaml ]]; then
            operation=consumer
        else
            operation=pvc
            cp "$3" "$FIXTURE_RENDERED"
        fi
        ;;
    'wait --namespace')
        [[ $3 == resize-pv && " $* " == *' --timeout=120s '* ]]
        if [[ " $* " == *' pvc/storage-pvc '* ]]; then operation=bound; else operation=ready; fi
        ;;
    'get pvc') operation=initial ;;
    *) echo "Unexpected kubectl invocation" >&2; exit 88 ;;
esac
echo "$operation" >> "$FIXTURE_LOG"
if [[ ${FIXTURE_FAIL:-} == "$operation" ]]; then exit 9; fi
if [[ ${FIXTURE_CLEANUP_FAIL:-} == true && $operation == cleanup ]]; then exit 7; fi
case "$operation" in
    classes) cat "$FIXTURE_CLASSES" ;;
    consumer) touch "$FIXTURE_CONSUMER" ;;
    bound)
        if [[ ${FIXTURE_BINDING:-} == WaitForFirstConsumer ]]; then [[ -f $FIXTURE_CONSUMER ]]; fi
        ;;
    initial)
        selected=$(sed -n 's/^  storageClassName: //p' "$FIXTURE_RENDERED")
        jq -n --arg storage_class "$selected" --arg size "${FIXTURE_SIZE:-1Gi}" --arg phase "${FIXTURE_PHASE:-Bound}" '
          {status:{phase:$phase},spec:{storageClassName:$storage_class,accessModes:["ReadWriteOnce"],resources:{requests:{storage:$size}}}}'
        ;;
esac
MOCK
chmod +x "$temp_dir/bin/kubectl"
export PATH="$temp_dir/bin:/usr/bin:/bin" KUBECONFIG=/dev/null
[[ $(command -v kubectl) == "$temp_dir/bin/kubectl" ]]
export FIXTURE_LOG="$temp_dir/commands" FIXTURE_CLASSES="$temp_dir/classes.json"
export FIXTURE_RENDERED="$temp_dir/rendered.yaml" FIXTURE_CONSUMER="$temp_dir/consumer"
export FIXTURE_ARGS="$temp_dir/argv"
passed=0
run_setup() {
    : > "$FIXTURE_LOG"
    : > "$FIXTURE_ARGS"
    rm -f "$FIXTURE_RENDERED" "$FIXTURE_CONSUMER"
    (cd "$temp_dir/task" && bash setup.sh) > "$temp_dir/output" 2>&1
}
record_pass() {
    passed=$((passed + 1))
    printf 'PASS %s\n' "$1"
}
expect_success() {
    local expected_class=$1
    local expected_binding=$2
    if ! run_setup; then cat "$temp_dir/output" >&2; exit 1; fi
    grep -qx "  storageClassName: $expected_class" "$FIXTURE_RENDERED"
    grep -qx '      storage: 1Gi' "$FIXTURE_RENDERED"
    grep -qx '    - ReadWriteOnce' "$FIXTURE_RENDERED"
    local consumer_line bound_line
    consumer_line=$(grep -n '^consumer$' "$FIXTURE_LOG" | cut -d: -f1)
    bound_line=$(grep -n '^bound$' "$FIXTURE_LOG" | cut -d: -f1)
    if [[ $expected_binding == WaitForFirstConsumer ]]; then
        [[ $consumer_line -lt $bound_line ]]
    else
        [[ $bound_line -lt $consumer_line ]]
    fi
    if grep -qx cleanup "$FIXTURE_LOG"; then exit 1; fi
    record_pass "$expected_class/$expected_binding"
}
expect_failure() {
    local description=$1
    local cleanup=$2
    if run_setup; then echo "Unexpected success: $description" >&2; exit 1; fi
    if [[ $cleanup == yes ]]; then
        [[ $(tail -n 1 "$FIXTURE_LOG") == cleanup ]]
    else
        if grep -qx cleanup "$FIXTURE_LOG"; then exit 1; fi
    fi
    record_pass "$description"
}
printf '{"items":[]}' > "$FIXTURE_CLASSES"
expect_failure 'no eligible class' no
[[ $(cat "$FIXTURE_LOG") == classes ]]
grep -q 'No expandable StorageClass available' "$temp_dir/output"
printf '{"items":[{"metadata":{"name":"nonexpandable"},"allowVolumeExpansion":false}]}' > "$FIXTURE_CLASSES"
expect_failure 'nonexpandable class' no
[[ $(cat "$FIXTURE_LOG") == classes ]]
for binding in Immediate WaitForFirstConsumer ''; do
    export FIXTURE_BINDING="$binding"
    jq -n --arg binding "$binding" '{items:[{metadata:{name:"single"},allowVolumeExpansion:true} | if $binding == "" then . else .volumeBindingMode=$binding end]}' > "$FIXTURE_CLASSES"
    expect_success single "${binding:-Immediate}"
done
unset FIXTURE_BINDING
printf '%s' '{"items":[{"metadata":{"name":"z"},"allowVolumeExpansion":true},{"metadata":{"name":"a"},"allowVolumeExpansion":true}]}' > "$FIXTURE_CLASSES"
cp "$FIXTURE_CLASSES" "$temp_dir/multiple.json"
expect_success a Immediate
for annotation in storageclass.kubernetes.io/is-default-class storageclass.beta.kubernetes.io/is-default-class; do
    jq --arg annotation "$annotation" '.items[0].metadata.annotations={($annotation):"true"}' "$temp_dir/multiple.json" > "$FIXTURE_CLASSES"
    expect_success z Immediate
done
jq '.items[].metadata.annotations={"storageclass.kubernetes.io/is-default-class":"true"}' "$temp_dir/multiple.json" > "$FIXTURE_CLASSES"
expect_success a Immediate
for failure in classes delete create pvc consumer bound initial ready; do
    export FIXTURE_FAIL="$failure"
    case "$failure" in classes|delete|create) cleanup=no ;; *) cleanup=yes ;; esac
    expect_failure "kubectl $failure failure" "$cleanup"
done
unset FIXTURE_FAIL
export FIXTURE_FAIL=bound FIXTURE_CLEANUP_FAIL=true
expect_failure 'bounded cleanup failure preserves setup failure' yes
grep -q 'Failed to clean up task namespace' "$temp_dir/output"
unset FIXTURE_FAIL FIXTURE_CLEANUP_FAIL
export FIXTURE_BINDING=WaitForFirstConsumer FIXTURE_FAIL=bound
jq '.items[].volumeBindingMode="WaitForFirstConsumer"' "$temp_dir/multiple.json" > "$FIXTURE_CLASSES"
expect_failure 'WFFC pending timeout' yes
grep -qx consumer "$FIXTURE_LOG"
unset FIXTURE_BINDING FIXTURE_FAIL
cp "$temp_dir/multiple.json" "$FIXTURE_CLASSES"
printf 'not-json' > "$FIXTURE_CLASSES"
expect_failure 'malformed discovery data' no
[[ $(cat "$FIXTURE_LOG") == classes ]]
cp "$temp_dir/multiple.json" "$FIXTURE_CLASSES"
export FIXTURE_SIZE=2Gi
expect_failure 'bound initial request not 1Gi' yes
unset FIXTURE_SIZE
export FIXTURE_PHASE=Pending
expect_failure 'not actually Bound' yes
unset FIXTURE_PHASE
for field in name binding; do
    if [[ $field == name ]]; then
        jq '.items[].metadata.name="bad;$(touch SHOULD_NOT_EXIST)"' "$temp_dir/multiple.json" > "$FIXTURE_CLASSES"
    else
        jq '.items[].volumeBindingMode="Unsupported"' "$temp_dir/multiple.json" > "$FIXTURE_CLASSES"
    fi
    expect_failure "invalid $field" no
    [[ $(cat "$FIXTURE_LOG") == classes ]]
    [[ ! -e $temp_dir/task/SHOULD_NOT_EXIST ]]
done
cp "$temp_dir/multiple.json" "$FIXTURE_CLASSES"
sed 's/storage: 1Gi/storage: 2Gi/' "$task_dir/artifacts/storage-pvc.yaml" > "$temp_dir/task/artifacts/storage-pvc.yaml"
expect_failure 'malformed initial fixture size' no
[[ $(cat "$FIXTURE_LOG") == classes ]]
cp "$task_dir/artifacts/storage-pvc.yaml" "$temp_dir/task/artifacts/storage-pvc.yaml"
printf '      storage: 2Gi\n' >> "$temp_dir/task/artifacts/storage-pvc.yaml"
expect_failure 'duplicate initial size field' no
[[ $(cat "$FIXTURE_LOG") == classes ]]
printf 'PVC setup: %s scenarios passed\n' "$passed"
