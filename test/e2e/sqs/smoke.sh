#!/usr/bin/env bash

THIS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
ROOT_DIR="$THIS_DIR/../../.."
SCRIPTS_DIR="$ROOT_DIR/scripts"

source "$SCRIPTS_DIR/lib/common.sh"
source "$SCRIPTS_DIR/lib/k8s.sh"
source "$SCRIPTS_DIR/lib/aws/sqs.sh"
source "$SCRIPTS_DIR/lib/testutil.sh"

test_name="$( filenoext "${BASH_SOURCE[0]}" )"
service_name="sqs"
ack_ctrl_pod_id=$( controller_pod_id "sqs")
debug_msg "executing test: $service_name/$test_name"

queue_name="ack-test-smoke-$service_name"
resource_name="queues/$queue_name"

# PRE-CHECKS
if sqs_queue_exists "$queue_name"; then
    echo "FAIL: expected $queue_name to not exist in SQS. Did previous test run cleanup?"
    exit 1
fi

if k8s_resource_exists "$resource_name"; then
    echo "FAIL: expected $resource_name to not exist. Did previous test run cleanup?"
    exit 1
fi

# TEST ACTIONS and ASSERTIONS

cat <<EOF | kubectl apply -f -
apiVersion: sqs.services.k8s.aws/v1alpha1
kind: Queue
metadata:
  name: $queue_name
spec:
  queueName: $queue_name
EOF

sleep 20

debug_msg "checking queue $queue_name created in SQS"
if ! sqs_queue_exists "$queue_name"; then
    echo "FAIL: expected $queue_name to have been created in SQS"
    kubectl logs -n ack-system "$ack_ctrl_pod_id"
    exit 1
fi

kubectl delete "$resource_name" 2>/dev/null
assert_equal "0" "$?" "Expected success from kubectl delete but got $?" || exit 1

# Deletion of queues is not fast for SQS. The queue continues to exist and be
# returned in list operations for many seconds after the AWS DeleteQueue API
# call returned success. So, wait here a bit before asserting that the AWS SQS
# API no longer shows the queue...

sleep 30

if sqs_queue_exists "$queue_name"; then
    echo "FAIL: expected $queue_name to be deleted in SQS"
    kubectl logs -n ack-system "$ack_ctrl_pod_id"
    exit 1
fi
