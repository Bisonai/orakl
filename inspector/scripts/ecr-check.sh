#!/bin/sh

# Dependencies: curl, jq, yq

# Check the number of tags for a repository
check_repository_tags() {
    repository_name="${1}"
    max_tag_count="${2}"
    service_names="${@:3}"

    # Make the API call
    response=$(curl -s -X POST \
                    -H "Content-Type: application/json" \
                    -d "{\"registryAliasName\": \"bisonai\",\"repositoryName\": \"${repository_name}\"}" \
                    https://api.us-east-1.gallery.ecr.aws/describeImageTags)

    # Count the number of tags
    available_tags=$(echo "$response" | jq '.imageTagDetails |= sort_by(.createdAt)')
    tag_count=$(echo "$available_tags" | jq '.imageTagDetails[].imageTag | select(. != "latest")' | wc -l | tr -d '[:space:]')

    # Check if the tag count exceeds the maximum allowed
    if [ "${tag_count}" -gt "${max_tag_count}" ]; then
        echo "$repository_name: count FAIL (${tag_count} tags)"
    else
        echo "$repository_name: count OK"
    fi

    check_used_tags "cypress"
    check_used_tags "baobab"
}

check_used_tags() {
    chain="${1}"

    for service_name in ${service_names}; do
        helm_chart=$(curl -s --raw "https://raw.githubusercontent.com/Bisonai/orakl-helm-charts/idc-fly/${service_name}/values.${chain}.yaml")
        listener_tag=$(echo "${helm_chart}" | yq eval ".global.image.listenerTag")
        worker_tag=$(echo "${helm_chart}" | yq eval ".global.image.workerTag")
        reporter_tag=$(echo "${helm_chart}" | yq eval ".global.image.reporterTag")
        cur_image_tag=$(echo "${available_tags}" | jq -r ".imageTagDetails[$i].imageTag")
        tag=$(echo "${helm_chart}" | yq eval '.global.image.tag')

        # Sometimes we split tags into listener, worker, and reporter.
        if [ "${tag}" = "null" ]; then
            listener_tag=$(echo "${helm_chart}" | yq eval '.global.image.listenerTag')
            worker_tag=$(echo "${helm_chart}" | yq eval '.global.image.workerTag')
            reporter_tag=$(echo "${helm_chart}" | yq eval '.global.image.reporterTag')

            check_tag ${listener_tag} "listener"
            check_tag ${worker_tag} "worker"
            check_tag ${reporter_tag} "reporter"

            continue
        fi

        check_tag ${tag}
    done
}

check_tag() {
    tag="${1}"
    tag_name="${2}"
    if [ -z "${tag_name}" ]; then
        tag_name=""
    else
        tag_name=", ${tag_name}"
    fi

    found_tag=false
    for i in $(seq 0 $((tag_count))); do
        cur_image_tag=$(echo "${available_tags}" | jq -r ".imageTagDetails[$i].imageTag")
        if [ "${cur_image_tag}" = "${tag}" ]; then
            found_tag=true
        fi
    done

    if [ "${found_tag}" = false ]; then
        echo "(${chain}) ${repository_name}: ${service_name} FAIL (${tag}${tag_name})"
    else
        echo "(${chain}) ${repository_name}: ${service_name} OK (${tag}${tag_name})"
    fi
}

# orakl-general has quite mixed set of tags, so we skip it
# check_repository_tags "orakl-general"
check_repository_tags "orakl-core" 6 "vrf" "request-response" "aggregator"
check_repository_tags "orakl-api" 6 "api"
check_repository_tags "orakl-cli" 6 "cli"
check_repository_tags "orakl-fetcher" 6 "fetcher"
check_repository_tags "orakl-delegator" 6 "delegator"
