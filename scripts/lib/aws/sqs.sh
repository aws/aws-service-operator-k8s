#!/usr/bin/env bash

THIS_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
ROOT_DIR="$THIS_DIR/../../.."
SCRIPTS_DIR="$ROOT_DIR/scripts"

. $SCRIPTS_DIR/lib/common.sh
. $SCRIPTS_DIR/lib/aws.sh

# sqs_queue_exists() returns 0 if an SQS queue with the supplied name exists, 1
# otherwise.
#
# Usage:
#
#   if ! sqs_queue_exists "$queue_name"; then
#       echo "Queue $queue_name does not exist!"
#   fi
sqs_queue_exists() {
    __queue_name="$1"
    daws sqs get-queue-url --queue-name "$__queue_name" --output json >/dev/null 2>&1
    if [[ $? -eq 254 ]]; then
        return 1
    else
        return 0
    fi
}
