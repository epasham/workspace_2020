#!/bin/bash
trap 'kill $(jobs -p)' EXIT

# check env var
if [ -z "$1" ]; then
  echo please run ./local-development.sh MICROSERVICE_NAME
  exit 1;
fi
MICROSERVICE=$1

# check microservice application dir
if [ ! -d ${MICROSERVICE_DIR} ]; then
  echo cannot find microservice application dir for microservice ${MICROSERVICE} [${MICROSERVICE_DIR}]
  exit 1;
fi
cd $(dirname $0)/../microservices/${MICROSERVICE}

# check microservice settings for skaffold
if [ ! -f ../../development/${MICROSERVICE}/skaffold.yaml ]; then
  echo cannot find skaffold settings for microservice ${MICROSERVICE} [development/${MICROSERVICE}/skaffold.yaml]
  exit 1;
fi

# Setup kubernetes context
if [ -z "${REMOTE_CONTEXT}" ]; then
  echo "Please set REMOTE_CONTEXT env var for remote cluster";
  exit 1;
fi
if [ -z "${LOCAL_CONTEXT}" ]; then
  echo "Please set LOCAL_CONTEXT env var for local cluster";
  exit 1;
fi
if ! kubectl --context ${REMOTE_CONTEXT} version; then
  echo "Context REMOTE_CONTEXT ${REMOTE_CONTEXT} cannot connect."
  exit 1; 
fi
if ! kubectl --context ${LOCAL_CONTEXT} version; then
  echo "Context LOCAL_CONTEXT ${LOCAL_CONTEXT} cannot connect."
  exit 1;
fi

# connect remote kubernetes and local kubernetes by telepresence
TELEPRESENCE_CMD="telepresence \
  --context ${REMOTE_CONTEXT} \
  --namespace ${MICROSERVICE} \
  --swap-deployment ${MICROSERVICE} \
  --expose 8080:80 \
  --run sleep 10000"
echo $TELEPRESENCE_CMD
$TELEPRESENCE_CMD &


# watch files and run applications on local kubernetes by skaffold
SKAFFOLD_CMD="skaffold dev \
  --kube-context ${LOCAL_CONTEXT} \
  --filename ../../development/${MICROSERVICE}/skaffold.yaml \
  --port-forward"

echo $SKAFFOLD_CMD
while true; do
  $SKAFFOLD_CMD
done;

wait

echo Finished development mode


